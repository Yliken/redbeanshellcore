// Package mock 提供纯 Go 的假远端适配器，用于测试和示例。
//  它满足 core.Operation 的方式是：把底层 Transport 返回的字节原样回吐，
//  不经过任何真实 PHP 运行时。配合 transport/mock 可以搭出完全自包含的 demo。
package mock

import (
	"context"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Adapter 是 mock 适配器。故意做成无状态——"远端"状态由 mock Transport 的 handler 持有。
type Adapter struct{}

// New 构建一个 mock Adapter。
func New() *Adapter { return &Adapter{} }

// Capabilities 返回 mock 节点支持的全部能力。
func (a *Adapter) Capabilities() []core.Capability {
	return []core.Capability{
		core.CapInfo,
		core.CapExec,
		core.CapFileList,
		core.CapFileRead,
		core.CapFileWrite,
		core.CapFileDelete,
		core.CapFileUpload,
	}
}

// HasCapability 判断是否支持给定能力。
func (a *Adapter) HasCapability(c core.Capability) bool {
	for _, have := range a.Capabilities() {
		if have == c {
			return true
		}
	}
	return false
}

// CheckCapabilities 校验 op 需要的能力是否都被满足。
func (a *Adapter) CheckCapabilities(op core.Operation) error {
	if aware, ok := op.(core.CapabilityAware); ok {
		for _, cap := range aware.RequiredCapabilities() {
			if !a.HasCapability(cap) {
				return fmt.Errorf("mock adapter 缺少能力 %q", cap)
			}
		}
	}
	return nil
}

// BuildInfo 是示例 / 测试用的占位。
func (a *Adapter) BuildInfo(_ context.Context, op core.Operation, sess *core.Session) (*core.Request, error) {
	_ = op
	_ = sess
	return core.NewRequest("info"), nil
}
