// Package httpform 实现把 POST 请求以 application/x-www-form-urlencoded
// 形式发出去的 Transport，行为上对应 AntSword 的 PHP 客户端。
//
// req.Params 里的值会原样成为表单字段；需要 base64 等协议编码时，
// 由对应 Operation 在构建 Request 时完成。Metadata 里的 payload_form_field
// 决定哪个字段携带主 payload。
//
// ⚠️ 仅用于授权的实验室环境。InsecureTLS 默认关闭，调用方需自行显式开启。
package httpform

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/protocol/wire"
	"github.com/Yliken/redbeanshellcore/transport/useragent"
)

const (
	maxResponseBodySize int64 = 64 * 1024 * 1024
)

// HTTPProtocol 表示可选的 HTTP 协议版本。
type HTTPProtocol int

const (
	ProtocolAuto   HTTPProtocol = iota // 自动协商
	ProtocolHTTP11                     // HTTP/1.1
	ProtocolHTTP2                      // HTTP/2
	ProtocolHTTP3                      // HTTP/3
)

// ProxyType 表示代理类型。
type ProxyType int

const (
	ProxyHTTP   ProxyType = iota // HTTP 代理
	ProxySOCKS5                  // SOCKS5 代理
)

// ProxyConfig 描述一个代理节点。
type ProxyConfig struct {
	Type     ProxyType
	Host     string
	Port     int
	Username string
	Password string
}

// TLSFingerprint 控制 TLS 客户端指纹的随机化配置。
type TLSFingerprint struct {
	Enabled          bool
	MinTLSVersion    uint16
	CipherSuites     []uint16
	CurvePreferences []tls.CurveID
}

// Options 包含传输层的全部可配置选项。
type Options struct {
	Timeout            time.Duration
	InsecureTLS        bool
	DisableCompression bool

	// P0.1 — User-Agent 轮换
	UAPool     *useragent.Pool
	UARotation bool

	// P0.3 — 动态表单字段名
	DynamicFieldNames bool
	FieldGen          *FieldGenerator

	// P0.5 — 请求正文填充（Honeypot Fields）
	EnablePadding bool
	HoneypotCount int
	honeypot      *HoneypotFields

	// P0.6 — TLS 指纹随机化
	TLSFingerprint TLSFingerprint

	// P0.8 — 多协议协商
	Protocol HTTPProtocol

	// P0.9 — 连接池与会话维持
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	EnableCookieJar     bool

	// P0.10 — 代理链与出口 IP 轮换
	ProxyChain    []ProxyConfig
	ProxyRotation bool
	// P1.1 — Wire Protocol
	WireProtocol bool

	// P1 — Body 级加密：发送前序列化整个表单并加密，响应先解密再走原链路。
	BodyCrypto  core.BodyCrypto
	CryptoField string
	BodyCodec   wire.BodyCodec
}

// DefaultOptions 返回默认的传输层选项。
func DefaultOptions() Options {
	return Options{
		Timeout:             30 * time.Second,
		InsecureTLS:         false,
		DisableCompression:  false,
		UARotation:          false,
		DynamicFieldNames:   false,
		EnablePadding:       false,
		HoneypotCount:       3,
		Protocol:            ProtocolAuto,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     90 * time.Second,
		EnableCookieJar:     true,
		ProxyRotation:       false,
		WireProtocol:        false,
		CryptoField:         "__crypto",
		BodyCodec:           wire.NewCompactFormCodec(),
		TLSFingerprint: TLSFingerprint{
			Enabled:       false,
			MinTLSVersion: tls.VersionTLS12,
		},
	}
}

// Transport 是基于表单的 HTTP 传输实现，支持全部 P0 流量伪装特性。
type Transport struct {
	Endpoint     string
	ExtraHeaders map[string]string
	Options      Options

	// 长期复用的 http.Client（P0.9 — 连接复用）
	client     *http.Client
	clientOnce sync.Once
	cookieJar  http.CookieJar
	proxyNext  int
	proxyMu    sync.Mutex
}

// New 构建一个默认超时为 15 秒的 httpform Transport。
// 如需使用 P0 流量伪装特性，请在返回后设置 Transport.Options。
func New(endpoint string) *Transport {
	return &Transport{
		Endpoint:     endpoint,
		ExtraHeaders: make(map[string]string),
		Options:      DefaultOptions(),
	}
}

// NewWithOptions 用完整选项构建 Transport。
func NewWithOptions(endpoint string, opts Options) *Transport {
	t := &Transport{
		Endpoint:     endpoint,
		ExtraHeaders: make(map[string]string),
		Options:      opts,
	}
	return t
}

