// Package retry 提供带退避的重试中间件。
// 默认仅对幂等读操作的网络错误和超时重试。
package retry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// Options 调整重试行为。
type Options struct {
	MaxAttempts int           // 总尝试次数（默认 3）
	Backoff     time.Duration // 基础退避（默认 200ms）
	MaxBackoff  time.Duration // 最大退避（默认 5s）
	Retryable   func(op string) bool
	ShouldRetry func(op string, err error) bool
	Logger      *slog.Logger
}

// Middleware 返回带重试能力的中间件。
func Middleware(opts Options) core.Middleware {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.Backoff <= 0 {
		opts.Backoff = 200 * time.Millisecond
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 5 * time.Second
	}
	if opts.Retryable == nil {
		opts.Retryable = defaultRetryable
	}
	if opts.ShouldRetry == nil {
		opts.ShouldRetry = defaultShouldRetry
	}
	if opts.Logger == nil {
		opts.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			if !opts.Retryable(req.Operation) {
				return next(ctx, req)
			}
			var lastErr error
			var lastResp *core.Response
			for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
				resp, err := next(ctx, req)
				if resp != nil {
					lastResp = resp
				}
				if err == nil {
					return resp, nil
				}
				lastErr = err
				if !opts.ShouldRetry(req.Operation, err) {
					return resp, err
				}
				if attempt == opts.MaxAttempts {
					break
				}
				wait := backoffForAttempt(opts.Backoff, opts.MaxBackoff, attempt)
				opts.Logger.Info("remote_node_retry",
					"operation", req.Operation,
					"node", req.NodeID,
					"attempt", attempt,
					"backoff_ms", wait.Milliseconds(),
					"error", err.Error(),
				)
				timer := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return lastResp, ctx.Err()
				case <-timer.C:
				}
			}
			return lastResp, lastErr
		}
	}
}

func defaultShouldRetry(_ string, err error) bool {
	return core.IsKind(err, core.ErrNetwork) || core.IsKind(err, core.ErrTimeout)
}

func backoffForAttempt(base, maximum time.Duration, attempt int) time.Duration {
	wait := base
	for i := 1; i < attempt; i++ {
		if wait >= maximum || wait > maximum/2 {
			return maximum
		}
		wait *= 2
	}
	if wait > maximum {
		return maximum
	}
	return wait
}

func defaultRetryable(op string) bool {
	switch op {
	case "info", "file.list", "file.read", "file.download":
		return true
	}
	return false
}
