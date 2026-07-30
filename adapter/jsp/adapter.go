// Package jsp is the JSP (Java Servlet) adapter.
// It provides operations compatible with a JSP shell that dispatches
// based on an action parameter, returning structured output.
package jsp

import (
	"encoding/base64"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// Adapter assembles the JSP templates, parser, capabilities, and obfuscator.
type Adapter struct {
	templates *JSPTemplates
	parser    *Parser
	caps      *Capabilities
	obf       *Obfuscator
}

// New builds a JSP Adapter with default obfuscation.
func New() *Adapter {
	return &Adapter{
		templates: NewJSPTemplates(),
		parser:    NewParser(),
		caps:      NewCapabilities(),
		obf:       DefaultObfuscator(),
	}
}

// NewWithObfuscator builds a JSP Adapter with a custom obfuscator.
func NewWithObfuscator(obf *Obfuscator) *Adapter {
	return &Adapter{
		templates: NewJSPTemplates(),
		parser:    NewParser(),
		caps:      NewCapabilities(),
		obf:       obf,
	}
}

// Capabilities returns the adapter's declared capabilities.
func (a *Adapter) Capabilities() []core.Capability { return a.caps.All() }

// FillPlaceholders replaces runtime parameters in template code.
func (a *Adapter) FillPlaceholders(code string, params map[string][]byte) string {
	out := code
	for k, v := range params {
		encoded := base64.StdEncoding.EncodeToString(v)
		out = strings.ReplaceAll(out, "#{base64::"+string(k)+"}", encoded)
		out = strings.ReplaceAll(out, "#"+string(k), string(v))
	}
	return out
}

// Templates exposes the underlying templates package.
func (a *Adapter) Templates() *JSPTemplates { return a.templates }

// Parser exposes the parser for handler code.
func (a *Adapter) Parser() *Parser { return a.parser }

// Obfuscator returns the adapter's obfuscator.
func (a *Adapter) Obfuscator() *Obfuscator { return a.obf }
