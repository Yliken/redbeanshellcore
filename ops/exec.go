package ops

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// ExecOperation 在远端跑一条 shell 命令，拿到 stdout/stderr。
type ExecOperation struct {
	Command string            // 要执行的 shell 命令
	Bin     string            // 默认 shell 路径，例如 /bin/sh 或 C:\Windows\system32\cmd.exe
	Env     map[string]string // 注入的环境变量（可选）
}

// NewExec 给命令用默认 /bin/sh 构建 Exec。
func NewExec(cmd string) *ExecOperation {
	return &ExecOperation{Command: cmd, Bin: "/bin/sh"}
}

// NewExecWithBin 指定替代 shell 二进制路径。
func NewExecWithBin(cmd, bin string) *ExecOperation {
	return &ExecOperation{Command: cmd, Bin: bin}
}

// Name 返回操作名。
func (op *ExecOperation) Name() string { return "exec" }

// Build 生成携带命令和环境变量的请求。
func (op *ExecOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.Payload = []byte(op.Command)
	if op.Bin != "" {
		req.SetParamString("bin", op.Bin)
	}
	if len(op.Env) > 0 {
		pairs := make([]string, 0, len(op.Env))
		for k, v := range op.Env {
			pairs = append(pairs, k+"="+v)
		}
		req.SetParamString("env", strings.Join(pairs, "|||asline|||"))
	}
	return req, nil
}

// Parse 把响应按行分类："STDERR://" 开头进 stderr，"ret=127" 开头取退出码，其余当作 stdout。
func (op *ExecOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("exec.Parse: 响应为空")
	}
	raw := string(resp.Body)
	var stderr []string
	var stdout []string
	var exitCode int
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "STDERR://") {
			stderr = append(stderr, strings.TrimPrefix(line, "STDERR://"))
			continue
		}
		if strings.HasPrefix(line, "ret=") {
			if c, err := strconv.Atoi(strings.TrimPrefix(line, "ret=")); err == nil {
				exitCode = c
			}
			continue
		}
		stdout = append(stdout, line)
	}
	return &core.ExecResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		Stdout:     strings.Join(stdout, "\n"),
		Stderr:     strings.Join(stderr, "\n"),
		ExitCode:   exitCode,
	}, nil
}

// RiskLevel 返回 "exec"，让 middleware 自由决定是否拦截。
func (op *ExecOperation) RiskLevel() core.RiskLevel { return core.RiskExec }
