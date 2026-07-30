package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

// stubOp 是测试专用的 operation。
type stubOp struct {
	name  string
	build func() (*core.Request, error)
	parse func(*core.Response) (core.Result, error)
}

func (s *stubOp) Name() string { return s.name }
func (s *stubOp) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	return s.build()
}
func (s *stubOp) Parse(_ context.Context, r *core.Response) (core.Result, error) {
	if s.parse != nil {
		return s.parse(r)
	}
	return &core.BoolResult{BaseResult: core.NewBaseResult(s.name, r.Body), OK: true}, nil
}

func TestClientDoSuccess(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.Body = req.Payload
		return resp, nil
	})
	c := core.NewClient(
		core.WithSession(&core.Session{NodeID: "n1", Endpoint: "http://x"}),
		core.WithTransport(tr),
	)

	op := &stubOp{
		name: "echo",
		build: func() (*core.Request, error) {
			r := core.NewRequest("echo")
			r.Payload = []byte("hello")
			return r, nil
		},
	}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("不应出错: %v", err)
	}
	if string(res.Raw()) != "hello" {
		t.Fatalf("raw 不符合预期: %q", string(res.Raw()))
	}
}

func TestClientDoOperationBuildError(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{
		name:  "broken",
		build: func() (*core.Request, error) { return nil, errors.New("boom") },
	}
	_, err := c.Do(context.Background(), op)
	if err == nil {
		t.Fatalf("应返回错误")
	}
	if !core.IsKind(err, core.ErrProtocol) {
		t.Fatalf("应返回 ErrProtocol，实际返回 %T: %v", err, err)
	}
}

func TestClientDoTransportError(t *testing.T) {
	sentinel := errors.New("i/o timeout")
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return nil, sentinel
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{
		name:  "noop",
		build: func() (*core.Request, error) { return core.NewRequest("noop"), nil },
	}
	_, err := c.Do(context.Background(), op)
	if !errors.Is(err, sentinel) {
		t.Fatalf("应透传底层错误，实际返回 %v", err)
	}
}
