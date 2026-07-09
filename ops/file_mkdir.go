package ops

import (
	"context"
	"errors"

	"github.com/yliken/redbeanshellcore/core"
)

// FileMkdirOperation 在远端创建目录。
type FileMkdirOperation struct {
	Path string
}

// NewFileMkdir 构建一个 FileMkdir 操作。
func NewFileMkdir(path string) *FileMkdirOperation {
	return &FileMkdirOperation{Path: path}
}

// Name 返回操作名。
func (op *FileMkdirOperation) Name() string { return "file.mkdir" }

// Build 生成带路径参数的请求。
func (op *FileMkdirOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	return req, nil
}

// Parse 把远端的 "1"/"0" 转成 BoolResult。
func (op *FileMkdirOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.mkdir.Parse: 响应为空")
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
func (op *FileMkdirOperation) RiskLevel() core.RiskLevel { return core.RiskWrite }
