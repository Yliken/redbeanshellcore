// Package httpform 实现把 POST 请求以 application/x-www-form-urlencoded
//  形式发出去的 Transport，行为上对应 AntSword 的 PHP 客户端。
//
// req.Params 里每一条 kv 都会作为一个表单字段；Metadata 里的
//  auth_password_field（或由 Manager 注入到 Meta 中）决定哪个字段携带
//  主 payload（通常是密码字段名，PHP Shell 会 eval 该字段的值）。
//
// ⚠️ 仅用于授权的实验室环境。InsecureTLS 默认关闭，调用方需自行显式开启。
package httpform

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

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
//  req.Params 作为额外字段。响应 body 最大读取 64MiB。
func (t *Transport) RoundTrip(ctx context.Context, req *core.Request) (*core.Response, error) {
	if t.Endpoint == "" {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: endpoint 为空"}
	}

	form := url.Values{}
	// 主 payload 写入密码字段；字段名由 Metadata["auth_password_field"] 决定。
	if field := req.Meta["auth_password_field"]; field != "" {
		form.Set(field, string(req.Payload))
	} else {
		form.Set("antpwd", string(req.Payload))
	}
	// 把额外参数按 base64 解码后放入字段，这与 AntSword 一致。
	for k, v := range req.Params {
		form.Set(k, string(v))
	}
	// 处理 env 模板变量：|||asline||| 分隔成对，每对 |||askey||| 分隔成 kv。
	if env := req.Meta["env_template_vars"]; env != "" {
		for _, kv := range strings.Split(env, "|||asline|||") {
			if k, v, ok := strings.Cut(kv, "|||askey|||"); ok {
				form.Set(k, v)
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
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	for k, v := range t.ExtraHeaders {
		httpReq.Header.Set(k, v)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: POST 失败", Cause: err}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024))
	if err != nil {
		return nil, &core.OpError{Kind: core.ErrNetwork, Operation: req.Operation, Message: "httpform: 读取 body 失败", Cause: err}
	}
	out := core.NewResponse()
	out.RequestID = req.ID
	out.StatusCode = resp.StatusCode
	out.Body = data
	out.Headers = flattenHeaders(resp.Header)
	return out, nil
}

func (t *Transport) buildClient() *http.Client {
	tr := &http.Transport{}
	if t.InsecureTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Timeout:   t.Timeout,
		Transport: tr,
	}
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) == 0 {
			continue
		}
		out[k] = v[0]
	}
	return out
}

var _ = errors.New
var _ = fmt.Sprintf
