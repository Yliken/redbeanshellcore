package core_test

import (
	"context"
	stdbase64 "encoding/base64"
	"errors"
	"testing"

	codec "github.com/Yliken/redbeanshellcore/codec/base64"
	"github.com/Yliken/redbeanshellcore/core"
	envelopemarker "github.com/Yliken/redbeanshellcore/envelope/marker"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

// riskAwareStub 是带 RiskAware 接口的 stub operation。
type riskAwareStub struct {
	stubOp
	risk core.RiskLevel
}

func (r *riskAwareStub) RiskLevel() core.RiskLevel { return r.risk }

// captureTransport 把 req.Meta 暴露给测试断言。
func captureTransport(captured *map[string]string) *transportmock.Transport {
	return transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
		*captured = req.Meta
		resp := core.NewResponse()
		resp.Body = req.Payload
		resp.StatusCode = 200
		return resp, nil
	})
}

func TestClientDo_StatusCode401(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 401
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrAuth) {
		t.Fatalf("期望 ErrAuth，got %v", err)
	}
}

func TestClientDo_StatusCode403(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 403
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrPermission) {
		t.Fatalf("期望 ErrPermission，got %v", err)
	}
}

func TestClientDo_StatusCode404(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 404
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrNotFound) {
		t.Fatalf("期望 ErrNotFound，got %v", err)
	}
}

func TestClientDo_StatusCode500(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 500
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrRemoteRuntime) {
		t.Fatalf("期望 ErrRemoteRuntime，got %v", err)
	}
}

func TestClientDo_StatusCode502(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 502
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	// 5xx 统一映射为 ErrRemoteRuntime
	if !core.IsKind(err, core.ErrRemoteRuntime) {
		t.Fatalf("502 应映射为 ErrRemoteRuntime，got %v", err)
	}
}

func TestClientDo_StatusCode499_Default(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 499 // 非标准 4xx
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	// 非 401/403/404 的 4xx 走 default → ErrNetwork
	if !core.IsKind(err, core.ErrNetwork) {
		t.Fatalf("499 应为 default → ErrNetwork，got %v", err)
	}
}

func TestClientDo_RequestIDPresent(t *testing.T) {
	var capturedID string
	tr := transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
		capturedID = req.ID
		resp := core.NewResponse()
		resp.RequestID = req.ID
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) {
		r := core.NewRequest("info")
		return r, nil
	}}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if res.OperationName() == "" {
		t.Fatal("Result 应带 OperationName")
	}
	if capturedID == "" {
		t.Fatal("req.ID 不应为空（应被 newRequestID 填充）")
	}
}

func TestClientDo_RiskLevelInjected(t *testing.T) {
	var captured map[string]string
	tr := captureTransport(&captured)
	c := core.NewClient(core.WithTransport(tr))

	op := &riskAwareStub{
		stubOp: stubOp{name: "exec", build: func() (*core.Request, error) { return core.NewRequest("exec"), nil }},
		risk:   core.RiskExec,
	}

	_, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if captured["risk_level"] != string(core.RiskExec) {
		t.Fatalf("期望 risk_level=exec，got %q", captured["risk_level"])
	}
}

func TestClientDo_RiskLevelNotInjectedForNonAwareOp(t *testing.T) {
	var captured map[string]string
	tr := captureTransport(&captured)
	c := core.NewClient(core.WithTransport(tr))

	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if _, ok := captured["risk_level"]; ok {
		t.Fatal("非 RiskAware 操作不应注入 risk_level")
	}
}

func TestClientDo_SessionMetadataMerged(t *testing.T) {
	var captured map[string]string
	tr := captureTransport(&captured)
	c := core.NewClient(
		core.WithTransport(tr),
		core.WithSession(&core.Session{
			NodeID:   "n1",
			Metadata: map[string]string{"auth_password_field": "mykey", "password_value": "secret"},
		}),
	)

	op := &stubOp{name: "info", build: func() (*core.Request, error) {
		r := core.NewRequest("info")
		// 如果 Build 已经写了同名的 key，session 不应覆盖
		r.Meta["auth_password_field"] = "from-build"
		return r, nil
	}}

	_, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	// session 的 password_value 应被合并
	if captured["password_value"] != "secret" {
		t.Fatalf("session metadata 未合并: password_value=%q", captured["password_value"])
	}
	// 但 session 不应覆盖 Build 已经写入的 key
	if captured["auth_password_field"] != "from-build" {
		t.Fatalf("session 不应覆盖 Build 已写入的 key: got %q", captured["auth_password_field"])
	}
}

