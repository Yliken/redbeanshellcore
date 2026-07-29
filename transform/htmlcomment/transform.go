// Package htmlcomment 提供将响应正文包裹在 HTML 注释中的 Transform。
// 适用于需要将 C2 流量伪装成普通 HTML 页面的场景。
package htmlcomment

import (
	"context"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Transform 将响应正文包裹在 <!-- --> HTML 注释中。
type Transform struct {
	prefix     string // 注释前缀（如 " <!-- "）
	suffix     string // 注释后缀（如 " --> "）
	wrapRound  bool   // 是否在请求方向也包裹
}

// New 构建一个 HTML 注释 Transform。
// wrapRequest 为 true 时也会在请求方向包裹。
func New(wrapRequest bool) *Transform {
	return &Transform{
		prefix:    "<!-- ",
		suffix:    " -->",
		wrapRound: wrapRequest,
	}
}

// NewCustom 构建一个自定义前缀/后缀的 Transform。
func NewCustom(prefix, suffix string, wrapRequest bool) *Transform {
	return &Transform{
		prefix:    prefix,
		suffix:    suffix,
		wrapRound: wrapRequest,
	}
}

func (t *Transform) Name() string { return "htmlcomment" }

func (t *Transform) Direction() core.TransformDirection {
	if t.wrapRound {
		return core.TransformBoth
	}
	return core.TransformResponse
}

func (t *Transform) ApplyRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("htmlcomment: request 为空")
	}
	if !t.wrapRound {
		return req, nil
	}
	req.Payload = append([]byte(t.prefix), append(req.Payload, []byte(t.suffix)...)...)
	return req, nil
}

func (t *Transform) ApplyResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("htmlcomment: response 为空")
	}
	resp.Body = append([]byte(t.prefix), append(resp.Body, []byte(t.suffix)...)...)
	return resp, nil
}
