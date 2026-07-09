package core

import "context"

// Handler 是中间件链的最后一环——真正完成 round-trip 的地方。
type Handler func(ctx context.Context, req *Request) (*Response, error)

// Middleware 用横切行为（日志、审计、只读、超时、重试…）包裹一个 Handler。
type Middleware func(next Handler) Handler

// Chain 把一组 Middleware 按顺序包到 root Handler 外面。
//  顺序为 mws[0] 在最外层、mws[len-1] 最靠近 root。
func Chain(root Handler, mws ...Middleware) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		root = mws[i](root)
	}
	return root
}