func TestClientDo_EnvelopeMarkerRoundTrip(t *testing.T) {
	// 真实 transport 会把 req.Meta 里的 envelope tag 搬到 resp.Meta，
	// 否则 Extract 无法知道用什么 tag 截取。这里用同一份 mock 模拟该行为。
	tr := transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.Body = append([]byte{}, req.Payload...)
		resp.StatusCode = 200
		// 真实 transport 会把 envelope tag 从 req.Meta 搬到 resp.Meta
		for _, key := range []string{"marker.tag_s", "marker.tag_e"} {
			if v, ok := req.Meta[key]; ok {
				resp.Meta[key] = v
			}
		}
		return resp, nil
	})
	// 用不透明但封闭的 Client 挂 envelope
	c := core.NewClient(
		core.WithTransport(tr),
		core.WithEnvelope(envelopemarker.NewWithLength(8)),
	)

	op := &stubOp{
		name:  "echo",
		build: func() (*core.Request, error) { r := core.NewRequest("echo"); r.Payload = []byte("secret-payload"); return r, nil },
	}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if string(res.Raw()) != "secret-payload" {
		t.Fatalf("信封往返后 payload 应还原，got=%q", res.Raw())
	}
}

func TestClientDo_Base64CodecRoundTrip(t *testing.T) {
	tr := transportmock.New(transportmock.EchoHandler)
	c := core.NewClient(
		core.WithTransport(tr),
		core.WithCodec(codec.New()),
	)

	original := []byte("hello base64 world")
	op := &stubOp{
		name:  "echo",
		build: func() (*core.Request, error) { r := core.NewRequest("echo"); r.Payload = original; return r, nil },
	}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	if string(res.Raw()) != string(original) {
		t.Fatalf("base64 往返后 payload 应还原，got=%q", res.Raw())
	}

	// 手动验证 codec 确实工作过：编码后的 payload 应该出现在 transport 发出去的请求里
	var capturedPayload []byte
	c2 := core.NewClient(
		core.WithTransport(transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
			capturedPayload = req.Payload
			resp := core.NewResponse()
			resp.Body = req.Payload
			return resp, nil
		})),
		core.WithCodec(codec.New()),
	)
	_, err = c2.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("Do 出错: %v", err)
	}
	// 发出去的 payload 应是 base64 编码后的
	decoded, err := stdbase64.StdEncoding.DecodeString(string(capturedPayload))
	if err != nil {
		t.Fatalf("发出去的 payload 应是 base64 编码: %v", err)
	}
	if string(decoded) != string(original) {
		t.Fatalf("base64 往返后内容不一致: got=%q", decoded)
	}
}

func TestClientDo_200NoMapping(t *testing.T) {
	// 200 状态码不应触发任何错误映射
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 200
		resp.Body = []byte("ok")
		return resp, nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("200 不应报错: %v", err)
	}
	if string(res.Raw()) != "ok" {
		t.Fatalf("Body 不符合预期: %q", res.Raw())
	}
}

func TestClientDo_NilBuildResult(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})
	c := core.NewClient(core.WithTransport(tr))
	op := &stubOp{name: "broken", build: func() (*core.Request, error) { return nil, nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrProtocol) {
		t.Fatalf("nil req 应返回 ErrProtocol，got %v", err)
	}
}

func TestClientDo_NoTransport(t *testing.T) {
	c := core.NewClient()
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrNetwork) {
		t.Fatalf("未配置 transport 应返回 ErrNetwork，got %v", err)
	}
}

func TestClientDo_ParseError(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})
	c := core.NewClient(core.WithTransport(tr))
	sentinel := errors.New("parse boom")
	op := &stubOp{
		name:  "bad",
		build: func() (*core.Request, error) { return core.NewRequest("bad"), nil },
		parse: func(_ *core.Response) (core.Result, error) { return nil, sentinel },
	}

	_, err := c.Do(context.Background(), op)
	if !core.IsKind(err, core.ErrParse) {
		t.Fatalf("Parse 失败应映射为 ErrParse，got %v", err)
	}
}
