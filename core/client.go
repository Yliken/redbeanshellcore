package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"
)

// Client 是访问单个远端节点的顶层入口。
// 它执行完整的请求生命周期：
//
//	Operation.Build
//	  -> Codec.EncodeRequest
//	  -> Envelope.Wrap
//	  -> Transform.ApplyRequest
//	  -> Middleware Chain
//	     -> Transport.RoundTrip
//	     -> transform.ApplyResponse
//	     -> Envelope.Extract
//	     -> Codec.DecodeResponse
//	     -> Operation.Parse
//
// Middleware 包裹完整的响应处理流程，因此能观察 HTTP 状态、解码和解析错误。
type Client struct {
	session     *Session
	transport   Transport
	codec       Codec
	envelope    Envelope
	crypto      Crypto
	bodyCrypto  BodyCrypto
	configErr   error
	transforms  []Transform
	middlewares []Middleware
}

// NewClient 用功能选项的方式构建一个 Client。
// 调用方通常至少需要 Transport；Codec 和 Envelope 均为可选组件。
func NewClient(opts ...Option) *Client {
	c := &Client{
		transforms:  make([]Transform, 0),
		middlewares: make([]Middleware, 0),
	}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// Session 返回 Client 内的 session，供外部读取 / 补充元数据。
func (c *Client) Session() *Session { return c.session }

// Do 执行单个 Operation，返回结构化的 Result。
func (c *Client) Do(ctx context.Context, op Operation) (Result, error) {
	if c.configErr != nil {
		return nil, NewOpError(ErrProtocol, "", c.nodeID(), c.configErr.Error(), c.configErr)
	}
	if isNilInterface(op) {
		return nil, NewOpError(ErrProtocol, "", c.nodeID(), "operation 不能为空", nil)
	}
	opName := op.Name()
	if isNilInterface(c.transport) {
		return nil, NewOpError(ErrNetwork, opName, c.nodeID(), "transport 未配置", nil)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	adapterName := c.sessionAdapter()
	if adapterName != "" {
		cn := ""
		if !isNilInterface(c.codec) {
			cn = c.codec.Name()
		}
		en := ""
		if !isNilInterface(c.envelope) {
			if named, ok := c.envelope.(interface{ Name() string }); ok {
				en = named.Name()
			} else {
				en = "marker"
			}
		}
		if err := ValidateProfile(cn, en, adapterName); err != nil {
			return nil, err
		}
		mode := ""
		if !isNilInterface(c.crypto) {
			mode = c.crypto.Name()
		} else if !isNilInterface(c.bodyCrypto) {
			mode = c.bodyCrypto.Name()
		}
		if mode != "" {
			if profile, ok := KnownAdapterProfiles[adapterName]; ok && !profile.SupportsCrypto(mode) {
				return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "adapter "+adapterName+" does not support crypto mode "+mode, nil)
			}
			if declared := c.sessionMeta("crypto_mode"); declared != "" && declared != mode {
				return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "crypto mode mismatch: shell="+declared+" client="+mode, nil)
			}
		}
	}

	// 把 node id 带到 context 中，方便中间件引用。
	ctx = ContextWithNode(ctx, c.nodeID())

	// 1. 构建请求。
	req, err := op.Build(ctx, c.session)
	if err != nil {
		return nil, wrapBuildError(opName, c.nodeID(), err)
	}
	if req == nil {
		return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "operation.Build 返回了 nil", nil)
	}
	if req.Operation == "" {
		req.Operation = opName
	}
	req.NodeID = c.nodeID()

	// 注入审计用元数据。
	req.SetMeta("operation", opName)
	req.SetMeta("node_id", c.nodeID())
	req.SetMeta("ts", time.Now().UTC().Format(time.RFC3339))
	if !isNilInterface(c.codec) {
		req.SetMeta("codec", c.codec.Name())
	}

	// 生成不可预测的请求 ID，贯穿本次请求生命周期。
	if req.ID == "" {
		req.ID = newRequestID()
	}

	// 把操作风险等级注入 req.Meta，让 Readonly 中间件能按 RiskLevel 拦截。
	if aware, ok := op.(RiskAware); ok && !isNilInterface(aware) {
		req.SetMeta("risk_level", string(aware.RiskLevel()))
	}

	// session metadata 只补充缺失键，不覆盖 Operation.Build 已提供的值。
	if c.session != nil {
		for k, v := range c.session.Metadata {
			if _, ok := req.Meta[k]; !ok {
				req.SetMeta(k, v)
			}
		}
	}

	// 2. 编码（可选）。
	if !isNilInterface(c.codec) {
		req, err = c.codec.EncodeRequest(ctx, req)
		if err != nil {
			return nil, wrapError(ErrEncode, opName, c.nodeID(), "codec 编码失败", err)
		}
		if req == nil {
			return nil, NewOpError(ErrEncode, opName, c.nodeID(), "codec.EncodeRequest 返回了 nil", nil)
		}
	}

	// 3. 加边界（可选）。
	if !isNilInterface(c.envelope) {
		req, err = c.envelope.Wrap(ctx, req)
		if err != nil {
			return nil, wrapError(ErrEnvelope, opName, c.nodeID(), "envelope wrap 失败", err)
		}
		if req == nil {
			return nil, NewOpError(ErrEnvelope, opName, c.nodeID(), "envelope.Wrap 返回了 nil", nil)
		}
	}

	// 4. 应用请求方向的 Transform。
	for _, transform := range c.transforms {
		if isNilInterface(transform) {
			return nil, NewOpError(ErrEncode, opName, c.nodeID(), "transform 不能为空", nil)
		}
		direction := transform.Direction()
		if direction != TransformRequest && direction != TransformBoth {
			continue
		}
		req, err = transform.ApplyRequest(ctx, req)
		if err != nil {
			return nil, wrapError(ErrEncode, opName, c.nodeID(), "transform 请求失败: "+transform.Name(), err)
		}
		if req == nil {
			return nil, NewOpError(ErrEncode, opName, c.nodeID(), "transform.ApplyRequest 返回了 nil: "+transform.Name(), nil)
		}
	}

	for _, middleware := range c.middlewares {
		if isNilInterface(middleware) {
			return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "middleware 不能为空", nil)
		}
	}

	var result Result
	root := Handler(func(ctx context.Context, request *Request) (*Response, error) {
		result = nil
		// encrypt request before sending
		if !isNilInterface(c.crypto) && isNilInterface(c.bodyCrypto) {
			var encErr error
			request, encErr = c.crypto.Encrypt(ctx, request)
			if encErr != nil {
				return nil, wrapError(ErrCrypto, opName, c.nodeID(), "crypto encrypt failed", encErr)
			}
		}
		resp, roundTripErr := c.transport.RoundTrip(ctx, request)
		if resp != nil {
			prepareResponse(resp, request, c.nodeID())
		}
		// decrypt response after receiving
		if resp != nil && !isNilInterface(c.crypto) && isNilInterface(c.bodyCrypto) {
			var decErr error
			resp, decErr = c.crypto.Decrypt(ctx, resp)
			if decErr != nil {
				return resp, wrapError(ErrCrypto, opName, c.nodeID(), "crypto decrypt failed", decErr)
			}
		}
		if roundTripErr != nil {
			return resp, wrapError(ErrNetwork, opName, c.nodeID(), "transport round-trip 失败", roundTripErr)
		}
		if resp == nil {
			return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "transport.RoundTrip 返回了 nil", nil)
		}

		if resp.StatusCode != http.StatusOK {
			return resp, mapHTTPStatus(resp.StatusCode, opName, c.nodeID())
		}

		if !isNilInterface(c.envelope) {
			nextResp, extractErr := c.envelope.Extract(ctx, resp)
			if nextResp != nil {
				resp = nextResp
				prepareResponse(resp, request, c.nodeID())
			}
			if extractErr != nil {
				return resp, wrapError(ErrEnvelope, opName, c.nodeID(), "envelope extract 失败", extractErr)
			}
			if nextResp == nil {
				return resp, NewOpError(ErrEnvelope, opName, c.nodeID(), "envelope.Extract 返回了 nil", nil)
			}
		}

		// 先解包 Envelope，再应用响应 Transform（反向顺序）。
		for i := len(c.transforms) - 1; i >= 0; i-- {
			transform := c.transforms[i]
			direction := transform.Direction()
			if direction != TransformResponse && direction != TransformBoth {
				continue
			}
			nextResp, applyErr := transform.ApplyResponse(ctx, resp)
			if nextResp != nil {
				resp = nextResp
				prepareResponse(resp, request, c.nodeID())
			}
			if applyErr != nil {
				return resp, wrapError(ErrDecode, opName, c.nodeID(), "transform 响应失败: "+transform.Name(), applyErr)
			}
			if nextResp == nil {
				return resp, NewOpError(ErrDecode, opName, c.nodeID(), "transform.ApplyResponse 返回了 nil: "+transform.Name(), nil)
			}
		}

		if !isNilInterface(c.codec) {
			nextResp, decodeErr := c.codec.DecodeResponse(ctx, resp)
			if nextResp != nil {
				resp = nextResp
				prepareResponse(resp, request, c.nodeID())
			}
			if decodeErr != nil {
				return resp, wrapError(ErrDecode, opName, c.nodeID(), "codec 解码失败", decodeErr)
			}
			if nextResp == nil {
				return resp, NewOpError(ErrDecode, opName, c.nodeID(), "codec.DecodeResponse 返回了 nil", nil)
			}
		}

		parsed, parseErr := op.Parse(ctx, resp)
		if parseErr != nil {
			return resp, wrapError(ErrParse, opName, c.nodeID(), "operation parse 失败", parseErr)
		}
		if isNilInterface(parsed) {
			return resp, NewOpError(ErrParse, opName, c.nodeID(), "operation.Parse 返回了 nil", nil)
		}
		result = parsed
		return resp, nil
	})

	next := root
	if len(c.middlewares) > 0 {
		next = Chain(root, c.middlewares...)
	}
	_, err = next(ctx, req)
	if err != nil {
		return nil, wrapError(ErrNetwork, opName, c.nodeID(), "middleware chain 失败", err)
	}
	if isNilInterface(result) {
		return nil, NewOpError(ErrProtocol, opName, c.nodeID(), "middleware 未执行完整操作流程", nil)
	}
	return result, nil
}

