package core

import "context"

// Codec 负责把 Request 编码成 wire 格式、把 Response 从 wire 格式解码回来。
//  Codec 不发数据、不解析 operation、不做策略判断。
type Codec interface {
	Name() string
	EncodeRequest(ctx context.Context, req *Request) (*Request, error)
	DecodeResponse(ctx context.Context, resp *Response) (*Response, error)
}
