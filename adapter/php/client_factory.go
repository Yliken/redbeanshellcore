package php

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/ops"
	"github.com/Yliken/redbeanshellcore/transport/httpform"
)

// ClientFactory 是为 PHP Shell 定制的 ClientFactory。
// 它根据节点配置组装 HTTP Form Client。调用方应使用 PHP 专属 Operation；
// WrapOp 可显式转换部分通用 Info/FileList/FileRead/FileDownload/Exec 操作。
type ClientFactory struct {
	// ApplyTemplate 用来把 ops.New*() 替换成 PHP 专属版本。
	ApplyTemplate bool
}

// NewClientFactory 构建一个默认开启 ApplyTemplate 的 PHP 客户端工厂。
func NewClientFactory() *ClientFactory { return &ClientFactory{ApplyTemplate: true} }

// NewClient 根据 NodeRecord 组装一个可用的 Client。
func (f *ClientFactory) NewClient(_ context.Context, rec *core.NodeRecord) (*core.Client, error) {
	if rec == nil {
		return nil, fmt.Errorf("php.ClientFactory: node record 不能为空")
	}
	if rec.Config.Endpoint == "" {
		return nil, fmt.Errorf("php.ClientFactory: 节点 %q 没有 endpoint", rec.Config.ID)
	}

	tr, err := f.buildTransport(rec)
	if err != nil {
		return nil, err
	}

	codec, err := f.buildCodec(rec)
	if err != nil {
		return nil, err
	}

	sess := &core.Session{
		NodeID:       rec.Config.ID,
		Endpoint:     rec.Config.Endpoint,
		Adapter:      rec.Config.Adapter,
		Transport:    rec.Config.Transport,
		Codec:        rec.Config.Codec,
		Capabilities: append([]core.Capability(nil), rec.Capabilities...),
		Metadata:     copyMap(rec.Metadata),
	}
	if rec.Config.Auth != nil {
		for k, v := range rec.Config.Auth {
			sess.Metadata[k] = v
		}
	}

	return core.NewClient(
		core.WithSession(sess),
		core.WithTransport(tr),
		core.WithCodec(codec),
	), nil
}

func (f *ClientFactory) buildTransport(rec *core.NodeRecord) (core.Transport, error) {
	switch rec.Config.Transport {
	case "", "httpform":
		tr := httpform.New(rec.Config.Endpoint)
		tr.Timeout = 30 * time.Second
		if rec.Config.Options["insecure_tls"] == "true" {
			tr.InsecureTLS = true
		}
		return tr, nil
	default:
		return nil, fmt.Errorf("php.ClientFactory: 不支持的 transport %q", rec.Config.Transport)
	}
}

func (f *ClientFactory) buildCodec(rec *core.NodeRecord) (core.Codec, error) {
	switch rec.Config.Codec {
	case "", "plain":
		return nil, nil
	default:
		return nil, fmt.Errorf("php.ClientFactory: 暂不支持 codec %q", rec.Config.Codec)
	}
}

// WrapOp 显式把已知通用 Operation 翻译成语义等价的 PHP 专属版本。
// 已经是 PHP 版本或未知自定义类型时原样返回；该方法不会被 Client 自动调用。
// 通用 FileUpload 持有 reader，转换会产生隐式消费，因此必须使用 NewPhpFileUpload。
func (f *ClientFactory) WrapOp(operation core.Operation) (core.Operation, error) {
	if isNilOperation(operation) {
		return nil, fmt.Errorf("php.ClientFactory.WrapOp: operation 不能为空")
	}
	if !f.ApplyTemplate {
		return operation, nil
	}
	switch operation := operation.(type) {
	case *phpInfo, *phpFileList, *phpFileRead, *phpFileDownload, *phpFileUpload, *phpExec:
		return operation, nil
	case *ops.InfoOperation:
		return NewPhpInfo(), nil
	case *ops.FileListOperation:
		return NewPhpFileList(operation.Path), nil
	case *ops.FileReadOperation:
		return NewPhpFileRead(operation.Path), nil
	case *ops.FileDownloadOperation:
		return NewPhpFileDownload(operation.Path), nil
	case *ops.ExecOperation:
		translated := NewPhpExec(operation.Command).WithBin(operation.Bin)
		for key, value := range copyMap(operation.Env) {
			translated.WithEnv(key, value)
		}
		return translated, nil
	default:
		return operation, nil
	}
}

func isNilOperation(operation core.Operation) bool {
	if operation == nil {
		return true
	}
	value := reflect.ValueOf(operation)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func copyMap(in map[string]string) map[string]string {
	if in == nil {
		return make(map[string]string)
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
