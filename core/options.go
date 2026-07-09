package core

import "context"

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
