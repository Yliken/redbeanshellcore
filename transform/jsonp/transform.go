// Package jsonp 提供将响应正文包裹在 JSONP 回调中的 Transform。
package jsonp

import (
	"context"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Transform 将响应正文包裹在 JSONP 回调函数调用中。
type Transform struct {
	callback  string
	wrapRound bool
}

// New 构建一个 JSONP Transform。
// callback 为回调函数名（如 "callback"、"jsonp"），为空时每次随机生成。
func New(callback string) *Transform {
	return &Transform{
		callback:  callback,
		wrapRound: false,
	}
}

func (t *Transform) Name() string { return "jsonp" }

func (t *Transform) Direction() core.TransformDirection {
	if t.wrapRound {
		return core.TransformBoth
	}
	return core.TransformResponse
}

func (t *Transform) ApplyRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("jsonp: request 为空")
	}
	return req, nil
}

func (t *Transform) ApplyResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("jsonp: response 为空")
	}
	cb := t.callback
	if cb == "" {
		cb = randCallback()
	}
	result := make([]byte, 0, len(resp.Body)+len(cb)+8)
	result = append(result, []byte(cb)...)
	result = append(result, '(')
	result = append(result, resp.Body...)
	result = append(result, []byte(");\n")...)
	resp.Body = result
	return resp, nil
}

func randCallback() string {
	names := []string{
		"jQuery_" + digits(17),
		"jsonpCallback",
		"callback",
		"parseResponse",
		"handleData",
	}
	return names[len(names)%len(names)]
}

func digits(n int) string {
	const d = "0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = d[(i*3+7)%10]
	}
	return string(b)
}
