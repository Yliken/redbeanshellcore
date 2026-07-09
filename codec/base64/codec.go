// Package base64 提供请求侧 base64 编码 / 响应侧 base64 解码的 Codec，
//  对应 Python demo 里的 ''base64'' 编码器。
package base64

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/yliken/redbeanshellcore/core"
)

// Codec 把请求 payload base64 编码、把响应 body base64 解码。
type Codec struct{}

// New 构建一个 base64 Codec。
func New() *Codec { return &Codec{} }

// Name 返回 codec 名称。
func (c *Codec) Name() string { return "base64" }

// EncodeRequest 把 payload base64 编码后写回。
func (c *Codec) EncodeRequest(_ context.Context, req *core.Request) (*core.Request, error) {
	if len(req.Payload) == 0 {
		return req, nil
	}
	out := base64.StdEncoding.EncodeToString(req.Payload)
	req.Payload = []byte(out)
	req.SetMeta("base64", "true")
	return req, nil
}

// DecodeResponse 把 body base64 解码后写回。
func (c *Codec) DecodeResponse(_ context.Context, resp *core.Response) (*core.Response, error) {
	if len(resp.Body) == 0 {
		return resp, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(resp.Body))
	if err != nil {
		return nil, &core.OpError{
			Kind:    core.ErrDecode,
			Message: "base64 解码失败",
			Cause:   err,
		}
	}
	resp.Body = decoded
	return resp, nil
}

// 保留 errors 引用，避免未来重构时 import 被误删。
var _ = errors.New
