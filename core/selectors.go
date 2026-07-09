package core

import (
	"context"
	"errors"
	"fmt"
)

// selectTransport 根据 record.Config.Transport 挑一种传输实现。
//  目前只内置了 "" / "httpform" / "mock" 三者，其他都要调用方自己接。
func selectTransport(ctx context.Context, record *NodeRecord) (Transport, error) {
	switch record.Config.Transport {
	case "", "httpform":
		return nil, errors.New("remote-node-core: httpform 传输未接入；请提供自定义 ClientFactory")
	case "mock":
		return nil, errors.New("remote-node-core: mock transport 请用 transport/mock.New 构造")
	default:
		return nil, fmt.Errorf("remote-node-core: 未知的 transport 类型 %q", record.Config.Transport)
	}
}

func selectCodec(ctx context.Context, record *NodeRecord) (Codec, error) {
	if record.Config.Codec == "" || record.Config.Codec == "plain" {
		return nil, nil // plain 就是默认无变换
	}
	return nil, fmt.Errorf("remote-node-core: 未知的 codec 类型 %q", record.Config.Codec)
}

func selectEnvelope(ctx context.Context, record *NodeRecord) (Envelope, error) {
	if record.Config.Envelope == "" || record.Config.Envelope == "marker" {
		return nil, nil // marker 默认使用无变换版本；需要显式调用方介入
	}
	return nil, fmt.Errorf("remote-node-core: 未知的 envelope 类型 %q", record.Config.Envelope)
}
