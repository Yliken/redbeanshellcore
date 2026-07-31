package httpform

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/aesgcm"
	"github.com/Yliken/redbeanshellcore/protocol/wire"
)

var bodyCryptoKey = []byte("0123456789abcdef0123456789abcdef")

func TestRoundTrip_BodyCryptoFullChain(t *testing.T) {
	cr, err := aesgcm.New(bodyCryptoKey)
	if err != nil {
		t.Fatalf("aesgcm.New: %v", err)
	}

	var gotForm string
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm.Encode()

		encrypted := []byte(r.PostForm.Get("__crypto"))
		plain, err := cr.DecryptBody(context.Background(), encrypted)
		if err != nil {
			t.Errorf("server decrypt: %v", err)
		}
		values, err := wire.NewCompactFormCodec().Decode(plain)
		if err != nil {
			t.Errorf("server decode body: %v", err)
		}
		if string(values["antpwd"]) != "main-payload" {
			t.Errorf("antpwd 不在加密体内: %q", values["antpwd"])
		}
		if string(values["extra"]) != "v1" {
			t.Errorf("extra 不在加密体内: %q", values["extra"])
		}

		respBody, err := cr.EncryptBody(context.Background(), []byte("ok-encrypted"))
		if err != nil {
			t.Errorf("server encrypt response: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBody)
	})

	tr := NewWithOptions(s.URL, Options{BodyCrypto: cr})
	req := core.NewRequest("exec")
	req.Payload = []byte("main-payload")
	req.SetParamString("extra", "v1")

	resp, err := tr.RoundTrip(context.Background(), req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if string(resp.Body) != "ok-encrypted" {
		t.Fatalf("响应未解密: got %q", resp.Body)
	}
	if strings.Contains(gotForm, "main-payload") || strings.Contains(gotForm, "extra=v1") {
		t.Fatalf("明文字段不应出现在请求表单: %q", gotForm)
	}
	if !strings.Contains(gotForm, "__crypto=") {
		t.Fatalf("加密体应放在 __crypto 字段: %q", gotForm)
	}
}

func TestRoundTrip_BodyCryptoCustomField(t *testing.T) {
	cr, _ := aesgcm.New(bodyCryptoKey)
	var gotForm url.Values
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotForm = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	tr := NewWithOptions(s.URL, Options{BodyCrypto: cr, CryptoField: "c"})
	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	if _, err := tr.RoundTrip(context.Background(), req); err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if gotForm.Get("c") == "" {
		t.Fatalf("自定义加密字段 c 缺失: %+v", gotForm)
	}
	if gotForm.Get("__crypto") != "" {
		t.Fatalf("默认字段 __crypto 不应出现: %+v", gotForm)
	}
}

func TestRoundTrip_BodyCryptoWireProtocolConflict(t *testing.T) {
	cr, _ := aesgcm.New(bodyCryptoKey)
	tr := New("http://example.com/shell.php")
	tr.Options.BodyCrypto = cr
	tr.Options.WireProtocol = true

	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	_, err := tr.RoundTrip(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "互斥") {
		t.Fatalf("BodyCrypto+WireProtocol 应返回互斥错误，got %v", err)
	}
}

func TestRoundTrip_BodyCryptoDecryptError(t *testing.T) {
	cr, _ := aesgcm.New(bodyCryptoKey)
	s := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("plaintext-not-encrypted"))
	})

	tr := NewWithOptions(s.URL, Options{BodyCrypto: cr})
	req := core.NewRequest("exec")
	req.Payload = []byte("main")
	_, err := tr.RoundTrip(context.Background(), req)
	if err == nil || !core.IsKind(err, core.ErrCrypto) {
		t.Fatalf("响应解密失败应返回 ErrCrypto，got %v", err)
	}
}