// RoundTrip 把请求编码为表单 POST，应用 P0 流量伪装特性。
func (t *Transport) RoundTrip(ctx context.Context, req *core.Request) (*core.Response, error) {
	if req == nil {
		return nil, &core.OpError{Kind: core.ErrProtocol, Message: "httpform: request 不能为空"}
	}
	if t.Endpoint == "" {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: endpoint 为空"}
	}
	if t.Options.BodyCrypto != nil && t.Options.WireProtocol {
		return nil, &core.OpError{Kind: core.ErrProtocol, Operation: req.Operation, Message: "httpform: BodyCrypto 与 WireProtocol 互斥，不能同时启用"}
	}

	// 构建表单
	form, err := t.buildForm(ctx, req)
	if err != nil {
		return nil, err
	}

	httpClient := t.getClient()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: 构造请求失败", Cause: err}
	}

	// 设置请求头
	t.setHeaders(httpReq, req)

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: POST 失败", Cause: err}
	}
	defer httpResp.Body.Close()

	data, oversized, readErr := readBodyLimited(httpResp.Body, maxResponseBodySize)
	out := core.NewResponse()
	out.RequestID = req.ID
	out.StatusCode = httpResp.StatusCode
	out.Headers = flattenHeaders(httpResp.Header)
	if t.Options.BodyCrypto != nil && len(data) > 0 {
		decrypted, decErr := t.Options.BodyCrypto.DecryptBody(ctx, data)
		if decErr != nil {
			out.Body = data
			return out, &core.OpError{Kind: core.ErrCrypto, Operation: req.Operation, Message: "httpform: 响应解密失败", Cause: decErr}
		}
		data = decrypted
	}
	out.Body = data
	if readErr != nil {
		return out, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: 读取 body 失败", Cause: readErr}
	}
	if oversized {
		return out, &core.OpError{Kind: core.ErrProtocol, Operation: req.Operation, Message: "httpform: 响应 body 超过 64MiB 上限"}
	}
	return out, nil
}

// getClient 返回长期复用的 http.Client（P0.9）。
func (t *Transport) getClient() *http.Client {
	t.clientOnce.Do(func() {
		tr := t.buildTransport()
		var jar http.CookieJar
		if t.Options.EnableCookieJar {
			jar = newCookieJar()
		}
		t.client = &http.Client{
			Timeout:   t.effectiveTimeout(),
			Transport: tr,
			Jar:       jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	})
	return t.client
}

func (t *Transport) effectiveTimeout() time.Duration {
	if t.Options.Timeout > 0 {
		return t.Options.Timeout
	}
	return 30 * time.Second
}

// buildTransport 构建底层 http.Transport（集成 P0.6、P0.8、P0.9、P0.10）。
func (t *Transport) buildTransport() *http.Transport {
	tr := &http.Transport{
		MaxIdleConns:        t.Options.MaxIdleConns,
		MaxIdleConnsPerHost: t.Options.MaxIdleConnsPerHost,
		IdleConnTimeout:     t.Options.IdleConnTimeout,
		DisableCompression:  t.Options.DisableCompression,
		ForceAttemptHTTP2:   true,
	}

	// P0.6 — TLS 指纹随机化
	if t.Options.TLSFingerprint.Enabled {
		tr.TLSClientConfig = t.buildTLSConfig()
	} else if t.Options.InsecureTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12}
	}

	// P0.8 — 多协议协商
	t.configureProtocol(tr)

	// P0.10 — 代理链
	if len(t.Options.ProxyChain) > 0 {
		tr.Proxy = t.proxyFunc()
	}

	return tr
}

func (t *Transport) buildTLSConfig() *tls.Config {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: t.Options.InsecureTLS,
		MinVersion:         t.Options.TLSFingerprint.MinTLSVersion,
	}

	if len(t.Options.TLSFingerprint.CipherSuites) > 0 {
		tlsCfg.CipherSuites = t.Options.TLSFingerprint.CipherSuites
	} else {
		// 每次连接从常见密码套件池中随机选一份子集，模拟不同浏览器的偏好
		tlsCfg.CipherSuites = pickRandomCipherSuites()
	}

	if len(t.Options.TLSFingerprint.CurvePreferences) > 0 {
		tlsCfg.CurvePreferences = t.Options.TLSFingerprint.CurvePreferences
	} else {
		tlsCfg.CurvePreferences = []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		}
	}
	return tlsCfg
}

