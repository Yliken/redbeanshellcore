package ops

import (
	"context"
	"errors"

	"github.com/Yliken/redbeanshellcore/core"
)

// FileDeleteOperation 删除远端文件 / 目录。
type FileDeleteOperation struct {
	Path string
}

// NewFileDelete 构建一个 FileDelete 操作。
func NewFileDelete(path string) *FileDeleteOperation {
	return &FileDeleteOperation{Path: path}
}

// Name 返回操作名。
func (op *FileDeleteOperation) Name() string { return "file.delete" }

// Build 生成带路径参数的请求。
func (op *FileDeleteOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	return req, nil
}

// Parse 把远端的 "1"/"0" 转成 BoolResult。
func (op *FileDeleteOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.delete.Parse: 响应为空")
	}
	trimmed := string(resp.Body)
	ok := trimmed == "1" || trimmed == "ok"
	return &core.BoolResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		OK:         ok,
		Message:    trimmed,
	}, nil
}

// RiskLevel 把本操作归类为破坏性。
func (op *FileDeleteOperation) RiskLevel() core.RiskLevel { return core.RiskDestructive }
