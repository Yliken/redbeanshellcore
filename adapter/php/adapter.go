// Package php 是 AntSword / PHP 兼容适配器。它把通用的 ops.Operation
//  适配成 PHP 专属的 wire 请求。这个包里的任何东西都不算 SDK core——全部可替换。
package php

import (
	"encoding/base64"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// Adapter 把模板源、占位符替换器和 parser 组装到一起。
//  DefaultClientFactory 在看到 record.Config.Adapter == "php" 时会用它。
type Adapter struct {
	templates   *PHPTemplates
	parser      *Parser
	caps        *Capabilities
}

// New 构建一个 php Adapter。
func New() *Adapter {
	return &Adapter{
		templates: NewPHPTemplates(),
		parser:    NewParser(),
		caps:      NewCapabilities(),
	}
}

// Capabilities 返回适配器声明的能力。
func (a *Adapter) Capabilities() []core.Capability { return a.caps.All() }

// FillPlaceholders 把运行时参数替换进模板的占位符。
//  Python 里对应 AntSword._fill；这里从 req.Params 按 key 取值。
func (a *Adapter) FillPlaceholders(code string, params map[string][]byte) string {
	out := code
	for k, v := range params {
		encoded := base64.StdEncoding.EncodeToString(v)
		out = strings.ReplaceAll(out, "#{base64::"+string(k)+"}", encoded)
		out = strings.ReplaceAll(out, "#"+string(k), string(v))
	}
	return out
}

// Templates 暴露底层模板包。只有适配器该调它。
func (a *Adapter) Templates() *PHPTemplates { return a.templates }

// Parser 暴露 parser 给 handler 代码。
func (a *Adapter) Parser() *Parser { return a.parser }
