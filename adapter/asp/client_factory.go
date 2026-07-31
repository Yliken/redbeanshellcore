package asp

import (
	"context"
	"fmt"
	"io"
	"reflect"

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
		return nil, fmt.Errorf("asp.ClientFactory: node record cannot be nil")
	}
	if rec.Config.Endpoint == "" {
		return nil, fmt.Errorf("asp.ClientFactory: node %q has no endpoint", rec.Config.ID)
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
	if v, ok := rec.Config.Options["hmac_key"]; ok && v != "" {
		sess.Metadata["hmac_key"] = v
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
	opts, wireProto := httpform.ParseTransportOptions(rec)
	switch rec.Config.Transport {
	case "", "httpform":
		return httpform.NewWithOptions(rec.Config.Endpoint, opts), wireProto, nil
	default:
		return nil, false, fmt.Errorf("asp.ClientFactory: unsupported transport %q", rec.Config.Transport)
	}
}

func (f *ClientFactory) WrapOp(operation core.Operation) (core.Operation, error) {
	if !f.ApplyTemplate {
		return operation, nil
	}
	if operation == nil || isNil(operation) {
		return nil, fmt.Errorf("asp.ClientFactory.WrapOp: operation cannot be nil")
	}
	switch op := operation.(type) {
	case *aspInfo, *aspExec, *aspFileList, *aspFileRead, *aspFileDownload, *aspFileUpload:
		return op, nil
	case *ops.InfoOperation:
		return NewAspInfo(), nil
	case *ops.FileListOperation:
		return NewAspFileList(op.Path), nil
	case *ops.FileReadOperation:
		return NewAspFileRead(op.Path), nil
	case *ops.FileDownloadOperation:
		return NewAspFileDownload(op.Path), nil
	case *ops.ExecOperation:
		translated := NewAspExec(op.Command).WithBin(op.Bin)
		for key, value := range copyMap(op.Env) {
			translated.WithEnv(key, value)
		}
		return translated, nil
	case *ops.FileUploadOperation:
		data, err := io.ReadAll(op.Reader)
		if err != nil {
			return nil, fmt.Errorf("asp.ClientFactory: read upload data: %w", err)
		}
		return NewAspFileUpload(op.RemotePath, data), nil
	default:
		return operation, nil
	}
}

func isNil(op core.Operation) bool {
	if op == nil {
		return true
	}
	v := reflect.ValueOf(op)
	return v.Kind() == reflect.Pointer && v.IsNil()
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
