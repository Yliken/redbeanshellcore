package core

import (
	"context"
	"testing"
)

func TestDefaultSelector_TransportEmpty(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Transport: ""}}
	_, err := selectTransport(context.Background(), rec)
	if err == nil {
		t.Fatal("空 transport 默认不应接入（需自定义 ClientFactory）")
	}
}

func TestDefaultSelector_TransportHttpform(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Transport: "httpform"}}
	_, err := selectTransport(context.Background(), rec)
	if err == nil {
		t.Fatal("httpform 默认不应接入（需自定义 ClientFactory）")
	}
}

func TestDefaultSelector_TransportUnknown(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Transport: "grpc"}}
	_, err := selectTransport(context.Background(), rec)
	if err == nil {
		t.Fatal("未知 transport 应返回错误")
	}
}

func TestDefaultSelector_CodecPlain(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Codec: "plain"}}
	codec, err := selectCodec(context.Background(), rec)
	if err != nil {
		t.Fatalf("plain codec 不应出错: %v", err)
	}
	if codec != nil {
		t.Fatal("plain codec 应返回 nil")
	}
}

func TestDefaultSelector_CodecEmpty(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Codec: ""}}
	codec, err := selectCodec(context.Background(), rec)
	if err != nil {
		t.Fatalf("空 codec 不应出错: %v", err)
	}
	if codec != nil {
		t.Fatal("空 codec 应返回 nil")
	}
}

func TestDefaultSelector_CodecUnknown(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Codec: "xor"}}
	_, err := selectCodec(context.Background(), rec)
	if err == nil {
		t.Fatal("未知 codec 应返回错误")
	}
}

func TestDefaultSelector_EnvelopeMarkerNil(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Envelope: "marker"}}
	env, err := selectEnvelope(context.Background(), rec)
	if err != nil {
		t.Fatalf("marker envelope 不应出错: %v", err)
	}
	if env != nil {
		t.Fatal("marker envelope 在默认工厂里应返回 nil（不启用）")
	}
}

func TestDefaultSelector_EnvelopeEmpty(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Envelope: ""}}
	env, err := selectEnvelope(context.Background(), rec)
	if err != nil {
		t.Fatalf("空 envelope 不应出错: %v", err)
	}
	if env != nil {
		t.Fatal("空 envelope 应返回 nil（不启用）")
	}
}

func TestDefaultSelector_EnvelopeUnknown(t *testing.T) {
	rec := &NodeRecord{Config: NodeConfig{Envelope: "custom"}}
	_, err := selectEnvelope(context.Background(), rec)
	if err == nil {
		t.Fatal("未知 envelope 应返回错误")
	}
}

func TestDefaultBuilderFactory_CannotCreateClient(t *testing.T) {
	// 默认工厂无法构造 Client（transport 未接入）
	factory := DefaultClientFactory()
	rec := &NodeRecord{Config: NodeConfig{ID: "n1", Transport: "httpform"}}
	_, err := factory.NewClient(context.Background(), rec)
	if err == nil {
		t.Fatal("默认工厂构造 httpform Client 应失败")
	}
}

func TestDefaultBuilderFactory_CanCreateMockClient(t *testing.T) {
	// 空 Transport 走默认分支，仍然报错
	factory := DefaultClientFactory()
	rec := &NodeRecord{Config: NodeConfig{ID: "n1"}}
	_, err := factory.NewClient(context.Background(), rec)
	if err == nil {
		t.Fatal("默认工厂任何 transport 默认都应失败")
	}
}