func (c *Client) nodeID() string {
	if c.session == nil {
		return ""
	}
	return c.session.NodeID
}

func prepareResponse(resp *Response, req *Request, nodeID string) {
	if resp.RequestID == "" {
		resp.RequestID = req.ID
	}
	resp.NodeID = nodeID
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	if resp.Meta == nil {
		resp.Meta = make(map[string]string)
	}
	for key, value := range req.Meta {
		if !strings.HasPrefix(key, "marker.") {
			continue
		}
		if _, exists := resp.Meta[key]; !exists {
			resp.Meta[key] = value
		}
	}
}

func wrapBuildError(opName, nodeID string, err error) error {
	return wrapError(ErrProtocol, opName, nodeID, "operation build 失败", err)
}

func wrapError(kind ErrorKind, opName, nodeID, message string, cause error) error {
	if cause == nil {
		return NewOpError(kind, opName, nodeID, message, nil)
	}
	var existing *OpError
	if errors.As(cause, &existing) {
		copyErr := *existing
		if copyErr.Kind == "" {
			copyErr.Kind = kind
		}
		if copyErr.Operation == "" {
			copyErr.Operation = opName
		}
		if copyErr.NodeID == "" {
			copyErr.NodeID = nodeID
		}
		if copyErr.Message == "" {
			copyErr.Message = message
		}
		if copyErr.Kind == ErrNetwork && isTimeoutError(cause) {
			copyErr.Kind = ErrTimeout
		}
		return &copyErr
	}
	if isTimeoutError(cause) {
		kind = ErrTimeout
	}
	return NewOpError(kind, opName, nodeID, message, cause)
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// AdapterProfile describes supported codec/envelope combinations for an adapter.
type AdapterProfile struct {
	Name        string
	Codecs      []string
	Envelopes   []string
	Crypto      bool // 是否支持加密模式（兼容旧字段）
	CryptoModes []string
}

// SupportsCrypto reports whether the adapter profile supports the given crypto
// mode. CryptoModes takes precedence; when it is empty the legacy Crypto bool
// is used.
func (p AdapterProfile) SupportsCrypto(mode string) bool {
	if len(p.CryptoModes) > 0 {
		for _, m := range p.CryptoModes {
			if m == mode {
				return true
			}
		}
		return false
	}
	return p.Crypto
}

// SupportsEnvelope reports whether the adapter profile accepts the given
// envelope name. A nil Envelopes list or the "any" sentinel skips validation;
// an empty non-nil list rejects every envelope.
func (p AdapterProfile) SupportsEnvelope(name string) bool {
	if name == "" {
		return true
	}
	if p.Envelopes == nil {
		return true
	}
	for _, e := range p.Envelopes {
		if e == "any" || e == name {
			return true
		}
	}
	return false
}

// ValidateProfile checks if the codec/envelope combo is compatible with the adapter.
func ValidateProfile(codecName, envelopeName, adapter string) error {
	profile, ok := KnownAdapterProfiles[adapter]
	if !ok {
		return nil
	}
	if codecName != "" {
		if len(profile.Codecs) == 0 {
			return NewOpError(ErrProtocol, "", "", "adapter "+adapter+" does not support any codec, got "+codecName, nil)
		}
		if !containsString(profile.Codecs, codecName) {
			return NewOpError(ErrProtocol, "", "", "adapter "+adapter+" does not support codec "+codecName, nil)
		}
	}
	if envelopeName != "" {
		if profile.Envelopes == nil {
			return nil
		}
		if len(profile.Envelopes) == 0 {
			return NewOpError(ErrProtocol, "", "", "adapter "+adapter+" does not support any envelope, got "+envelopeName, nil)
		}
		if !profile.SupportsEnvelope(envelopeName) {
			return NewOpError(ErrProtocol, "", "", "adapter "+adapter+" does not support envelope "+envelopeName, nil)
		}
	}
	return nil
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// KnownAdapterProfiles maps adapter names to their supported component profiles.
var KnownAdapterProfiles = map[string]AdapterProfile{
	"php": {
		Name:      "php",
		Codecs:    []string{},
		Envelopes: []string{"marker"},
	},
	"php-eval": {
		Name:        "php-eval",
		Codecs:      []string{},
		Envelopes:   []string{},
		Crypto:      true,
		CryptoModes: []string{"aes-gcm"},
	},
	"asp": {
		Name:      "asp",
		Codecs:    []string{},
		Envelopes: []string{},
	},
	"aspx": {
		Name:      "aspx",
		Codecs:    []string{},
		Envelopes: []string{},
	},
	"jsp": {
		Name:        "jsp",
		Codecs:      []string{},
		Envelopes:   []string{},
		Crypto:      true,
		CryptoModes: []string{"aes-gcm"},
	},
}

func (c *Client) sessionMeta(key string) string {
	if c.session == nil {
		return ""
	}
	return c.session.Metadata[key]
}

func (c *Client) sessionAdapter() string {
	if c.session == nil {
		return ""
	}
	if c.session.Adapter != "" {
		return c.session.Adapter
	}
	return c.session.Metadata["adapter"]
}

// mapHTTPStatus 把 HTTP 状态码映射为对应的 SDK 错误分类。
func mapHTTPStatus(code int, op, node string) *OpError {
	switch {
	case code == 401:
		return NewOpError(ErrAuth, op, node, "远端返回 401 未授权", nil)
	case code == 403:
		return NewOpError(ErrPermission, op, node, "远端返回 403 拒绝访问", nil)
	case code == 404:
		return NewOpError(ErrNotFound, op, node, "远端返回 404 资源不存在", nil)
	case code >= 500 && code < 600:
		return NewOpError(ErrRemoteRuntime, op, node,
			fmt.Sprintf("远端返回 %d 运行时异常", code), nil)
	default:
		return NewOpError(ErrNetwork, op, node,
			fmt.Sprintf("远端返回未预期状态码 %d", code), nil)
	}
}
