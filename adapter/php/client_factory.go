package php

import (
	"context"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/envelope/marker"
	"github.com/Yliken/redbeanshellcore/ops"
	"github.com/Yliken/redbeanshellcore/transport/httpform"
)

type ClientFactory struct {
	ApplyTemplate bool
}

func NewClientFactory() *ClientFactory { return &ClientFactory{ApplyTemplate: true} }

func (f *ClientFactory) NewClient(_ context.Context, rec *core.NodeRecord) (*core.Client, error) {
	if rec == nil {
		return nil, fmt.Errorf("php.ClientFactory: node record 不能为空")
	}
	if rec.Config.Endpoint == "" {
		return nil, fmt.Errorf("php.ClientFactory: 节点 %q 没有 endpoint", rec.Config.ID)
	}

	tr, wireProto, err := f.buildTransport(rec)
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

	if wireProto {
		return core.NewClient(
			core.WithSession(sess),
			core.WithTransport(tr),
			core.WithEnvelope(marker.NewWithWire()),
		), nil
	}
	return core.NewClient(
		core.WithSession(sess),
		core.WithTransport(tr),
	), nil
}

func (f *ClientFactory) buildTransport(rec *core.NodeRecord) (*httpform.Transport, bool, error) {
	opts := httpform.DefaultOptions()
	opts.Timeout = 30 * time.Second
	wireProto := false

	if rec.Config.Options != nil {
		if v, ok := rec.Config.Options["insecure_tls"]; ok && v == "true" {
			opts.InsecureTLS = true
		}
		if v, ok := rec.Config.Options["timeout"]; ok {
			if d, err := time.ParseDuration(v); err == nil {
				opts.Timeout = d
			}
		}
		if v, ok := rec.Config.Options["ua_rotation"]; ok && v == "true" {
			opts.UARotation = true
			opts.UAPool = nil
		}
		if v, ok := rec.Config.Options["dynamic_fields"]; ok && v == "true" {
			opts.DynamicFieldNames = true
			opts.FieldGen = httpform.NewFieldGenerator()
		}
		if v, ok := rec.Config.Options["honeypot_count"]; ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				opts.EnablePadding = true
				opts.HoneypotCount = n
			}
		}
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
		if v, ok := rec.Config.Options["tls_fingerprint"]; ok && v == "true" {
			opts.TLSFingerprint.Enabled = true
		}
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
		if v, ok := rec.Config.Options["wire_protocol"]; ok && v == "true" {
			opts.WireProtocol = true
			wireProto = true
		}
	}

	switch rec.Config.Transport {
	case "", "httpform":
		return httpform.NewWithOptions(rec.Config.Endpoint, opts), wireProto, nil
	default:
		return nil, false, fmt.Errorf("php.ClientFactory: 不支持的 transport %q", rec.Config.Transport)
	}
}

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
	case *ops.FileUploadOperation:
		data, err := io.ReadAll(operation.Reader)
		if err != nil {
			return nil, fmt.Errorf("php.ClientFactory: read upload data: %w", err)
		}
		return NewPhpFileUpload(operation.RemotePath, data), nil
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
