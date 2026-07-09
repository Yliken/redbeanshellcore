package core

import "context"

// TransformDirection 表示 Transform 作用在请求、响应还是双向。
type TransformDirection string

const (
	TransformRequest  TransformDirection = "request"  // 仅请求
	TransformResponse TransformDirection = "response" // 仅响应
	TransformBoth     TransformDirection = "both"     // 双向
)

// Transform 是给安全工具开发者和研究员预留的流量变形扩展点。
//  core 只定义接口与生命周期，不内置任何规避实现。
type Transform interface {
	Name() string
	Direction() TransformDirection
	ApplyRequest(ctx context.Context, req *Request) (*Request, error)
	ApplyResponse(ctx context.Context, resp *Response) (*Response, error)
}
