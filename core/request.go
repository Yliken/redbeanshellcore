package core

import (
	"crypto/rand"
	"encoding/hex"
)

// Request 表示一次已经准备好发送到远端节点的请求。
type Request struct {
	ID        string
	NodeID    string            // 所属节点 ID
	Operation string            // 操作名（info / exec / file.read …）
	Payload   []byte            // 主请求体
	Params    map[string][]byte // 结构化参数
	Headers   map[string]string // HTTP / 传输层头
	Meta      map[string]string // codec / envelope / adapter 的元数据
}

// NewRequest 构建一个已初始化 map 的空 Request。
func NewRequest(op string) *Request {
	return &Request{
		Operation: op,
		Params:    make(map[string][]byte),
		Headers:   make(map[string]string),
		Meta:      make(map[string]string),
	}
}

// SetParam 写入一个字节类型的参数。
func (r *Request) SetParam(key string, value []byte) {
	if r.Params == nil {
		r.Params = make(map[string][]byte)
	}
	r.Params[key] = value
}

// SetParamString 写入一个字符串类型的参数（常用便捷方法）。
func (r *Request) SetParamString(key, value string) {
	r.SetParam(key, []byte(value))
}

// GetParam 读取一个字节类型的参数，第二返回值表示是否存在。
func (r *Request) GetParam(key string) ([]byte, bool) {
	if r.Params == nil {
		return nil, false
	}
	v, ok := r.Params[key]
	return v, ok
}

// SetHeader 设置一个 HTTP / 传输层头。
func (r *Request) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
}

// newRequestID 生成一个不可预测的十六进制请求 ID。
func newRequestID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// SetMeta 写入 codec / envelope / adapter 的元数据。
func (r *Request) SetMeta(key, value string) {
	if r.Meta == nil {
		r.Meta = make(map[string]string)
	}
	r.Meta[key] = value
}
