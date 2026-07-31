// Package aesgcm provides an AES-256-GCM Crypto implementation.
//
// It uses AES-GCM authenticated encryption with a random nonce per
// invocation. The wire format is:
//
//	nonce (12 bytes) || ciphertext || auth tag (16 bytes)
//
// Key size must be 16 (AES-128), 24 (AES-192), or 32 (AES-256) bytes.
// AES-256 (32 bytes) is strongly recommended.
package aesgcm

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Crypto implements core.Crypto using AES-GCM authenticated encryption.
type Crypto struct {
	key []byte
}

// New creates an AES-GCM Crypto with the given key.
// Valid key sizes are 16, 24, and 32 bytes (AES-128/192/256).
func New(key []byte) (*Crypto, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("aesgcm: key must be 16, 24, or 32 bytes")
	}
	return &Crypto{key: key}, nil
}

func (c *Crypto) Name() string { return "aes-gcm" }

// Encrypt encrypts req.Payload using AES-GCM.
// The wire format is base64(nonce || ciphertext || auth tag).
// Base64 encoding makes the output suitable for PHP shells that
// expect a base64-encoded input.
func (c *Crypto) Encrypt(_ context.Context, req *core.Request) (*core.Request, error) {
	if len(req.Payload) == 0 {
		return req, nil
	}

	sealed, err := c.seal(req.Payload)
	if err != nil {
		return nil, err
	}
	// Base64 encode for transport compatibility
	req.Payload = []byte(base64.StdEncoding.EncodeToString(sealed))
	req.SetMeta("crypto", "aes-gcm")
	return req, nil
}

// Decrypt decrypts resp.Body using AES-GCM.
// Expects wire format: base64(nonce || ciphertext || auth tag).
// If the body is not valid base64-encoded ciphertext (e.g. plaintext
// response from a PHP shell), it returns the body unchanged.
func (c *Crypto) Decrypt(_ context.Context, resp *core.Response) (*core.Response, error) {
	if len(resp.Body) == 0 {
		return resp, nil
	}

	// First, base64-decode the wire format
	decoded, b64Err := base64.StdEncoding.DecodeString(string(resp.Body))
	if b64Err != nil {
		// Not base64-encoded 鈥?pass through (e.g. plaintext response)
		return resp, nil
	}
	if len(decoded) < 12 {
		return resp, nil
	}

	plaintext, err := c.open(decoded)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: decrypt: %w", err)
	}

	resp.Body = plaintext
	return resp, nil
}

// EncryptBody implements core.BodyCrypto using the same
// base64(nonce || ciphertext || auth tag) wire format.
func (c *Crypto) EncryptBody(_ context.Context, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	sealed, err := c.seal(body)
	if err != nil {
		return nil, err
	}
	return []byte(base64.StdEncoding.EncodeToString(sealed)), nil
}

// DecryptBody implements core.BodyCrypto. Unlike Crypto.Decrypt it is strict:
// a malformed body is an error because a body-encrypted shell always returns
// encrypted responses.
func (c *Crypto) DecryptBody(_ context.Context, body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		return nil, fmt.Errorf("aesgcm: body is not valid base64 ciphertext: %w", err)
	}
	return c.open(decoded)
}

func (c *Crypto) seal(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("aesgcm: generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *Crypto) open(sealed []byte) ([]byte, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: new GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(sealed) < nonceSize {
		return nil, errors.New("aesgcm: ciphertext too short")
	}
	nonce, ciphertext := sealed[:nonceSize], sealed[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aesgcm: decrypt: %w", err)
	}
	return plaintext, nil
}

var _ core.BodyCrypto = (*Crypto)(nil)
