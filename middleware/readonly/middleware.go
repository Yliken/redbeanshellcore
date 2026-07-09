// Package readonly 实现只读策略，拦截所有写入类操作。
//  命中规则的操作会直接返回 ErrPolicyDenied，不会进入 Transport。
package readonly

import (
	"context"

	"github.com/Yliken/redbeanshellcore/core"
)

// Middleware 返回拒绝写入类操作的中间件。
func Middleware() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			if isWrite(req) {
				return nil, &core.OpError{
					Kind:      core.ErrPolicyDenied,
					Operation: req.Operation,
					NodeID:    req.NodeID,
					Message:   "readonly: 写入/破坏性操作已被策略拦截",
				}
			}
			return next(ctx, req)
		}
	}
}

// isWrite 判断请求是否为写操作。先按 operation 名字匹配，再看 RiskAware 元数据。
func isWrite(req *core.Request) bool {
	switch req.Operation {
	case "exec", "file.write", "file.delete", "file.rename",
		"file.mkdir", "file.upload":
		return true
	}
	// RiskAware 接口可以覆盖静态名字列表。
	if level, ok := metaRiskAware(req); ok {
		switch level {
		case core.RiskWrite, core.RiskExec, core.RiskDestructive:
			return true
		}
	}
	return false
}

// metaRiskAware 解析 req.Meta["risk_level"]，作为 RiskAware 的轻量替代。
func metaRiskAware(req *core.Request) (core.RiskLevel, bool) {
	v := req.Meta["risk_level"]
	if v == "" {
		return "", false
	}
	return core.RiskLevel(v), true
}
