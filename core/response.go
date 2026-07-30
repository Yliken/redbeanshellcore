package core

// Response 表示从远端节点拿到的原始响应。
type Response struct {
	RequestID  string
	NodeID     string
	StatusCode int
	Body       []byte
	Headers    map[string]string
	Meta       map[string]string
	EnvelopeOK bool        // 标记 Envelope 是否成功提取（供 Middleware 观测）
}

// NewResponse 构建一个已初始化 map 的空 Response。
func NewResponse() *Response {
	return &Response{
		Headers: make(map[string]string),
		Meta:    make(map[string]string),
	}
}

// SetMeta 写入响应元数据。
func (r *Response) SetMeta(key, value string) {
	if r.Meta == nil {
		r.Meta = make(map[string]string)
	}
	r.Meta[key] = value
}
