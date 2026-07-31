package aspx

import "github.com/Yliken/redbeanshellcore/core"

type Capabilities struct{}

func NewCapabilities() *Capabilities { return &Capabilities{} }

func (c *Capabilities) All() []core.Capability {
	return []core.Capability{core.CapInfo, core.CapExec, core.CapFileList, core.CapFileRead, core.CapFileUpload}
}

func (c *Capabilities) HasCapability(cap core.Capability) bool {
	for _, have := range c.All() { if have == cap { return true } }; return false
}
