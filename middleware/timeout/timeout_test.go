package timeout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestMiddlewareNormalizesDeadline(t *testing.T) {
	handler := Middleware(Options{Timeout: 5 * time.Millisecond})(func(ctx context.Context, _ *core.Request) (*core.Response, error) {
		<-ctx.Done()
		return core.NewResponse(), ctx.Err()
	})
	req := core.NewRequest("info")
	req.NodeID = "node-1"

	resp, err := handler(context.Background(), req)
	if resp == nil || !core.IsKind(err, core.ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("应保留响应并归一化超时: resp=%v err=%v", resp, err)
	}
}

func TestMiddlewareRespectsExistingShorterDeadline(t *testing.T) {
	parent, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	start := time.Now()
	handler := Middleware(Options{Timeout: time.Second})(func(ctx context.Context, _ *core.Request) (*core.Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	_, err := handler(parent, core.NewRequest("info"))
	if !core.IsKind(err, core.ErrTimeout) {
		t.Fatalf("已有 deadline 应被保留并归一化: %v", err)
	}
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("middleware 不应延长调用方 deadline")
	}
}

func TestMiddlewareDoesNotRelabelCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	handler := Middleware(Options{Timeout: time.Second})(func(ctx context.Context, _ *core.Request) (*core.Response, error) {
		return nil, ctx.Err()
	})

	_, err := handler(ctx, core.NewRequest("info"))
	if core.IsKind(err, core.ErrTimeout) || !errors.Is(err, context.Canceled) {
		t.Fatalf("context.Canceled 不应标记为 timeout: %v", err)
	}
}
