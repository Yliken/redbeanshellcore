package php

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strconv"
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
	), nil
}
func (f *ClientFactory) buildTransport(rec *core.NodeRecord) (*httpform.Transport, error) {
	opts := httpform.DefaultOptions()
	opts.Timeout = 30 * time.Second

	// 解析扩展选项
	if rec.Config.Options != nil {
		if v, ok := rec.Config.Options["insecure_tls"]; ok && v == "true" {
			opts.InsecureTLS = true
		}
		if v, ok := rec.Config.Options["timeout"]; ok {
			if d, err := time.ParseDuration(v); err == nil {
				opts.Timeout = d
			}
		}

		// P0.1 — UA 轮换
		if v, ok := rec.Config.Options["ua_rotation"]; ok && v == "true" {
			opts.UARotation = true
			opts.UAPool = nil // 将使用默认池
		}

		// P0.3 — 动态字段名
		if v, ok := rec.Config.Options["dynamic_fields"]; ok && v == "true" {
			opts.DynamicFieldNames = true
			opts.FieldGen = httpform.NewFieldGenerator()
		}

		// P0.5 — 诱饵字段填充
		if v, ok := rec.Config.Options["honeypot_count"]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts.EnablePadding = true
				opts.HoneypotCount = n
			}
		}

		// P0.10 代理配置
		if v, ok := rec.Config.Options["proxy"]; ok && v != "" {
			host, portStr, err := net.SplitHostPort(v)
			if err == nil {
				port, portErr := strconv.Atoi(portStr)
				if portErr == nil {
					opts.ProxyChain = []httpform.ProxyConfig{
						{Type: httpform.ProxyHTTP, Host: host, Port: port},
					}
				}
			}
		}

		// P0.6 — TLS 指纹随机化
		if v, ok := rec.Config.Options["tls_fingerprint"]; ok && v == "true" {
			opts.TLSFingerprint.Enabled = true
		}

		// P0.8 — 多协议协商
		if v, ok := rec.Config.Options["http_protocol"]; ok {
			switch v {
			case "http1.1":
				opts.Protocol = httpform.ProtocolHTTP11
			case "http2":
				opts.Protocol = httpform.ProtocolHTTP2
			case "http3":
				opts.Protocol = httpform.ProtocolHTTP3
			}
		}

		// P0.9 — 连接池配置
		if v, ok := rec.Config.Options["max_idle_conns"]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts.MaxIdleConns = n
			}
		}
		if v, ok := rec.Config.Options["max_idle_per_host"]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts.MaxIdleConnsPerHost = n
			}
		}
		if v, ok := rec.Config.Options["cookie_jar"]; ok && v == "false" {
			opts.EnableCookieJar = false
		}
	}

	switch rec.Config.Transport {
	case "", "httpform":
		return httpform.NewWithOptions(rec.Config.Endpoint, opts), nil
	default:
		return nil, fmt.Errorf("php.ClientFactory: 不支持的 transport %q", rec.Config.Transport)
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
