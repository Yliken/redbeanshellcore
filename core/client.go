package core

import (
	"context"
	"fmt"
	"time"
)

// Client 是访问单个远端节点的顶层入口。
//  它执行完整的请求生命周期：
//
//	Operation.Build
//	  -> Codec.EncodeRequest
//	  -> Envelope.Wrap
//	  -> Transform.ApplyRequest
//	  -> Middleware Chain
//	  -> Transport.RoundTrip
//	  -> Transform.ApplyResponse
//	  -> Envelope.Extract
//	  -> Codec.DecodeResponse
//	  -> Operation.Parse
type Client struct {
	session     *Session
	transport   Transport
	codec       Codec
	envelope    Envelope
	transforms  []Transform
	middlewares []Middleware
}

// NewClient 用功能选项的方式构建一个 Client。
//  调用方通常至少需要 Transport、Codec、Envelope。
func NewClient(opts ...Option) *Client {
	c := &Client{
		transforms:  make([]Transform, 0),
		middlewares: make([]Middleware, 0),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Session 返回 Client 内的 session，供外部读取 / 补充元数据。
func (c *Client) Session() *Session { return c.session }

// Do 执行单个 Operation，返回结构化的 Result。
func (c *Client) Do(ctx context.Context, op Operation) (Result, error) {
	if c.transport == nil {
		return nil, NewOpError(ErrNetwork, op.Name(), c.nodeID(), "transport 未配置", nil)
	}

	// 把 node id 带到 context 中，方便中间件引用
	ctx = ContextWithNode(ctx, c.nodeID())

	// 1. 构建
	req, err := op.Build(ctx, c.session)
	if err != nil {
		return nil, wrapBuildError(op, c.nodeID(), err)
	}
	if req == nil {
		return nil, NewOpError(ErrProtocol, op.Name(), c.nodeID(), "operation.Build 返回了 nil", nil)
	}
	req.NodeID = c.nodeID()

	// 注入审计用元数据
	req.Meta["operation"] = op.Name()
	req.Meta["node_id"] = c.nodeID()
	req.Meta["ts"] = time.Now().UTC().Format(time.RFC3339)
	if c.codec != nil {
		req.Meta["codec"] = c.codec.Name()
	}

	// 把 session metadata 合并到 req.Meta，让 transport 能读到
	// auth_password_field / password_value 等字段。
	if c.session != nil {
		for k, v := range c.session.Metadata {
			if _, ok := req.Meta[k]; !ok {
				req.Meta[k] = v
			}
		}
	}

	// 2. 编码（可选）
	if c.codec != nil {
		req, err = c.codec.EncodeRequest(ctx, req)
		if err != nil {
			return nil, wrapError(ErrEncode, op.Name(), c.nodeID(), "codec 编码失败", err)
		}
	}

	// 3. 加边界（可选）
	if c.envelope != nil {
		req, err = c.envelope.Wrap(ctx, req)
		if err != nil {
			return nil, wrapError(ErrEnvelope, op.Name(), c.nodeID(), "envelope wrap 失败", err)
		}
	}

	// 4. 应用请求方向的 Transform
	for _, t := range c.transforms {
		if t.Direction() == TransformRequest || t.Direction() == TransformBoth {
			req, err = t.ApplyRequest(ctx, req)
			if err != nil {
				return nil, wrapError(ErrEncode, op.Name(), c.nodeID(), "transform 请求失败: "+t.Name(), err)
			}
		}
	}

	// 组装中间件链
	next := Handler(func(ctx context.Context, r *Request) (*Response, error) {
		resp, err := c.transport.RoundTrip(ctx, r)
		if err != nil {
			return nil, wrapError(ErrNetwork, op.Name(), c.nodeID(), "transport round-trip 失败", err)
		}
		resp.NodeID = c.nodeID()
		return resp, nil
	})
	if len(c.middlewares) > 0 {
		next = Chain(next, c.middlewares...)
	}

	// 5. 跑链
	resp, err := next(ctx, req)
	if err != nil {
		return nil, err
	}

	// 6. 应用响应方向的 Transform（倒序）
	for i := len(c.transforms) - 1; i >= 0; i-- {
		t := c.transforms[i]
		if t.Direction() == TransformResponse || t.Direction() == TransformBoth {
			resp, err = t.ApplyResponse(ctx, resp)
			if err != nil {
				return nil, wrapError(ErrDecode, op.Name(), c.nodeID(), "transform 响应失败: "+t.Name(), err)
			}
		}
	}

	// 7. 解开边界（可选）
	if c.envelope != nil {
		resp, err = c.envelope.Extract(ctx, resp)
		if err != nil {
			return nil, wrapError(ErrEnvelope, op.Name(), c.nodeID(), "envelope extract 失败", err)
		}
	}

	// 8. 解码（可选）
	if c.codec != nil {
		resp, err = c.codec.DecodeResponse(ctx, resp)
		if err != nil {
			return nil, wrapError(ErrDecode, op.Name(), c.nodeID(), "codec 解码失败", err)
		}
	}

	// 9. 解析
	res, err := op.Parse(ctx, resp)
	if err != nil {
		return nil, wrapError(ErrParse, op.Name(), c.nodeID(), "operation parse 失败", err)
	}
	return res, nil
}

func (c *Client) nodeID() string {
	if c.session == nil {
		return ""
	}
	return c.session.NodeID
}

func wrapBuildError(op Operation, nodeID string, err error) error {
	return wrapError(ErrProtocol, op.Name(), nodeID, "operation build 失败", err)
}

func wrapError(kind ErrorKind, opName, nodeID, msg string, cause error) error {
	if opErr, ok := cause.(*OpError); ok {
		// 保留已经包装过的内部 OpError
		return opErr
	}
	return NewOpError(kind, opName, nodeID, msg, cause)
}

// 下面这仨 var 是给未来字段扩容时提醒的占位；让 import 保留
// fmt 字段。如果哪天字段被删了需要把 _=fmt.Sprintf 删掉。
var _ = fmt.Sprintf
