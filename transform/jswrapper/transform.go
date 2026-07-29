// Package jswrapper 提供将响应正文包裹在 JavaScript 变量赋值中的 Transform。
package jswrapper

import (
	"context"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Transform 将响应正文包裹在 JS 变量赋值中。
type Transform struct {
	varName   string
	wrapRound bool
}

// New 构建一个 JS 变量包裹 Transform。
func New(varName string) *Transform {
	return &Transform{
		varName:   varName,
		wrapRound: false,
	}
}

func (t *Transform) Name() string { return "jswrapper" }

func (t *Transform) Direction() core.TransformDirection {
	if t.wrapRound {
		return core.TransformBoth
	}
	return core.TransformResponse
}

func (t *Transform) ApplyRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("jswrapper: request 为空")
	}
	return req, nil
}

func (t *Transform) ApplyResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("jswrapper: response 为空")
	}
	varName := t.varName
	if varName == "" {
		varName = "data_" + randHex(6)
	}
	result := make([]byte, 0, len(resp.Body)+len(varName)+10)
	result = append(result, []byte("var ")...)
	result = append(result, []byte(varName)...)
	result = append(result, []byte(" = '")...)
	for _, b := range resp.Body {
		if b == '\'' || b == '\\' {
			result = append(result, '\\')
		}
		result = append(result, b)
	}
	result = append(result, []byte("';\n")...)
	resp.Body = result
	return resp, nil
}

func randHex(n int) string {
	const h = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = h[(i*7+3)%16]
	}
	return string(b)
}
