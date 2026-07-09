// Package timeout 给 Transport RoundTrip 套上每请求的 context 截止时间。
package timeout

import (
	"context"
	"errors"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// Options 调整超时行为。
type Options struct {
	Timeout      time.Duration        // 默认超时
	TimeoutShort time.Duration        // 快速操作的超时
	TimeoutLong  time.Duration        // 慢速操作（下载/上传/执行）的超时
	ShortTimeout func(op string) bool // 命中返回 true 表示用短超时
}

// Middleware 返回带截止时间的中间件。
func Middleware(opts Options) core.Middleware {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			deadline := opts.Timeout
			if opts.ShortTimeout != nil && opts.ShortTimeout(req.Operation) && opts.TimeoutShort > 0 {
				deadline = opts.TimeoutShort
			} else if opts.TimeoutLong > 0 && isLongOp(req.Operation) {
				deadline = opts.TimeoutLong
			}
			cancel := func() {}
			if _, ok := ctx.Deadline(); !ok {
				ctx, cancel = context.WithTimeout(ctx, deadline)
			}
			defer cancel()
			return next(ctx, req)
		}
	}
}

func isLongOp(op string) bool {
	switch op {
	case "file.download", "file.upload", "exec":
		return true
	}
	return false
}

var _ = errors.New
