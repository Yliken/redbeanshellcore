package ops

import (
	"context"
	"errors"

	"github.com/yliken/redbeanshellcore/core"
)

// FileDownloadOperation 把远端文件读到内存（二进制安全）。
type FileDownloadOperation struct {
	Path string
}

// NewFileDownload 构建一个 FileDownload 操作。
func NewFileDownload(path string) *FileDownloadOperation {
	return &FileDownloadOperation{Path: path}
}

// Name 返回操作名。
func (op *FileDownloadOperation) Name() string { return "file.download" }

// Build 生成带路径参数的请求。
func (op *FileDownloadOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	return req, nil
}

// Parse 把响应原样放进 FileReadResult.Data。二进制安全。
func (op *FileDownloadOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.download.Parse: 响应为空")
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
func (op *FileDownloadOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }
