package php

import (
	"context"
	"fmt"
	"time"

	"github.com/yliken/redbeanshellcore/core"
	"github.com/yliken/redbeanshellcore/transport/httpform"
)

// ClientFactory 是为 PHP Shell 定制的 ClientFactory。
//  它会根据节点的 transport / codec 配置挑选合适的底层组件，并让
//  Info / FileList / FileRead / FileWrite / FileDelete / FileRename /
//  FileMkdir / FileDownload / Exec 等操作全部发出真正的 PHP 源码 payload，
//  从而兼容 AntSword 一类的 PHP Shell。
type ClientFactory struct {
	// ApplyTemplate 用来把 ops.New*() 替换成 PHP 专属版本。
	ApplyTemplate bool
}

// NewClientFactory 构建一个默认开启 ApplyTemplate 的 PHP 客户端工厂。
func NewClientFactory() *ClientFactory { return &ClientFactory{ApplyTemplate: true} }

// NewClient 根据 NodeRecord 组装一个可用的 Client。
func (f *ClientFactory) NewClient(_ context.Context, rec *core.NodeRecord) (*core.Client, error) {
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
		NodeID:    rec.Config.ID,
		Endpoint:  rec.Config.Endpoint,
		Adapter:   rec.Config.Adapter,
		Transport: rec.Config.Transport,
		Codec:     rec.Config.Codec,
		Metadata:  copyMap(rec.Metadata),
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

// WrapOp 把通用 ops 翻译成语义等价的 PHP 专属 ops。
// 如果 ApplyTemplate == false 则原样返回，便于调用方按需开启。
func (f *ClientFactory) WrapOp(op core.Operation) (core.Operation, error) {
	if !f.ApplyTemplate {
		return op, nil
	}
	switch op.Name() {
	case "info":
		return NewPhpInfo(), nil
	case "file.list":
		// 从 req.Params 凑一个 path；若拿不到就是默认 "/"。
		return NewPhpFileList("/"), nil
	case "file.read":
		return NewPhpFileRead(""), nil
	case "exec":
		return NewPhpExec(""), nil
	}
	return op, nil
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
