package noop_test

import (
	"context"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/noop"
)

func TestNoopName(t *testing.T) {
	c := noop.New()
	if got := c.Name(); got != "noop" {
		t.Fatalf("expected name 'noop', got %q", got)
	}
}

func TestNoopEncrypt(t *testing.T) {
	c := noop.New()
	payload := []byte("hello world")
	req := core.NewRequest("test")
	req.Payload = payload

	out, err := c.Encrypt(context.Background(), req)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	if out != req {
		t.Fatal("Encrypt should return the same request pointer")
	}
	if string(out.Payload) != string(payload) {
		t.Fatalf("payload changed: got %q, want %q", string(out.Payload), string(payload))
	}
}

func TestNoopDecrypt(t *testing.T) {
	c := noop.New()
	body := []byte("response data")
	resp := core.NewResponse()
	resp.Body = body

	out, err := c.Decrypt(context.Background(), resp)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if out != resp {
		t.Fatal("Decrypt should return the same response pointer")
	}
	if string(out.Body) != string(body) {
		t.Fatalf("body changed: got %q, want %q", string(out.Body), string(body))
	}
}

func TestNoopEmptyPayload(t *testing.T) {
	c := noop.New()
	req := core.NewRequest("test")
	req.Payload = []byte{}

	out, err := c.Encrypt(context.Background(), req)
	if err != nil {
		t.Fatalf("Encrypt with empty payload returned error: %v", err)
	}
	if len(out.Payload) != 0 {
		t.Fatalf("expected empty payload, got %d bytes", len(out.Payload))
	}
}
