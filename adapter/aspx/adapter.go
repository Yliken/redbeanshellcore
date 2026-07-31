package aspx

import (
	"encoding/base64"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

type Adapter struct {
	templates *ASPXTemplates
	parser    *Parser
	caps      *Capabilities
}

func New() *Adapter { return &Adapter{templates: NewASPXTemplates(), parser: NewParser(), caps: NewCapabilities()} }
func (a *Adapter) Capabilities() []core.Capability { return a.caps.All() }
func (a *Adapter) Templates() *ASPXTemplates { return a.templates }
func (a *Adapter) Parser() *Parser { return a.parser }

func (a *Adapter) FillPlaceholders(code string, params map[string][]byte) string {
	out := code
	for k, v := range params {
		encoded := base64.StdEncoding.EncodeToString(v)
		out = strings.ReplaceAll(out, "#{base64::"+string(k)+"}", encoded)
		out = strings.ReplaceAll(out, "#"+string(k), string(v))
	}
	return out
}

