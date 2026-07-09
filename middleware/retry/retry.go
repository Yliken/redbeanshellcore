// Package retry 提供带退避的重试中间件。
//  依据设计文档，只对幂等读操作重试；破坏性 / 写入类操作直接跳过。
package retry

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/yliken/redbeanshellcore/core"
)

// Options 调整重试行为。
type Options struct {
	MaxAttempts  int             // 总尝试次数（默认 3）
	Backoff      time.Duration   // 基础退避（默认 200ms）
	MaxBackoff   time.Duration   // 最大退避（默认 5s）
	Retryable    func(op string) bool             // 自定义重试判定（nil 走内置读白名单）
	ShouldRetry  func(op string, err error) bool  // 自定义是否重试某次错误
	Logger       *slog.Logger                     // 重试时打日志
}

// Middleware 返回带重试能力的中间件。
func Middleware(opts Options) core.Middleware {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.Backoff <= 0 {
		opts.Backoff = 200 * time.Millisecond
	}
	if opts.MaxBackoff < 0 {
		opts.MaxBackoff = 5 * time.Second
	}
	if opts.Retryable == nil {
		opts.Retryable = defaultRetryable
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
			for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
				resp, err := next(ctx, req)
				if err == nil {
					return resp, nil
				}
				lastErr = err
				if opts.ShouldRetry != nil && !opts.ShouldRetry(req.Operation, err) {
					return resp, err
				}
				if attempt == opts.MaxAttempts {
					break
				}
				wait := opts.Backoff << (attempt - 1)
				if wait > opts.MaxBackoff {
					wait = opts.MaxBackoff
				}
				opts.Logger.Info("remote_node_retry",
					"operation", req.Operation,
					"node", req.NodeID,
					"attempt", attempt,
					"backoff_ms", wait.Milliseconds(),
					"error", err.Error(),
				)
				t := time.NewTimer(wait)
				select {
				case <-ctx.Done():
					t.Stop()
					return nil, ctx.Err()
				case <-t.C:
				}
			}
			return nil, lastErr
		}
	}
}

// defaultRetryable 仅允许内置读操作重试。
func defaultRetryable(op string) bool {
	switch op {
	case "info", "file.list", "file.read", "file.download":
		return true
	}
	return false
}