// configureProtocol 根据配置设置多协议行为。
func (t *Transport) configureProtocol(tr *http.Transport) {
	switch t.Options.Protocol {
	case ProtocolHTTP11:
		tr.ForceAttemptHTTP2 = false
		tr.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	case ProtocolHTTP2:
		tr.ForceAttemptHTTP2 = true
	case ProtocolHTTP3:
		// HTTP/3 (QUIC) 需要 quic-go 或对应库，在此标记为待支持
		tr.ForceAttemptHTTP2 = false
	default:
		// ProtocolAuto — 默认行为，允许 HTTP/2 协商
		tr.ForceAttemptHTTP2 = true
	}
}

// proxyFunc 返回一个支持轮换的代理选择函数（P0.10）。
func (t *Transport) proxyFunc() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		proxy := t.nextProxy()
		if proxy == nil {
			return http.ProxyFromEnvironment(req)
		}

		var scheme string
		switch proxy.Type {
		case ProxySOCKS5:
			scheme = "socks5"
		default:
			scheme = "http"
		}

		proxyURL := &url.URL{
			Scheme: scheme,
			Host:   fmt.Sprintf("%s:%d", proxy.Host, proxy.Port),
		}

		if proxy.Username != "" {
			proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
		}

		return proxyURL, nil
	}
}

func (t *Transport) nextProxy() *ProxyConfig {
	if len(t.Options.ProxyChain) == 0 {
		return nil
	}

	if !t.proxyRotation() {
		return &t.Options.ProxyChain[0]
	}

	t.proxyMu.Lock()
	defer t.proxyMu.Unlock()

	idx := t.proxyNext
	t.proxyNext = (t.proxyNext + 1) % len(t.Options.ProxyChain)
	return &t.Options.ProxyChain[idx]
}

func (t *Transport) proxyRotation() bool {
	return t.Options.ProxyRotation
}

// ResetClient 重置 http.Client（在配置变更后调用）。
func (t *Transport) ResetClient() {
	t.clientOnce = sync.Once{}
	t.client = nil
}

// buildForm 构造表单，集成 P0.3（动态字段名）、P0.5（诱饵字段）和
// P1 BodyCrypto（整个表单序列化后加密为单一 CryptoField）。
func (t *Transport) buildForm(ctx context.Context, req *core.Request) (url.Values, error) {
	form := url.Values{}

	// P0.3 — 动态字段名
	payloadField := t.resolvePayloadField(req)

	// 主 payload 字段
	form.Set(payloadField, string(req.Payload))

	// req.Params 字段（如果用动态字段名模式，用随机字段名）
	for key, value := range req.Params {
		fieldName := key
		if t.Options.DynamicFieldNames {
			if t.Options.FieldGen != nil {
				fieldName = t.Options.FieldGen.Generate()
			} else {
				g := NewFieldGenerator()
				fieldName = g.Generate()
			}
		}
		form.Set(fieldName, string(value))
	}

	// env template vars: key|||askey|||value pairs joined by |||asline|||
	if env := req.Meta["env_template_vars"]; env != "" {
		for _, pair := range strings.Split(env, "|||asline|||") {
			if key, value, ok := strings.Cut(pair, "|||askey|||"); ok {
				form.Set(key, value)
			}
		}
	}

	// P0.5 — 诱饵字段（Honeypot）
	var honeypotFields map[string]string
	if t.Options.EnablePadding && t.Options.HoneypotCount > 0 {
		hf := t.getHoneypot()
		honeypotFields = hf.Generate()
		for name, value := range honeypotFields {
			if form.Get(name) == "" {
				form.Set(name, value)
			}
		}
	}

	// P1.1 — Wire Protocol 字段
	if t.Options.WireProtocol {
		key := req.Meta["hmac_key"]
		env := wire.NewRequestEnvelope(req.ID, string(req.Payload), key)
		for k, v := range env.FormFields() {
			form.Set(k, v)
		}
	}

	if t.Options.BodyCrypto != nil {
		values := make(map[string][]byte, len(form))
		for name, entries := range form {
			if len(entries) > 0 {
				values[name] = []byte(entries[0])
			}
		}
		encoded, err := t.bodyCodec().Encode(values)
		if err != nil {
			return nil, &core.OpError{Kind: core.ErrProtocol, Operation: req.Operation, Message: "httpform: 表单序列化失败", Cause: err}
		}
		encrypted, err := t.Options.BodyCrypto.EncryptBody(ctx, encoded)
		if err != nil {
			return nil, &core.OpError{Kind: core.ErrCrypto, Operation: req.Operation, Message: "httpform: 请求体加密失败", Cause: err}
		}

		out := url.Values{}
		out.Set(t.cryptoField(), string(encrypted))
		for name, value := range honeypotFields {
			if out.Get(name) == "" {
				out.Set(name, value)
			}
		}
		return out, nil
	}

	return form, nil
}

