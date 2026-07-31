package core

import (
	"context"
	"errors"
)

// Option 是 Client 的功能选项。
type Option func(*Client)

// WithSession 把 Session 挂到 Client 上。
func WithSession(s *Session) Option {
	return func(c *Client) { c.session = s }
}

// WithTransport 替换 Client 的传输层。
func WithTransport(t Transport) Option {
	return func(c *Client) { c.transport = t }
}

// WithCodec 替换 Client 的编解码器。
func WithCodec(co Codec) Option {
	return func(c *Client) { c.codec = co }
}

// WithEnvelope 替换 Client 的边界协议。
func WithEnvelope(e Envelope) Option {
	return func(c *Client) { c.envelope = e }
}

// WithTransforms 追加流量变形。
func WithTransforms(ts ...Transform) Option {
	return func(c *Client) { c.transforms = append(c.transforms, ts...) }
}

// WithMiddleware 追加中间件。
func WithMiddleware(mws ...Middleware) Option {
	return func(c *Client) { c.middlewares = append(c.middlewares, mws...) }
}

// WithCrypto 设置 Client 的流量加密组件。
// 当设置了 Crypto，请求在发送前会先加密，响应在接收后会先解密。
// 留空（不调用此选项）则不做加解密，与之前行为一致。
func WithCrypto(cr Crypto) Option {
	return func(c *Client) {
		if isNilInterface(cr) {
			return
		}
		if !isNilInterface(c.bodyCrypto) {
			c.configErr = errors.New("core: WithCrypto conflicts with WithBodyCrypto: payload-level and body-level encryption are mutually exclusive")
			return
		}
		c.crypto = cr
	}
}

// WithBodyCrypto sets the Client's body-level encryption component.
//
// When BodyCrypto is configured the Client skips the payload-level
// Encrypt/Decrypt steps; the transport owns encryption of the whole wire body.
// WithBodyCrypto and WithCrypto are mutually exclusive. Leave unset (do not
// call this option) to keep the previous payload-level behavior.
func WithBodyCrypto(bc BodyCrypto) Option {
	return func(c *Client) {
		if isNilInterface(bc) {
			return
		}
		if !isNilInterface(c.crypto) {
			c.configErr = errors.New("core: WithBodyCrypto conflicts with WithCrypto: body-level and payload-level encryption are mutually exclusive")
			return
		}
		c.bodyCrypto = bc
	}
}

type nodeKey struct{}

// ContextWithNode 把 nodeID 注入到 context 中。
func ContextWithNode(ctx context.Context, nodeID string) context.Context {
	return context.WithValue(ctx, nodeKey{}, nodeID)
}

// NodeFromContext 从 context 取出 nodeID；取不到时返回 ""。
func NodeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(nodeKey{}).(string); ok {
		return v
	}
	return ""
}
