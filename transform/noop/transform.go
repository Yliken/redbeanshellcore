// Package noop 提供 no-op 的 Transform 实现，请求 / 响应原样透传。
//  它是未配置流量变形时的默认值，也是自定义 Transform 的参考模板。
package noop

import (
	"context"

	"github.com/Yliken/redbeanshellcore/core"
)

// Transform 是 no-op 的 Transform 实现。
type Transform struct{}

// New 构建一个 no-op Transform。
func New() *Transform { return &Transform{} }

// Name 返回 "noop"。
func (t *Transform) Name() string { return "noop" }

// Direction 返回 "both"（双向都不做变换）。
func (t *Transform) Direction() core.TransformDirection { return core.TransformBoth }

// ApplyRequest 原样返回请求。
func (t *Transform) ApplyRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	return req, nil
}

// ApplyResponse 原样返回响应。
func (t *Transform) ApplyResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	return resp, nil
}
