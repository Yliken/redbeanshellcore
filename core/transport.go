package core

import "context"

// Transport 负责把 Request 发出去、把 Response 收回来。
//  Transport 不需要理解 operation / codec / envelope 任何一层。
type Transport interface {
	RoundTrip(ctx context.Context, req *Request) (*Response, error)
}
