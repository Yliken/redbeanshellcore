package ops

import (
	"context"
	"errors"

	"github.com/yliken/redbeanshellcore/core"
)

// FileRenameOperation 重命名 / 移动远端文件。
type FileRenameOperation struct {
	Src string
	Dst string
}

// NewFileRename 构建一个 FileRename 操作。
func NewFileRename(src, dst string) *FileRenameOperation {
	return &FileRenameOperation{Src: src, Dst: dst}
}

// Name 返回操作名。
func (op *FileRenameOperation) Name() string { return "file.rename" }

// Build 生成带 src + dst 两个参数的请求。
func (op *FileRenameOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("src", op.Src)
	req.SetParamString("dst", op.Dst)
	return req, nil
}

// Parse 把远端的 "1"/"0" 转成 BoolResult。
func (op *FileRenameOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.rename.Parse: 响应为空")
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
func (op *FileRenameOperation) RiskLevel() core.RiskLevel { return core.RiskWrite }
