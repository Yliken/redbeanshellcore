package php

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/crypto/aesgcm"
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

	bodyCrypto, err := f.buildBodyCrypto(rec)
	if err != nil {
		return nil, err
	}
	if bodyCrypto != nil {
		if wireProto {
			return nil, fmt.Errorf("php.ClientFactory: crypto body 与 wire protocol 互斥")
		}
		tr.Options.BodyCrypto = bodyCrypto
		if v := rec.Config.Options["crypto_field"]; v != "" {
			tr.Options.CryptoField = v
		}
	}

	adapterName := rec.Config.Adapter
	if bodyCrypto != nil && (adapterName == "" || adapterName == "php") {
		// Body crypto targets the eval-style shell, whose profile is php-eval.
		adapterName = "php-eval"
	}
	sess := &core.Session{
		NodeID:       rec.Config.ID,
		Endpoint:     rec.Config.Endpoint,
		Adapter:      adapterName,
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
	if v, ok := rec.Config.Options["hmac_key"]; ok && v != "" {
		sess.Metadata["hmac_key"] = v
	}
	if bodyCrypto != nil {
		if v := rec.Config.Options["crypto_mode"]; v != "" {
			sess.Metadata["crypto_mode"] = v
		}
	}

	if wireProto {
		opts := []core.Option{
			core.WithSession(sess),
			core.WithTransport(tr),
			core.WithEnvelope(marker.NewWithWire()),
		}
		return core.NewClient(opts...), nil
	}
	opts := []core.Option{
		core.WithSession(sess),
		core.WithTransport(tr),
	}
	if bodyCrypto != nil {
		opts = append(opts, core.WithBodyCrypto(bodyCrypto))
	}
	return core.NewClient(opts...), nil
}

// buildBodyCrypto reads body crypto configuration from node options and
// validates the crypto mode and shell key fingerprint when both are declared.
// Supported options:
//
//	crypto_key_hex        hex-encoded AES key (preferred)
//	crypto_key            raw key bytes (fallback)
//	crypto_field          POST field carrying the encrypted body (default __crypto)
//	crypto_mode           expected shell crypto mode
//	shell_key_fingerprint expected shell key fingerprint
func (f *ClientFactory) buildBodyCrypto(rec *core.NodeRecord) (core.BodyCrypto, error) {
	if rec == nil || rec.Config.Options == nil {
		return nil, nil
	}
	var key []byte
	if keyHex := rec.Config.Options["crypto_key_hex"]; keyHex != "" {
		decoded, err := hex.DecodeString(keyHex)
		if err != nil {
			return nil, fmt.Errorf("php.ClientFactory: crypto_key_hex 必须是 hex 编码: %w", err)
		}
		key = decoded
	} else if raw := rec.Config.Options["crypto_key"]; raw != "" {
		key = []byte(raw)
	}
	if len(key) == 0 {
		return nil, nil
	}

	cr, err := aesgcm.New(key)
	if err != nil {
		return nil, fmt.Errorf("php.ClientFactory: crypto_key 无效: %w", err)
	}
	mode, fingerprint := CryptoShellMeta(key)
	if modeOpt := rec.Config.Options["crypto_mode"]; modeOpt != "" && modeOpt != mode {
		return nil, fmt.Errorf("php.ClientFactory: crypto mode 不匹配: shell=%s client=%s", modeOpt, mode)
	}
	if fpOpt := rec.Config.Options["shell_key_fingerprint"]; fpOpt != "" && fpOpt != fingerprint {
		return nil, fmt.Errorf("php.ClientFactory: shell key 指纹不匹配: shell=%s client=%s", fpOpt, fingerprint)
	}
	return cr, nil
}

func (f *ClientFactory) buildTransport(rec *core.NodeRecord) (*httpform.Transport, bool, error) {
	opts, wireProto := httpform.ParseTransportOptions(rec)
	switch rec.Config.Transport {
	case "", "httpform":
		return httpform.NewWithOptions(rec.Config.Endpoint, opts), wireProto, nil
	default:
		return nil, false, fmt.Errorf("php.ClientFactory: unsupported transport %q", rec.Config.Transport)
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
