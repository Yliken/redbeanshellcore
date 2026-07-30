package httpform

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

// newServer 返回一个测试用 httptest.Server，handler 里能拿到请求体。
func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return s
}

func TestRoundTrip_Success(t *testing.T) {
	var gotBody string
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		gotBody = string(data)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok-response"))
	})

	tr := New(s.URL)
	// Timeout 已移至 Options 结构体
	req := core.NewRequest("exec")
	req.Payload = []byte("php-code-here")

	resp, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望 200，got %d", resp.StatusCode)
	}
	if string(resp.Body) != "ok-response" {
		t.Fatalf("body 不符合预期: %q", resp.Body)
	}
	// 主 payload 应出现在表单里
	if !strings.Contains(gotBody, "php-code-here") {
		t.Fatalf("表单里没有 payload: %q", gotBody)
	}
}

func TestRoundTrip_DefaultPasswordField(t *testing.T) {
	var gotForm url.Values
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main-payload")

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if got, _ := gotForm["antpwd"]; len(got) == 0 || got[0] != "main-payload" {
		t.Fatalf("默认密码字段 antpwd 未携带 payload: %+v", gotForm)
	}
}

func TestRoundTrip_CustomPasswordField(t *testing.T) {
	var gotForm url.Values
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main-payload")
	req.Meta["auth_password_field"] = "mykey"

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if got := gotForm.Get("mykey"); got != "main-payload" {
		t.Fatalf("自定义字段 mykey 未携带 payload: %+v", gotForm)
	}
	// 默认字段不应出现
	if _, exists := gotForm["antpwd"]; exists {
		t.Fatalf("自定义字段启用时不应再出现默认 antpwd")
	}
}

func TestRoundTrip_ParamsAsExtraFormFields(t *testing.T) {
	var gotForm url.Values
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	req.SetParamString("extra1", "v1")
	req.SetParamString("extra2", "v2")

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if gotForm.Get("extra1") != "v1" || gotForm.Get("extra2") != "v2" {
		t.Fatalf("额外参数未出现在表单里: %+v", gotForm)
	}
}

func TestRoundTrip_EnvTemplateVars(t *testing.T) {
	var gotForm url.Values
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	// 格式：k1|||askey|||v1|||asline|||k2|||askey|||v2
	req.Meta["env_template_vars"] = "k1|||askey|||v1|||asline|||k2|||askey|||v2"

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if gotForm.Get("k1") != "v1" || gotForm.Get("k2") != "v2" {
		t.Fatalf("env 模板变量未正确解析: %+v", gotForm)
	}
}

func TestRoundTrip_ServerErrorStatus(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main")

	resp, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("Transport 层不应把 HTTP 错误当 transport 错误: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("期望 500，got %d", resp.StatusCode)
	}
	if string(resp.Body) != "boom" {
		t.Fatalf("body 不符合预期: %q", resp.Body)
	}
}

func TestRoundTrip_EmptyEndpoint(t *testing.T) {
	tr := New("")
	req := core.NewRequest("exec")
	req.Payload = []byte("main")

	_, err := tr.RoundTrip(context.Background(), req)
	if err == nil {
		t.Fatal("endpoint 为空时应返回错误")
	}
}

func TestRoundTrip_InsecureTLS(t *testing.T) {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("tls-ok"))
	}))
	defer s.Close()

	tr := New(s.URL)
	tr.Options.InsecureTLS = true
	req := core.NewRequest("exec")
	req.Payload = []byte("main")

	resp, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("InsecureTLS RoundTrip 出错: %v", err)
	}
	if string(resp.Body) != "tls-ok" {
		t.Fatalf("body 不符合预期: %q", resp.Body)
	}
}

func TestRoundTrip_ExtraHeaders(t *testing.T) {
	var gotHeader string
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	tr.ExtraHeaders = map[string]string{"X-Custom": "hello"}
	req := core.NewRequest("exec")
	req.Payload = []byte("main")

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if gotHeader != "hello" {
		t.Fatalf("自定义头未发送: got=%q", gotHeader)
	}
}

func TestRoundTrip_RequestHeaders(t *testing.T) {
	var gotHeader string
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-From-Req")
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	req.SetHeader("X-From-Req", "from-req")

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if gotHeader != "from-req" {
		t.Fatalf("req 上的 header 未发送: got=%q", gotHeader)
	}
}

func TestRoundTrip_ContentTypeSet(t *testing.T) {
	var gotCT string
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.Payload = []byte("main")

	_, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Fatalf("Content-Type 不符合预期: %q", gotCT)
	}
}

func TestRoundTrip_RequestIDPropagated(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tr := New(s.URL)
	req := core.NewRequest("exec")
	req.ID = "fixed-req-id-123"
	req.Payload = []byte("main")

	resp, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip 出错: %v", err)
	}
	if resp.RequestID != "fixed-req-id-123" {
		t.Fatalf("RequestID 未透传到 response: %q", resp.RequestID)
	}
}

// 防止未来重构时误删 import 的保守检查
var _ = tls.VersionTLS13