func (t *Transport) cryptoField() string {
	if t.Options.CryptoField != "" {
		return t.Options.CryptoField
	}
	return "__crypto"
}

func (t *Transport) bodyCodec() wire.BodyCodec {
	if t.Options.BodyCodec != nil {
		return t.Options.BodyCodec
	}
	return wire.NewCompactFormCodec()
}

// resolvePayloadField 解析主 payload 字段名（P0.3）。
// 优先使用 payload_form_field，向后兼容旧的 auth_password_field key 名称。
func (t *Transport) resolvePayloadField(req *core.Request) string {
	// 优先使用 session metadata 指定的字段名
	if field := req.Meta["payload_form_field"]; field != "" {
		return field
	}
	// 向后兼容旧的 key 名称（auth_password_field）
	if field := req.Meta["auth_password_field"]; field != "" {
		return field
	}

	// P0.3 — 动态字段名模式
	if t.Options.DynamicFieldNames {
		if t.Options.FieldGen != nil {
			return t.Options.FieldGen.Generate()
		}
		g := NewFieldGenerator()
		return g.Generate()
	}

	return "antpwd"
}

func (t *Transport) getHoneypot() *HoneypotFields {
	if t.Options.honeypot == nil {
		t.Options.honeypot = NewHoneypotFields(t.Options.HoneypotCount)
	}
	return t.Options.honeypot
}

// setHeaders 设置 HTTP 请求头，集成 P0.1（UA 轮换）。
func (t *Transport) setHeaders(httpReq *http.Request, req *core.Request) {
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// P0.1 — User-Agent 轮换
	if t.Options.UARotation && t.Options.UAPool != nil {
		profile := t.Options.UAPool.Pick()
		httpReq.Header.Set("User-Agent", profile.UserAgent)

		if profile.Accept != "" {
			httpReq.Header.Set("Accept", profile.Accept)
		}
		if profile.AcceptLanguage != "" {
			httpReq.Header.Set("Accept-Language", profile.AcceptLanguage)
		}
		if profile.AcceptEncoding != "" {
			httpReq.Header.Set("Accept-Encoding", profile.AcceptEncoding)
		}
		if profile.SecCHUA != "" {
			httpReq.Header.Set("Sec-CH-UA", profile.SecCHUA)
		}
		if profile.SecCHUAMobile != "" {
			httpReq.Header.Set("Sec-CH-UA-Mobile", profile.SecCHUAMobile)
		}
		if profile.SecCHUAPlatform != "" {
			httpReq.Header.Set("Sec-CH-UA-Platform", profile.SecCHUAPlatform)
		}
	} else {
		httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	}

	// 从请求头部覆盖
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// ExtraHeaders 最后设置（最低优先级）
	for key, value := range t.ExtraHeaders {
		httpReq.Header.Set(key, value)
	}
}

func readBodyLimited(reader io.Reader, limit int64) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(data)) > limit {
		return data[:limit], true, err
	}
	return data, false, err
}

func flattenHeaders(headers http.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		out[key] = values[0]
	}
	return out
}

// cookieJar 是一个简单的内存 Cookie Jar 实现。
type cookieJar struct {
	mu  sync.Mutex
	jar map[string][]*http.Cookie
}

func newCookieJar() *cookieJar {
	return &cookieJar{jar: make(map[string][]*http.Cookie)}
}

func (j *cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if u != nil {
		j.jar[u.Host] = cookies
	}
}

func (j *cookieJar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	if u != nil {
		return j.jar[u.Host]
	}
	return nil
}

// pickRandomCipherSuites 从常见密码套件中随机选子集。
func pickRandomCipherSuites() []uint16 {
	allSuites := []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	}

	// 随机选取 4~8 个套件
	n, err := rand.Int(rand.Reader, big.NewInt(5))
	if err != nil {
		return allSuites[:8]
	}
	count := int(n.Int64()) + 4
	if count > len(allSuites) {
		count = len(allSuites)
	}

	selected := make([]uint16, 0, count)
	indices := shuffleIndices(len(allSuites))
	for i := 0; i < count && i < len(indices); i++ {
		selected = append(selected, allSuites[indices[i]])
	}
	return selected
}

// shuffleIndices 返回 [0, n) 的一个随机排列。
func shuffleIndices(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			continue
		}
		indices[i], indices[j.Int64()] = indices[j.Int64()], indices[i]
	}
	return indices
}
