package core

import (
	"context"
	"strings"
	"testing"
)

type bodyCryptoTestBody struct{}

func (bodyCryptoTestBody) Name() string { return "test-body" }
func (bodyCryptoTestBody) EncryptBody(_ context.Context, body []byte) ([]byte, error) {
	return body, nil
}
func (bodyCryptoTestBody) DecryptBody(_ context.Context, body []byte) ([]byte, error) {
	return body, nil
}

type bodyCryptoTestPayload struct {
	called bool
}

func (b *bodyCryptoTestPayload) Name() string { return "test-payload" }
func (b *bodyCryptoTestPayload) Encrypt(_ context.Context, req *Request) (*Request, error) {
	b.called = true
	return req, nil
}
func (b *bodyCryptoTestPayload) Decrypt(_ context.Context, resp *Response) (*Response, error) {
	b.called = true
	return resp, nil
}

type bodyCryptoTestOp struct{}

func (bodyCryptoTestOp) Name() string { return "echo" }
func (bodyCryptoTestOp) Build(_ context.Context, _ *Session) (*Request, error) {
	req := NewRequest("echo")
	req.Payload = []byte("hello")
	return req, nil
}
func (bodyCryptoTestOp) Parse(_ context.Context, resp *Response) (Result, error) {
	return &BoolResult{BaseResult: NewBaseResult("echo", resp.Body), OK: true}, nil
}

type bodyCryptoEchoTransport struct{}

func (bodyCryptoEchoTransport) RoundTrip(_ context.Context, req *Request) (*Response, error) {
	resp := NewResponse()
	resp.Body = append([]byte{}, req.Payload...)
	resp.StatusCode = 200
	return resp, nil
}

func TestWithBodyCryptoAndWithCryptoConflict(t *testing.T) {
	c := NewClient(WithCrypto(&bodyCryptoTestPayload{}), WithBodyCrypto(bodyCryptoTestBody{}))
	_, err := c.Do(context.Background(), bodyCryptoTestOp{})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("WithCrypto+WithBodyCrypto 应返回互斥错误，got %v", err)
	}

	c = NewClient(WithBodyCrypto(bodyCryptoTestBody{}), WithCrypto(&bodyCryptoTestPayload{}))
	_, err = c.Do(context.Background(), bodyCryptoTestOp{})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("WithBodyCrypto+WithCrypto 应返回互斥错误，got %v", err)
	}
}

func TestClientBodyCryptoSkipsPayloadCrypto(t *testing.T) {
	payload := &bodyCryptoTestPayload{}
	c := NewClient(
		WithTransport(bodyCryptoEchoTransport{}),
		WithBodyCrypto(bodyCryptoTestBody{}),
	)
	// Simulate a lower-level configuration that still holds a payload crypto:
	// body crypto must win and the payload crypto must not run.
	c.crypto = payload

	res, err := c.Do(context.Background(), bodyCryptoTestOp{})
	if err != nil {
		t.Fatalf("Do 不应失败: %v", err)
	}
	if string(res.Raw()) != "hello" {
		t.Fatalf("raw 不匹配: got %q", res.Raw())
	}
	if payload.called {
		t.Fatal("BodyCrypto 存在时不应执行 payload 级加解密")
	}
}

func TestClientWithBodyCryptoSucceeds(t *testing.T) {
	c := NewClient(
		WithTransport(bodyCryptoEchoTransport{}),
		WithBodyCrypto(bodyCryptoTestBody{}),
	)
	if _, err := c.Do(context.Background(), bodyCryptoTestOp{}); err != nil {
		t.Fatalf("BodyCrypto 路径不应失败: %v", err)
	}
}
