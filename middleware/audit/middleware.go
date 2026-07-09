// Package audit 给每个经过的请求记录一条 AuditEvent 到 Sink。
//  默认 Sink 是 stderr JSON handler；生产环境请替换成自己的 SIEM sink。
package audit

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/yliken/redbeanshellcore/core"
)

// AuditEvent 是单条审计事件的结构。
type AuditEvent struct {
	Time        time.Time `json:"ts"`           // 事件时间
	RequestID   string    `json:"request_id"`   // 请求 ID
	NodeID      string    `json:"node_id"`      // 节点 ID
	Operation   string    `json:"operation"`    // 操作名
	ArgsSummary string    `json:"args_summary"` // 参数摘要（截断后）
	Success     bool      `json:"success"`      // 是否成功
	ErrorKind   string    `json:"error_kind"`   // 错误分类
	Duration    int64     `json:"duration_ms"`  // 耗时毫秒
}

// Sink 是 AuditEvent 持久化的接口。
type Sink interface {
	Record(event AuditEvent) error
}

// defaultSink 把事件输出为一行 JSON 到 stderr。
type defaultSink struct {
	logger *slog.Logger
}

func (s *defaultSink) Record(event AuditEvent) error {
	s.logger.Info("audit", "event", event)
	return nil
}

// Options 调整审计中间件行为。
type Options struct {
	Sink       Sink
	ArgsMaxLen int           // 参数摘要最大长度
	Logger     *slog.Logger  // defaultSink 使用的 logger
}

// Middleware 返回一个记录 AuditEvent 的中间件。
func Middleware(opts ...Option) core.Middleware {
	cfg := Options{
		Sink:       &defaultSink{logger: slog.New(slog.NewJSONHandler(os.Stderr, nil))},
		ArgsMaxLen: 128,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			ev := AuditEvent{
				Time:        time.Now().UTC(),
				RequestID:   req.ID,
				NodeID:      req.NodeID,
				Operation:   req.Operation,
				Duration:    time.Since(start).Milliseconds(),
				Success:     err == nil,
				ArgsSummary: truncate(req.Meta["args_summary"], cfg.ArgsMaxLen),
			}
			if err != nil {
				oe := &core.OpError{}
				if errors.As(err, &oe) {
					ev.ErrorKind = string(oe.Kind)
				} else {
					ev.ErrorKind = string(core.ErrRemoteRuntime)
				}
			}
			_ = cfg.Sink.Record(ev)
			return resp, err
		}
	}
}

// Option 是调整 Options 的函数。
type Option func(*Options)

// WithSink 替换审计 sink。
func WithSink(s Sink) Option { return func(o *Options) { o.Sink = s } }

// WithArgsMaxLen 调整参数摘要最大长度。
func WithArgsMaxLen(n int) Option { return func(o *Options) { o.ArgsMaxLen = n } }

// WithLogger 替换 defaultSink 用 logger。
func WithLogger(l *slog.Logger) Option { return func(o *Options) { o.Logger = l } }

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "...[截断]"
}
