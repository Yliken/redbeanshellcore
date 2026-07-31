package wire

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

// BodyCodec serializes a request parameter map into a wire body and back.
// Transports use it before applying BodyCrypto so the whole form travels as a
// single encrypted field.
type BodyCodec interface {
	Encode(values map[string][]byte) ([]byte, error)
	Decode(data []byte) (map[string][]byte, error)
}

// CompactFormCodec is the default BodyCodec. It sorts field names, base64
// encodes every value, and joins pairs with "&" (key=base64value). PHP's
// parse_str and Java's URLDecoder can restore the original fields directly.
type CompactFormCodec struct{}

// NewCompactFormCodec builds a CompactFormCodec.
func NewCompactFormCodec() *CompactFormCodec { return &CompactFormCodec{} }

// Encode serializes values with sorted keys. Field names must not contain '='
// or '&'; generated adapter field names satisfy this constraint.
func (c *CompactFormCodec) Encode(values map[string][]byte) ([]byte, error) {
	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.ContainsAny(key, "&=") {
			return nil, fmt.Errorf("wire: compact form field name %q contains reserved character", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for i, key := range keys {
		if i > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(base64.StdEncoding.EncodeToString(values[key]))
	}
	return []byte(builder.String()), nil
}

// Decode restores a map produced by Encode.
func (c *CompactFormCodec) Decode(data []byte) (map[string][]byte, error) {
	out := make(map[string][]byte)
	if len(data) == 0 {
		return out, nil
	}
	for _, pair := range strings.Split(string(data), "&") {
		eq := strings.IndexByte(pair, '=')
		if eq <= 0 {
			return nil, fmt.Errorf("wire: compact form pair %q has no key", pair)
		}
		key := pair[:eq]
		value, err := base64.StdEncoding.DecodeString(pair[eq+1:])
		if err != nil {
			return nil, fmt.Errorf("wire: compact form value for %q is not valid base64: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}
