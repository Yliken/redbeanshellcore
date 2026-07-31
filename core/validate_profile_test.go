package core

import (
	"context"
	"strings"
	"testing"
)

func TestValidateProfileEmptyEnvelopesRejectsEveryEnvelope(t *testing.T) {
	for _, adapter := range []string{"asp", "aspx", "jsp", "php-eval"} {
		if err := ValidateProfile("", "marker", adapter); err == nil {
			t.Fatalf("adapter %s 的空 Envelopes 应拒绝 marker", adapter)
		}
	}
	if err := ValidateProfile("", "marker", "php"); err != nil {
		t.Fatalf("php 应支持 marker: %v", err)
	}
}

func TestValidateProfileNilEnvelopesSkipsValidation(t *testing.T) {
	KnownAdapterProfiles["nil-env"] = AdapterProfile{Name: "nil-env", Envelopes: nil}
	defer delete(KnownAdapterProfiles, "nil-env")
	if err := ValidateProfile("", "anything", "nil-env"); err != nil {
		t.Fatalf("nil Envelopes 应跳过 envelope 校验: %v", err)
	}
}

func TestValidateProfileAnySentinelSkipsValidation(t *testing.T) {
	KnownAdapterProfiles["any-env"] = AdapterProfile{Name: "any-env", Envelopes: []string{"any"}}
	defer delete(KnownAdapterProfiles, "any-env")
	if err := ValidateProfile("", "whatever", "any-env"); err != nil {
		t.Fatalf("any 哨兵应跳过 envelope 校验: %v", err)
	}
}

func TestAdapterProfileSupportsCrypto(t *testing.T) {
	if !KnownAdapterProfiles["php-eval"].SupportsCrypto("aes-gcm") {
		t.Fatal("php-eval 应支持 aes-gcm")
	}
	if KnownAdapterProfiles["php"].SupportsCrypto("aes-gcm") {
		t.Fatal("php 不应支持 aes-gcm")
	}
	if !KnownAdapterProfiles["jsp"].SupportsCrypto("aes-gcm") {
		t.Fatal("jsp 应支持 aes-gcm")
	}

	legacy := AdapterProfile{Crypto: true}
	if !legacy.SupportsCrypto("xor") {
		t.Fatal("CryptoModes 为空时应回退 Crypto bool")
	}
	strict := AdapterProfile{Crypto: true, CryptoModes: []string{"aes-gcm"}}
	if strict.SupportsCrypto("xor") {
		t.Fatal("CryptoModes 非空时应按列表校验")
	}
}

type namedEnvelope struct {
	name string
}

func (e namedEnvelope) Name() string { return e.name }
func (e namedEnvelope) Wrap(_ context.Context, req *Request) (*Request, error) {
	return req, nil
}
func (e namedEnvelope) Extract(_ context.Context, resp *Response) (*Response, error) {
	return resp, nil
}

type plainEnvelope struct{}

func (plainEnvelope) Wrap(_ context.Context, req *Request) (*Request, error) {
	return req, nil
}
func (plainEnvelope) Extract(_ context.Context, resp *Response) (*Response, error) {
	return resp, nil
}

func TestClientEnvelopeNameOptionalInterface(t *testing.T) {
	KnownAdapterProfiles["env-test"] = AdapterProfile{Name: "env-test", Envelopes: []string{"custom"}}
	defer delete(KnownAdapterProfiles, "env-test")

	c := NewClient(
		WithSession(&Session{NodeID: "n1", Adapter: "env-test"}),
		WithTransport(bodyCryptoEchoTransport{}),
		WithEnvelope(namedEnvelope{name: "custom"}),
	)
	if _, err := c.Do(context.Background(), bodyCryptoTestOp{}); err != nil {
		t.Fatalf("带 Name() 的 envelope 应按真实名称校验: %v", err)
	}

	c = NewClient(
		WithSession(&Session{NodeID: "n1", Adapter: "env-test"}),
		WithTransport(bodyCryptoEchoTransport{}),
		WithEnvelope(plainEnvelope{}),
	)
	if _, err := c.Do(context.Background(), bodyCryptoTestOp{}); err == nil {
		t.Fatal("无 Name() 的 envelope 回退 marker 后应被 env-test profile 拒绝")
	}
}

type aesGCMBodyCrypto struct{}

func (aesGCMBodyCrypto) Name() string { return "aes-gcm" }
func (aesGCMBodyCrypto) EncryptBody(_ context.Context, body []byte) ([]byte, error) {
	return body, nil
}
func (aesGCMBodyCrypto) DecryptBody(_ context.Context, body []byte) ([]byte, error) {
	return body, nil
}

func TestClientCryptoModeMismatch(t *testing.T) {
	c := NewClient(
		WithSession(&Session{
			NodeID:   "n1",
			Adapter:  "php-eval",
			Metadata: map[string]string{"crypto_mode": "xor"},
		}),
		WithTransport(bodyCryptoEchoTransport{}),
		WithBodyCrypto(aesGCMBodyCrypto{}),
	)
	_, err := c.Do(context.Background(), bodyCryptoTestOp{})
	if err == nil || !strings.Contains(err.Error(), "crypto mode mismatch") {
		t.Fatalf("应返回 crypto mode mismatch，got %v", err)
	}
}

func TestClientCryptoModeMatch(t *testing.T) {
	c := NewClient(
		WithSession(&Session{
			NodeID:   "n1",
			Adapter:  "php-eval",
			Metadata: map[string]string{"crypto_mode": "aes-gcm"},
		}),
		WithTransport(bodyCryptoEchoTransport{}),
		WithBodyCrypto(aesGCMBodyCrypto{}),
	)
	if _, err := c.Do(context.Background(), bodyCryptoTestOp{}); err != nil {
		t.Fatalf("匹配的 crypto mode 不应报错: %v", err)
	}
}
