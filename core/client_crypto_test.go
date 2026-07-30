package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/noop"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

// errCrypto is a test crypto that returns an error on Encrypt.
type errCrypto struct{}

func (e *errCrypto) Name() string { return "err" }
func (e *errCrypto) Encrypt(_ context.Context, _ *core.Request) (*core.Request, error) {
	return nil, errors.New("encrypt failed")
}
func (e *errCrypto) Decrypt(_ context.Context, _ *core.Response) (*core.Response, error) {
	return nil, errors.New("decrypt failed")
}

func TestClientWithNoopCrypto(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, req *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.StatusCode = 200; resp.Body = make([]byte, len(req.Payload))
		copy(resp.Body, req.Payload)
		return resp, nil
	})
	c := core.NewClient(
		core.WithSession(&core.Session{NodeID: "n1", Endpoint: "http://x"}),
		core.WithTransport(tr),
		core.WithCrypto(noop.New()),
	)

	op := &stubOp{
		name: "echo",
		build: func() (*core.Request, error) {
			r := core.NewRequest("echo")
			r.Payload = []byte("hello crypto")
			return r, nil
		},
	}

	res, err := c.Do(context.Background(), op)
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if string(res.Raw()) != "hello crypto" {
		t.Fatalf("raw mismatch: got %q, want %q", string(res.Raw()), "hello crypto")
	}
}

func TestClientCryptoEncryptError(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})
	c := core.NewClient(
		core.WithTransport(tr),
		core.WithCrypto(&errCrypto{}),
	)

	op := &stubOp{
		name: "broken",
		build: func() (*core.Request, error) {
			return core.NewRequest("broken"), nil
		},
	}

	_, err := c.Do(context.Background(), op)
	if err == nil {
		t.Fatal("expected error from crypto encrypt")
	}
	if !core.IsKind(err, core.ErrCrypto) {
		t.Fatalf("expected ErrCrypto, got %v", err)
	}
}

func TestClientCryptoDecryptError(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		return core.NewResponse(), nil
	})
	c := core.NewClient(
		core.WithTransport(tr),
		core.WithCrypto(&errCrypto{}),
	)

	op := &stubOp{
		name: "noop",
		build: func() (*core.Request, error) {
			r := core.NewRequest("noop")
			r.Payload = []byte("data")
			return r, nil
		},
	}

	_, err := c.Do(context.Background(), op)
	if err == nil {
		t.Fatal("expected error from crypto decrypt")
	}
	if !core.IsKind(err, core.ErrCrypto) {
		t.Fatalf("expected ErrCrypto, got %v", err)
	}
}
