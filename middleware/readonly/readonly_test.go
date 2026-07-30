package readonly

import (
	"context"
	"errors"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/transport/mock"
)

func TestReadonlyBlocksExec(t *testing.T) {
	var called bool
	root := func(_ context.Context, _ *core.Request) (*core.Response, error) {
		called = true
		return core.NewResponse(), nil
	}
	mw := Middleware()
	next := mw(root)

	// file.write 应被拦截
	req := core.NewRequest("file.write")
	req.NodeID = "n1"
	_, err := next(context.Background(), req)
	if err == nil {
		t.Fatalf("应返回 policy_denied")
	}
	if !core.IsKind(err, core.ErrPolicyDenied) {
		t.Fatalf("应返回 ErrPolicyDenied，实际返回 %v", err)
	}
	if called {
		t.Fatalf("handler 不应被调用")
	}

	// info 应放行
	req2 := core.NewRequest("info")
	req2.NodeID = "n1"
	if _, err := next(context.Background(), req2); err != nil {
		t.Fatalf("不应出错: %v", err)
	}
	if !called {
		t.Fatalf("handler 应被调用")
	}
}

func TestMockTransportSatisfiesInterface(t *testing.T) {
	_ = mock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return nil, errors.New("仅做编译期接口检查")
	})
}
