package core

import "context"

// CodecFn 根据 NodeRecord 构造一个 Codec。TransportFn / EnvelopeFn 同理。
type CodecFn func(ctx context.Context, record *NodeRecord) (Codec, error)
type TransportFn func(ctx context.Context, record *NodeRecord) (Transport, error)
type EnvelopeFn func(ctx context.Context, record *NodeRecord) (Envelope, error)
type AdapterFn func(ctx context.Context, record *NodeRecord) (adapterRegistry, error)

// adapterRegistry 是 Factory 探测适配器能力用的内部接口。
type adapterRegistry interface {
	Capabilities(ctx context.Context, record *NodeRecord) []Capability
}

// DefaultClientFactory 是框架自带的默认 Factory。
func DefaultClientFactory() ClientFactory {
	return &defaultBuilder{}
}

type defaultBuilder struct{}

func (b *defaultBuilder) NewClient(ctx context.Context, record *NodeRecord) (*Client, error) {
	// Transport 必填。
	transport, err := selectTransport(ctx, record)
	if err != nil {
		return nil, err
	}
	codec, err := selectCodec(ctx, record)
	if err != nil {
		return nil, err
	}
	envelope, err := selectEnvelope(ctx, record)
	if err != nil {
		return nil, err
	}
	sess := &Session{
		NodeID:       record.Config.ID,
		Endpoint:     record.Config.Endpoint,
		Adapter:      record.Config.Adapter,
		Transport:    record.Config.Transport,
		Codec:        record.Config.Codec,
		Capabilities: append([]Capability(nil), record.Capabilities...),
		Metadata:     mergeMaps(record.Metadata, nil),
	}
	// 把 Auth 合并到 metadata，让 Transport 自己按 key 读取。
	if record.Config.Auth != nil {
		sess.Metadata = mergeMaps(sess.Metadata, record.Config.Auth)
	}
	return NewClient(
		WithSession(sess),
		WithTransport(transport),
		WithCodec(codec),
		WithEnvelope(envelope),
	), nil
}

func mergeMaps(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
