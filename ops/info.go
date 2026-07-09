package ops

import (
	"context"
	"errors"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// InfoOperation 拉取远端节点的 OS / 运行时元数据。
type InfoOperation struct{}

// NewInfo 构建一个 Info 操作。
func NewInfo() *InfoOperation { return &InfoOperation{} }

// Name 返回操作名。
func (op *InfoOperation) Name() string { return "info" }

// Build 返回一个携带 adapter-defined info 模板的请求。
//  payload 有意保持不透明：调用方只选 op，adapter 负责"info" 最终在 wire 上的形态。
func (op *InfoOperation) Build(_ context.Context, sess *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.Payload = []byte(op.Name())
	req.SetMeta("adapter_meta_key", "info")
	_ = sess
	return req, nil
}

// Parse 把响应解码成 InfoResult。Python demo 返回单条 tab 分隔字符串：
//  {workdir}\t{drives}\t{os_info}\t{user}。更丰富的 adapter 可以在上面再包一层自定义 Parse。
func (op *InfoOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("info.Parse: 响应为空")
	}
	raw := string(resp.Body)
	parts := strings.Split(raw, "\t")
	res := &core.InfoResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
	}
	switch {
	case len(parts) >= 4:
		res.Workdir = parts[0]
		// parts[1] = 可用驱动器（typed 形式舍弃，但 Raw 保留）
		res.OS = parts[2]
		res.User = parts[3]
	case len(parts) >= 1:
		res.OS = parts[0]
	}
	return res, nil
}

// RequiredCapabilities 声明本操作需要的能力。
func (op *InfoOperation) RequiredCapabilities() []core.Capability {
	return []core.Capability{core.CapInfo}
}

// RiskLevel 把本操作归类为只读。
func (op *InfoOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }
