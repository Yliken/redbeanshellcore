// Package noop provides a no-op Crypto implementation that passes
// the request and response through unchanged.
//
// This is useful as a default, for testing, or as a placeholder
// when encryption is not needed but a non-nil Crypto is required.
package noop

import (
	"context"

	"github.com/Yliken/redbeanshellcore/core"
)

type Crypto struct{}

// New returns a no-op Crypto that performs no encryption or decryption.
func New() *Crypto { return &Crypto{} }

func (c *Crypto) Name() string { return "noop" }

func (c *Crypto) Encrypt(_ context.Context, req *core.Request) (*core.Request, error) {
	return req, nil
}

func (c *Crypto) Decrypt(_ context.Context, resp *core.Response) (*core.Response, error) {
	return resp, nil
}
