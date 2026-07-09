package ops

import (
	"context"
	"errors"

	"github.com/Yliken/redbeanshellcore/core"
)

// FileReadOperation 把远端文件原样读到内存。
type FileReadOperation struct {
	Path string
}

// NewFileRead 构建一个 FileRead 操作。
func NewFileRead(path string) *FileReadOperation {
	return &FileReadOperation{Path: path}
}

// Name 返回操作名。
func (op *FileReadOperation) Name() string { return "file.read" }

// Build 生成带路径参数的请求。
func (op *FileReadOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	return req, nil
}

// Parse 把响应原样放进 FileReadResult.Data。二进制安全，不做任何 transcode。
func (op *FileReadOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.read.Parse: 响应为空")
	}
	data := make([]byte, len(resp.Body))
	copy(data, resp.Body)
	return &core.FileReadResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		Path:       op.Path,
		Data:       data,
	}, nil
}

// RiskLevel 把本操作归类为只读。
func (op *FileReadOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }
