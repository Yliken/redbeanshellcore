package ops

import (
	"context"
	"errors"
	"io"

	"github.com/yliken/redbeanshellcore/core"
)

// FileUploadOperation 把本地 reader 的数据上传到远端路径。
type FileUploadOperation struct {
	RemotePath string
	Reader     io.Reader
	Append     bool // 是否追加
	ChunkSize  int  // 分片大小（当前实现一次读完，保留为未来接口）
}

// NewFileUpload 用 io.Reader 构建一个 FileUpload 操作。
func NewFileUpload(remotePath string, r io.Reader) *FileUploadOperation {
	return &FileUploadOperation{RemotePath: remotePath, Reader: r, ChunkSize: 64 * 1024}
}

// Name 返回操作名。
func (op *FileUploadOperation) Name() string { return "file.upload" }

// Build 把 reader 全部读进内存后塞进 Request。
// 未来适配器要支持 chunks 的话可以改这里，或者新加 stream 接口。
func (op *FileUploadOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	if op.Reader == nil {
		return nil, errors.New("file.upload.Build: reader 不能为空")
	}
	data, err := io.ReadAll(op.Reader)
	if err != nil {
		return nil, err
	}
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.RemotePath)
	req.SetParam("content", data)
	if op.Append {
		req.SetMeta("mode", "a")
	} else {
		req.SetMeta("mode", "w")
	}
	req.SetMeta("chunk_size", "full")
	return req, nil
}

// Parse 把远端的 "1"/"0" 转成 BoolResult。
func (op *FileUploadOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.upload.Parse: 响应为空")
	}
	trimmed := string(resp.Body)
	ok := trimmed == "1" || trimmed == "ok"
	return &core.BoolResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		OK:         ok,
		Message:    trimmed,
	}, nil
}

// RiskLevel 把本操作归类为写。
func (op *FileUploadOperation) RiskLevel() core.RiskLevel { return core.RiskWrite }
