package jsp

import "github.com/Yliken/redbeanshellcore/core"

// Capabilities 声明 jsp 适配器支持的能力。
type Capabilities struct{}

// NewCapabilities 构建一个 jsp Capabilities 实例。
func NewCapabilities() *Capabilities { return &Capabilities{} }

// All 返回本适配器能服务的全部能力。
func (c *Capabilities) All() []core.Capability {
	return []core.Capability{
		core.CapInfo,
		core.CapExec,
		core.CapFileList,
		core.CapFileRead,
	}
}

// HasCapability 让 middleware / Manager 可以统一做运行时检查。
func (c *Capabilities) HasCapability(cap core.Capability) bool {
	for _, have := range c.All() {
		if have == cap {
			return true
		}
	}
	return false
}
