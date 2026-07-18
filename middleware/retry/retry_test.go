package retry

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestDefaultRetriesNetworkErrorWithBackoff(t *testing.T) {
	attempts := 0
	handler := Middleware(Options{
		MaxAttempts: 2,
		Backoff:     10 * time.Millisecond,
		Logger:      discardLogger(),
	})(func(context.Context, *core.Request) (*core.Response, error) {
		attempts++
		return nil, core.NewOpError(core.ErrNetwork, "info", "node", "temporary", nil)
	})
	start := time.Now()
	_, err := handler(context.Background(), core.NewRequest("info"))
	if err == nil || attempts != 2 {
		t.Fatalf("网络错误应重试: attempts=%d err=%v", attempts, err)
	}
	if time.Since(start) < 8*time.Millisecond {
		t.Fatalf("零值 MaxBackoff 不应把退避截断为 0")
	}
}

func TestDefaultDoesNotRetryParseError(t *testing.T) {
	attempts := 0
	handler := Middleware(Options{MaxAttempts: 3, Backoff: time.Millisecond, Logger: discardLogger()})(func(context.Context, *core.Request) (*core.Response, error) {
		attempts++
		return nil, core.NewOpError(core.ErrParse, "info", "node", "permanent", nil)
	})
	_, _ = handler(context.Background(), core.NewRequest("info"))
	if attempts != 1 {
		t.Fatalf("解析错误不应默认重试，got %d", attempts)
	}
}

func TestRetryPreservesLastResponse(t *testing.T) {
	attempts := 0
	handler := Middleware(Options{MaxAttempts: 2, Backoff: time.Millisecond, Logger: discardLogger()})(func(context.Context, *core.Request) (*core.Response, error) {
		attempts++
		resp := core.NewResponse()
		resp.StatusCode = 502
		return resp, core.NewOpError(core.ErrNetwork, "info", "node", "temporary", nil)
	})
	resp, err := handler(context.Background(), core.NewRequest("info"))
	if err == nil || resp == nil || resp.StatusCode != 502 || attempts != 2 {
		t.Fatalf("应保留最后响应: resp=%+v attempts=%d err=%v", resp, attempts, err)
	}
}

func TestCustomShouldRetryOverridesDefault(t *testing.T) {
	attempts := 0
	handler := Middleware(Options{
		MaxAttempts: 2,
		Backoff:     time.Millisecond,
		Logger:      discardLogger(),
		ShouldRetry: func(string, error) bool { return true },
	})(func(context.Context, *core.Request) (*core.Response, error) {
		attempts++
		return nil, core.NewOpError(core.ErrParse, "info", "node", "forced", nil)
	})
	_, _ = handler(context.Background(), core.NewRequest("info"))
	if attempts != 2 {
		t.Fatalf("自定义 ShouldRetry 应完全覆盖默认策略，got %d", attempts)
	}
}
