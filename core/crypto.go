// Package core defines the Crypto interface for traffic encryption.
//
// Crypto sits between the Envelope and Transport layers in the Client pipeline.
// Implementations encrypt the fully assembled request payload before transmission
// and decrypt the raw response body before envelope extraction.
//
// This gives developers full control over how traffic is encrypted, enabling
// custom algorithms, key management, and protocol-specific transformations.
package core

import "context"

// Crypto handles encryption and decryption of request/response payloads.
//
// Position in the Client.Do() pipeline:
//
//	Request:  Build → Codec → Envelope → Transforms → Crypto.Encrypt → Transport
//	Response: Transport → Crypto.Decrypt → Transforms → Envelope → Codec → Parse
//
// Implementing this interface in your own package lets you plug custom
// encryption into the SDK without modifying core code.
//
// Built-in implementations live under crypto/ (aesgcm, noop).
type Crypto interface {
	// Name returns a short identifier for this crypto (e.g. "aes-gcm", "xor").
	Name() string

	// Encrypt transforms the request payload before it is sent over the wire.
	// The implementation should modify req.Payload in place or return a new Request.
	Encrypt(ctx context.Context, req *Request) (*Request, error)

	// Decrypt transforms the response body after it is received from the wire.
	// The implementation should modify resp.Body in place or return a new Response.
	Decrypt(ctx context.Context, resp *Response) (*Response, error)
}

// BodyCrypto encrypts and decrypts the complete wire body. It is a
// transport-level alternative to Crypto: the transport serializes the whole
// form, encrypts it, and submits a single crypto field, then decrypts the
// response before the normal response chain runs.
//
// BodyCrypto and Crypto are mutually exclusive on a Client. Configure the
// transport separately (for example transport/httpform.Options.BodyCrypto)
// and use WithBodyCrypto on the Client so the payload-level crypto step is
// skipped.
type BodyCrypto interface {
	// Name returns a short identifier for this body crypto (e.g. "aes-gcm").
	Name() string

	// EncryptBody encrypts the serialized request body.
	EncryptBody(ctx context.Context, body []byte) ([]byte, error)

	// DecryptBody decrypts the raw response body.
	DecryptBody(ctx context.Context, body []byte) ([]byte, error)
}
