// Package logging 为每个请求输出一行结构化日志。出于安全，
//  默认不会记录 password / token / 文件全文 / 未脱敏命令参数。
package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/yliken/redbeanshellcore/core"
)

// Options 调整日志行为。
type Options struct {
	Logger     *slog.Logger // 自定义 slog.Logger；nil 时默认用 stderr 文本 handler
	LogPayload bool         // 是否记录截断后的 payload（默认不记录）
	PayloadMax int          // payload 在日志中最多显示的字节数
}

// Middleware 返回一个在每次 Handler 调用时打日志的 core.Middleware。
func Middleware(opts ...Option) core.Middleware {
	cfg := Options{
		Logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
		LogPayload: false,
		PayloadMax: 64,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			start := time.Now()
			op := req.Operation
			node := req.NodeID
			if node == "" {
				node = core.NodeFromContext(ctx)
			}
			payload := ""
			if cfg.LogPayload && len(req.Payload) > 0 {
				payload = truncate(string(req.Payload), cfg.PayloadMax)
			}
			resp, err := next(ctx, req)
			dur := time.Since(start)
			if err != nil {
				_ = &core.OpError{} // 保留 errors.As 的能力
				cfg.Logger.Error("remote_node_request",
					"operation", op,
					"node", node,
					"duration_ms", dur.Milliseconds(),
					"error", err.Error(),
					"payload", payload,
				)
				return resp, err
			}
			status := 0
			if resp != nil {
				status = resp.StatusCode
			}
			cfg.Logger.Info("remote_node_request",
				"operation", op,
				"node", node,
				"status", status,
				"duration_ms", dur.Milliseconds(),
				"payload", payload,
			)
			return resp, nil
		}
	}
}

// Option 是调整 Options 的函数。
type Option func(*Options)

// WithLogger 覆盖默认 logger。
func WithLogger(l *slog.Logger) Option {
	return func(o *Options) { if l != nil { o.Logger = l } }
}

// WithPayloadLogging 开启截断后的 payload 日志。
func WithPayloadLogging(max int) Option {
	return func(o *Options) { o.LogPayload = true; o.PayloadMax = max }
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[截断]"
}

// 让 fmt 保留可达。
var _ = fmt.Sprintf
