package aesgcm

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestBodyCryptoRoundTrip(t *testing.T) {
	cr, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	body := []byte("antpwd=hello&z1=world")
	encrypted, err := cr.EncryptBody(context.Background(), body)
	if err != nil {
		t.Fatalf("EncryptBody: %v", err)
	}

	raw, err := base64.StdEncoding.DecodeString(string(encrypted))
	if err != nil {
		t.Fatalf("输出应为 base64: %v", err)
	}
	if len(raw) < 12+16 {
		t.Fatalf("wire body 应包含 nonce+ciphertext+tag，got %d bytes", len(raw))
	}
	if strings.Contains(string(encrypted), "hello") {
		t.Fatal("密文不应包含明文")
	}

	decrypted, err := cr.DecryptBody(context.Background(), encrypted)
	if err != nil {
		t.Fatalf("DecryptBody: %v", err)
	}
	if string(decrypted) != string(body) {
		t.Fatalf("round trip 不匹配: got %q", decrypted)
	}
}

func TestBodyCryptoStrictDecrypt(t *testing.T) {
	cr, err := New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := cr.DecryptBody(context.Background(), []byte("not base64 !!")); err == nil {
		t.Fatal("非 base64 body 应返回错误")
	}

	encrypted, err := cr.EncryptBody(context.Background(), []byte("payload"))
	if err != nil {
		t.Fatalf("EncryptBody: %v", err)
	}
	tampered := []byte(string(encrypted[:len(encrypted)-2]) + "AA")
	if _, err := cr.DecryptBody(context.Background(), []byte(tampered)); err == nil {
		t.Fatal("篡改的密文应返回认证失败")
	}
}
