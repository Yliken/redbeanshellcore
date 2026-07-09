package ops

import (
	"context"
	"errors"

	"github.com/yliken/redbeanshellcore/core"
)

// FileWriteOperation 覆盖或追加写远端文件。
type FileWriteOperation struct {
	Path    string
	Content []byte
	Append  bool // true 表示追加，false 表示覆盖
}

// NewFileWrite 构建覆盖写的 FileWrite。
func NewFileWrite(path string, content []byte) *FileWriteOperation {
	return &FileWriteOperation{Path: path, Content: content}
}

// NewFileAppend 构建追加写的 FileWrite。
func NewFileAppend(path string, content []byte) *FileWriteOperation {
	return &FileWriteOperation{Path: path, Content: content, Append: true}
}

// Name 返回操作名。
func (op *FileWriteOperation) Name() string { return "file.write" }

// Build 生成带路径和 content 参数的请求。
func (op *FileWriteOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	req.SetParam("content", op.Content)
	if op.Append {
		req.SetMeta("mode", "a")
	} else {
		req.SetMeta("mode", "w")
	}
	return req, nil
}

// Parse 把远端的 "1"/"0" 转成 BoolResult。
func (op *FileWriteOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.write.Parse: 响应为空")
	}
	trimmed := string(resp.Body)
	ok := trimmed == "1" || trimmed == "ok" || trimmed == "OK"
	return &core.BoolResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		OK:         ok,
		Message:    trimmed,
	}, nil
}

// RiskLevel 把本操作归类为写。
func (op *FileWriteOperation) RiskLevel() core.RiskLevel { return core.RiskWrite }
