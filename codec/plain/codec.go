// Package plain 提供默认的 no-op Codec，请求 / 响应字节原样透传。
package plain

import (
	"context"

	"github.com/yliken/redbeanshellcore/core"
)

// Codec 是 core.Codec 的恒等变换实现，也是未配置 codec 时的默认值。
type Codec struct{}

// New 构建一个 plain codec。
func New() *Codec { return &Codec{} }

// Name 返回 codec 名称，用于 meta / 日志。
func (c *Codec) Name() string { return "plain" }

// EncodeRequest 对 plain codec 是 no-op。
func (c *Codec) EncodeRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	return req, nil
}

// DecodeResponse 对 plain codec 是 no-op。
func (c *Codec) DecodeResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	return resp, nil
}
