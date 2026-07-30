package jsp

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/ops"
)

// ClientFactory creates JSP-adapted clients from node records.
type ClientFactory struct {
	ApplyTemplate bool
	Obfuscator    *Obfuscator // optional: shared obfuscation state
}

func NewClientFactory() *ClientFactory {
	return &ClientFactory{ApplyTemplate: true}
}

// WithObfuscator sets the obfuscation to use for wrapped operations.
func (f *ClientFactory) WithObfuscator(obf *Obfuscator) *ClientFactory {
	f.Obfuscator = obf
	return f
}

func (f *ClientFactory) NewClient(_ context.Context, rec *core.NodeRecord) (*core.Client, error) {
	return nil, fmt.Errorf("jsp.ClientFactory: JSP adapter does not provide a default client builder; " +
		"use core.NewClient with a configured httpform transport and session")
}

// WrapOp translates generic operations into JSP-specific operations.
func (f *ClientFactory) WrapOp(operation core.Operation) (core.Operation, error) {
	if isNilOperation(operation) {
		return nil, fmt.Errorf("jsp.ClientFactory.WrapOp: operation cannot be nil")
	}
	if !f.ApplyTemplate {
		return operation, nil
	}
	switch op := operation.(type) {
	case *jspInfo, *jspFileList, *jspFileRead, *jspFileDownload, *jspFileUpload, *jspExec:
		return op, nil
	case *ops.InfoOperation:
		wrapped := NewJspInfo()
		if f.Obfuscator != nil {
			wrapped.WithObfuscator(f.Obfuscator)
		}
		return wrapped, nil
	case *ops.FileListOperation:
		wrapped := NewJspFileList(op.Path)
		if f.Obfuscator != nil {
			wrapped.WithObfuscator(f.Obfuscator)
		}
		return wrapped, nil
	case *ops.FileReadOperation:
		wrapped := NewJspFileRead(op.Path)
		if f.Obfuscator != nil {
			wrapped.WithObfuscator(f.Obfuscator)
		}
		return wrapped, nil
	case *ops.FileDownloadOperation:
		wrapped := NewJspFileDownload(op.Path)
		if f.Obfuscator != nil {
			wrapped.WithObfuscator(f.Obfuscator)
		}
		return wrapped, nil
	case *ops.ExecOperation:
		translated := NewJspExec(op.Command).WithBin(op.Bin)
		for key, value := range copyMap(op.Env) {
			translated.WithEnv(key, value)
		}
		if f.Obfuscator != nil {
			translated.WithObfuscator(f.Obfuscator)
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
