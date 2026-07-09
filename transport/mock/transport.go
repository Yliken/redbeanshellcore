// Package mock 提供基于内存的 Transport，把请求路由到用户提供的 handler。
//  专为测试、示例和不需要真实网络的适配器原型设计。
package mock

import (
	"context"
	"errors"

	"github.com/Yliken/redbeanshellcore/core"
)

// Transport 是内存 Transport。每次 RoundTrip 都会派发到构造时注入的 Handler。
type Transport struct {
	Handler func(ctx context.Context, req *core.Request) (*core.Response, error)
}

// New 构建一个 Transport。如果 h 为 nil，每次 RoundTrip 返回 ErrNoHandler。
func New(h func(ctx context.Context, req *core.Request) (*core.Response, error)) *Transport {
	return &Transport{Handler: h}
}

// RoundTrip 调用配置好的 Handler。
func (t *Transport) RoundTrip(ctx context.Context, req *core.Request) (*core.Response, error) {
	if t.Handler == nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "mock transport: handler 未配置"}
	}
	return t.Handler(ctx, req)
}

// FailAlways 是一个返回给定错误的 handler 工厂。
func FailAlways(err error) func(context.Context, *core.Request) (*core.Response, error) {
	return func(_ context.Context, _ *core.Request) (*core.Response, error) {
		if err == nil {
			return nil, errors.New("mock transport: 注入的失败")
		}
		return nil, err
	}
}

// EchoHandler 把请求原封不动塞回 response payload，常用于测试管线。
func EchoHandler(_ context.Context, req *core.Request) (*core.Response, error) {
	resp := core.NewResponse()
	resp.Body = append([]byte{}, req.Payload...)
	return resp, nil
}

// StaticHandler 忽略请求，永远返回同一个 body。
func StaticHandler(body []byte) func(context.Context, *core.Request) (*core.Response, error) {
	return func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.Body = append([]byte{}, body...)
		return resp, nil
	}
}

// DispatchHandler 按 operation 名称分发到不同的 body，找不到时返回 ErrNotFound。
func DispatchHandler(table map[string][]byte) func(context.Context, *core.Request) (*core.Response, error) {
	return func(_ context.Context, req *core.Request) (*core.Response, error) {
		body, ok := table[req.Operation]
		if !ok {
			return nil, &core.OpError{Kind: core.ErrNotFound, Operation: req.Operation, Message: "mock dispatcher: 没有对应条目"}
		}
		resp := core.NewResponse()
		resp.Body = append([]byte{}, body...)
		return resp, nil
	}
}
