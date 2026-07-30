package aesgcm_test

import (
	"bytes"
	"encoding/base64"
	"context"
	"crypto/rand"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/aesgcm"
)

func TestNewInvalidKeySize(t *testing.T) {
	tests := []int{0, 1, 8, 15, 20, 33, 64}
	for _, n := range tests {
		key := make([]byte, n)
		_, err := aesgcm.New(key)
		if err == nil {
			t.Fatalf("expected error for key size %d, got nil", n)
		}
	}
}

func TestNewValidKeySizes(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		key := make([]byte, n)
		c, err := aesgcm.New(key)
		if err != nil {
			t.Fatalf("unexpected error for key size %d: %v", n, err)
		}
		if c.Name() != "aes-gcm" {
			t.Fatalf("expected name 'aes-gcm', got %q", c.Name())
		}
	}
}

func TestAESGCMRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, err := aesgcm.New(key)
	if err != nil {
		t.Fatal(err)
	}

	original := []byte("sensitive command payload data")
	req := core.NewRequest("exec")
	req.Payload = append([]byte{}, original...)

	ctx := context.Background()

	encReq, err := c.Encrypt(ctx, req)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if bytes.Equal(encReq.Payload, original) {
		t.Fatal("encrypted payload should differ from original")
	}

	if encReq.Meta["crypto"] != "aes-gcm" {
		t.Fatalf("missing crypto metadata")
	}

	resp := core.NewResponse()
	resp.Body = append([]byte{}, encReq.Payload...)

	decResp, err := c.Decrypt(ctx, resp)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decResp.Body, original) {
		t.Fatalf("round-trip mismatch: got %q, want %q", string(decResp.Body), string(original))
	}
}

func TestAESGCMWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, _ = rand.Read(key1)
	_, _ = rand.Read(key2)

	c1, _ := aesgcm.New(key1)
	c2, _ := aesgcm.New(key2)

	ctx := context.Background()
	req := core.NewRequest("test")
	req.Payload = []byte("secret message")

	encReq, err := c1.Encrypt(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	resp := core.NewResponse()
	resp.Body = append([]byte{}, encReq.Payload...)

	_, err = c2.Decrypt(ctx, resp)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key, got nil")
	}
}

func TestAESGCMTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, _ := aesgcm.New(key)

	ctx := context.Background()
	req := core.NewRequest("test")
	req.Payload = []byte("tamper test")

	encReq, err := c.Encrypt(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ := base64.StdEncoding.DecodeString(string(encReq.Payload))
	raw[len(raw)-1] ^= 0xFF
	encReq.Payload = []byte(base64.StdEncoding.EncodeToString(raw))

	resp := core.NewResponse()
	resp.Body = append([]byte{}, encReq.Payload...)

	_, err = c.Decrypt(ctx, resp)
	if err == nil {
		t.Fatal("expected error when decrypting tampered ciphertext, got nil")
	}
}

func TestAESGCMEmptyPayload(t *testing.T) {
	key := make([]byte, 32)
	c, _ := aesgcm.New(key)

	ctx := context.Background()
	req := core.NewRequest("test")
	req.Payload = []byte{}

	out, err := c.Encrypt(ctx, req)
	if err != nil {
		t.Fatalf("Encrypt with empty payload failed: %v", err)
	}
	if len(out.Payload) != 0 {
		t.Fatalf("expected empty encrypted payload, got %d bytes", len(out.Payload))
	}
}

func TestAESGCMEmptyResponse(t *testing.T) {
	key := make([]byte, 32)
	c, _ := aesgcm.New(key)

	ctx := context.Background()
	resp := core.NewResponse()
	resp.Body = []byte{}

	out, err := c.Decrypt(ctx, resp)
	if err != nil {
		t.Fatalf("Decrypt with empty body failed: %v", err)
	}
	if len(out.Body) != 0 {
		t.Fatalf("expected empty decrypted body, got %d bytes", len(out.Body))
	}
}



