// Package httpform 实现把 POST 请求以 application/x-www-form-urlencoded
// 形式发出去的 Transport，行为上对应 AntSword 的 PHP 客户端。
//
// req.Params 里的值会原样成为表单字段；需要 base64 等协议编码时，
// 由对应 Operation 在构建 Request 时完成。Metadata 里的 auth_password_field
// 决定哪个字段携带主 payload。
//
// ⚠️ 仅用于授权的实验室环境。InsecureTLS 默认关闭，调用方需自行显式开启。
package httpform

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

const maxResponseBodySize int64 = 64 * 1024 * 1024

// Transport 是基于表单的 HTTP 传输实现。
type Transport struct {
	Endpoint     string
	Timeout      time.Duration
	InsecureTLS  bool
	ExtraHeaders map[string]string
	Client       *http.Client
}

// New 构建一个默认超时为 15 秒的 httpform Transport。
func New(endpoint string) *Transport {
	return &Transport{
		Endpoint:     endpoint,
		Timeout:      15 * time.Second,
		ExtraHeaders: make(map[string]string),
	}
}

// RoundTrip 把请求编码为表单 POST。req.Payload 作为主密码字段 body，
// req.Params 作为额外字段。超过 64MiB 的响应会返回 ErrProtocol。
func (t *Transport) RoundTrip(ctx context.Context, req *core.Request) (*core.Response, error) {
	if req == nil {
		return nil, &core.OpError{Kind: core.ErrProtocol, Message: "httpform: request 不能为空"}
	}
	if t.Endpoint == "" {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: endpoint 为空"}
	}

	form := url.Values{}
	if field := req.Meta["auth_password_field"]; field != "" {
		form.Set(field, string(req.Payload))
	} else {
		form.Set("antpwd", string(req.Payload))
	}
	for key, value := range req.Params {
		form.Set(key, string(value))
	}
	if env := req.Meta["env_template_vars"]; env != "" {
		for _, pair := range strings.Split(env, "|||asline|||") {
			if key, value, ok := strings.Cut(pair, "|||askey|||"); ok {
				form.Set(key, value)
			}
		}
	}

	httpClient := t.Client
	if httpClient == nil {
		httpClient = t.buildClient()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: 构造请求失败", Cause: err}
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	for key, value := range t.ExtraHeaders {
		httpReq.Header.Set(key, value)
	}

	httpResp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: POST 失败", Cause: err}
	}
	defer httpResp.Body.Close()

	data, oversized, readErr := readBodyLimited(httpResp.Body, maxResponseBodySize)
	out := core.NewResponse()
	out.RequestID = req.ID
	out.StatusCode = httpResp.StatusCode
	out.Body = data
	out.Headers = flattenHeaders(httpResp.Header)
	if readErr != nil {
		return out, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: 读取 body 失败", Cause: readErr}
	}
	if oversized {
		return out, &core.OpError{Kind: core.ErrProtocol, Operation: req.Operation, Message: "httpform: 响应 body 超过 64MiB 上限"}
	}
	return out, nil
}

func readBodyLimited(reader io.Reader, limit int64) ([]byte, bool, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(data)) > limit {
		return data[:limit], true, err
	}
	return data, false, err
}

func (t *Transport) buildClient() *http.Client {
	transport := &http.Transport{}
	if t.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   t.Timeout,
		Transport: transport,
	}
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
