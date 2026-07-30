// Package audit 给每个经过的请求记录一条 AuditEvent 到 Sink。
// 默认 Sink 是 stderr JSON handler；生产环境请替换成自己的 SIEM sink。
package audit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// AuditEvent 是单条审计事件的结构。
type AuditEvent struct {
	Time        time.Time `json:"ts"`
	RequestID   string    `json:"request_id"`
	NodeID      string    `json:"node_id"`
	Operation   string    `json:"operation"`
	ArgsSummary string    `json:"args_summary"`
	Success     bool      `json:"success"`
	ErrorKind   string    `json:"error_kind"`
	Duration    int64     `json:"duration_ms"`
}

// Sink 是 AuditEvent 持久化的接口。
type Sink interface {
	Record(event AuditEvent) error
}

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
	ArgsMaxLen int
	Logger     *slog.Logger
	FailClosed bool
}

// Middleware 返回一个记录 AuditEvent 的中间件。
func Middleware(opts ...Option) core.Middleware {
	cfg := Options{
		ArgsMaxLen: 128,
		Logger:     slog.New(slog.NewJSONHandler(os.Stderr, nil)),
	}
	for _, option := range opts {
		if option != nil {
			option(&cfg)
		}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	if cfg.Sink == nil {
		cfg.Sink = &defaultSink{logger: cfg.Logger}
	}
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			start := time.Now()
			resp, err := next(ctx, req)
			event := AuditEvent{
				Time:        time.Now().UTC(),
				RequestID:   req.ID,
				NodeID:      req.NodeID,
				Operation:   req.Operation,
				Duration:    time.Since(start).Milliseconds(),
				Success:     err == nil,
				ArgsSummary: truncate(req.Meta["args_summary"], cfg.ArgsMaxLen),
			}
			if err != nil {
				var opErr *core.OpError
				if errors.As(err, &opErr) {
					event.ErrorKind = string(opErr.Kind)
				} else {
					event.ErrorKind = string(core.ErrRemoteRuntime)
				}
			}
			if recordErr := cfg.Sink.Record(event); recordErr != nil {
				if cfg.FailClosed {
					return resp, fmt.Errorf("audit: record failed (fail-closed): %w", recordErr)
				}
			}
			return resp, err
		}
	}
}

// Option 是调整 Options 的函数。
type Option func(*Options)

// WithSink 替换审计 sink。nil 表示回退到默认 sink。
func WithSink(s Sink) Option { return func(o *Options) { o.Sink = s } }

// WithArgsMaxLen 调整参数摘要最大长度。
func WithArgsMaxLen(n int) Option { return func(o *Options) { o.ArgsMaxLen = n } }

// WithLogger 替换 defaultSink 使用的 logger。
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		if logger != nil {
			o.Logger = logger
		}
	}
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n] + "...[截断]"
}
