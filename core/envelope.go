package core

import "context"

// Envelope 负责给请求加边界标记、从响应中截取出真实 payload。
//  Python demo 里的 tag_s / tag_e 协议就直接映射到这个接口。
type Envelope interface {
	Wrap(ctx context.Context, req *Request) (*Request, error)
	Extract(ctx context.Context, resp *Response) (*Response, error)
}
