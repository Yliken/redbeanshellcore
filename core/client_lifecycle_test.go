package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

func TestClientMiddlewareObservesFinalHTTPError(t *testing.T) {
	var observed error
	var status int
	observer := func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			resp, err := next(ctx, req)
			observed = err
			if resp != nil {
				status = resp.StatusCode
			}
			return resp, err
		}
	}
	client := core.NewClient(
		core.WithTransport(transportmock.New(func(context.Context, *core.Request) (*core.Response, error) {
			resp := core.NewResponse()
			resp.StatusCode = 500
			return resp, nil
		})),
		core.WithMiddleware(observer),
	)
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := client.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrRemoteRuntime) || !core.IsKind(observed, core.ErrRemoteRuntime) {
		t.Fatalf("Client 和 middleware 都应观察 ErrRemoteRuntime: client=%v middleware=%v", err, observed)
	}
	if status != 500 {
		t.Fatalf("middleware 应保留 HTTP status，got %d", status)
	}
}

func TestClientMiddlewareObservesParseError(t *testing.T) {
	var observed error
	observer := func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			resp, err := next(ctx, req)
			observed = err
			return resp, err
		}
	}
	client := core.NewClient(
		core.WithTransport(transportmock.New(transportmock.StaticHandler([]byte("bad")))),
		core.WithMiddleware(observer),
	)
	op := &stubOp{
		name:  "bad",
		build: func() (*core.Request, error) { return core.NewRequest("bad"), nil },
		parse: func(*core.Response) (core.Result, error) { return nil, errors.New("parse failed") },
	}

	_, err := client.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrParse) || !core.IsKind(observed, core.ErrParse) {
		t.Fatalf("Client 和 middleware 都应观察 ErrParse: client=%v middleware=%v", err, observed)
	}
}

func TestClientPropagatesOnlyMarkerMetadata(t *testing.T) {
	var responseMeta map[string]string
	client := core.NewClient(
		core.WithSession(&core.Session{Metadata: map[string]string{"auth_password_field": "secret"}}),
		core.WithTransport(transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
			return &core.Response{}, nil
		})),
	)
	op := &stubOp{
		name: "meta",
		build: func() (*core.Request, error) {
			req := &core.Request{Operation: "meta"}
			req.SetMeta("marker.tag_s", "start")
			return req, nil
		},
		parse: func(resp *core.Response) (core.Result, error) {
			responseMeta = resp.Meta
			return &core.BoolResult{BaseResult: core.NewBaseResult("meta", nil), OK: true}, nil
		},
	}

	if _, err := client.Do(context.Background(), op); err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if responseMeta["marker.tag_s"] != "start" {
		t.Fatalf("marker metadata 未传播: %#v", responseMeta)
	}
	if _, exists := responseMeta["auth_password_field"]; exists {
		t.Fatalf("认证 metadata 不应传播到响应: %#v", responseMeta)
	}
}

func TestClientRejectsNilContracts(t *testing.T) {
	t.Run("operation", func(t *testing.T) {
		client := core.NewClient(core.WithTransport(transportmock.New(transportmock.EchoHandler)))
		_, err := client.Do(context.Background(), nil)
		if !core.IsKind(err, core.ErrProtocol) {
			t.Fatalf("nil operation 应为 ErrProtocol，got %v", err)
		}
	})

	t.Run("transport response", func(t *testing.T) {
		client := core.NewClient(core.WithTransport(transportmock.New(func(context.Context, *core.Request) (*core.Response, error) {
			return nil, nil
		})))
		op := &stubOp{name: "nil-response", build: func() (*core.Request, error) { return core.NewRequest("nil-response"), nil }}
		_, err := client.Do(context.Background(), op)
		if !core.IsKind(err, core.ErrProtocol) {
			t.Fatalf("nil response 应为 ErrProtocol，got %v", err)
		}
	})

	t.Run("parse result", func(t *testing.T) {
		client := core.NewClient(core.WithTransport(transportmock.New(transportmock.EchoHandler)))
		op := &stubOp{
			name:  "nil-result",
			build: func() (*core.Request, error) { return core.NewRequest("nil-result"), nil },
			parse: func(*core.Response) (core.Result, error) { return nil, nil },
		}
		_, err := client.Do(context.Background(), op)
		if !core.IsKind(err, core.ErrParse) {
			t.Fatalf("nil parse result 应为 ErrParse，got %v", err)
		}
	})
}

func TestClientEnrichesOpErrorWithoutMutation(t *testing.T) {
	original := &core.OpError{Kind: core.ErrNetwork, Message: "network failed"}
	client := core.NewClient(
		core.WithSession(&core.Session{NodeID: "node-1"}),
		core.WithTransport(transportmock.New(func(context.Context, *core.Request) (*core.Response, error) {
			return nil, original
		})),
	)
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := client.Do(context.Background(), op)
	var got *core.OpError
	if !errors.As(err, &got) {
		t.Fatalf("期望 OpError，got %v", err)
	}
	if got.Operation != "info" || got.NodeID != "node-1" {
		t.Fatalf("错误上下文未补齐: %+v", got)
	}
	if original.Operation != "" || original.NodeID != "" {
		t.Fatalf("原错误不应被修改: %+v", original)
	}
}

func TestClientNormalizesDeadlineError(t *testing.T) {
	client := core.NewClient(core.WithTransport(transportmock.New(func(context.Context, *core.Request) (*core.Response, error) {
		return nil, context.DeadlineExceeded
	})))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := client.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline 应归一化并保留 cause，got %v", err)
	}
}
