// Package marker 实现了 Python demo 里的 tag_s / tag_e 边界协议
//
//	（Packet.wrap / Packet.extract）。当 payload 需要嵌在 HTML / 文本
//	响应里时，用这一层把真实内容截出来。
package marker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
)

// Envelope 用一对随机生成的 tag_s / tag_e 把请求包起来，再从响应中截取
//
//	两个标记之间的内容。
type Envelope struct {
	TagLen int // 标记长度（字节数）
}

// New 构建一个默认 tag 长度（16）的 Envelope。
func New() *Envelope { return &Envelope{TagLen: 16} }

// NewWithLength 构建一个自定义 tag 长度的 Envelope。
func NewWithLength(n int) *Envelope {
	if n <= 0 {
		n = 16
	}
	return &Envelope{TagLen: n}
}

// generate 生成一个指定长度的随机十六进制标记。
func (e *Envelope) generate() (string, error) {
	bytesNeeded := e.TagLen / 2
	if e.TagLen%2 != 0 {
		bytesNeeded++
	}
	buf := make([]byte, bytesNeeded)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("marker.Envelope: 生成标记失败: %w", err)
	}
	out := hex.EncodeToString(buf)
	if len(out) > e.TagLen {
		out = out[:e.TagLen]
	}
	return out, nil
}

// Wrap 把 tag_s / tag_e 写入 req.Meta，并让远端输出包含这对标记。
// PHP payload 使用 echo 语句；其他 payload 保留通用字节前后缀行为。
func (e *Envelope) Wrap(_ context.Context, req *core.Request) (*core.Request, error) {
	if req == nil {
		return nil, errors.New("marker.Envelope.Wrap: 请求不能为空")
	}
	tagS, err := e.generate()
	if err != nil {
		return nil, err
	}
	tagE, err := e.generate()
	if err != nil {
		return nil, err
	}
	req.SetMeta("marker.tag_s", tagS)
	req.SetMeta("marker.tag_e", tagE)
	if req.Meta["adapter"] == "php" {
		wrapped := make([]byte, 0, len(req.Payload)+len(tagS)+len(tagE)+18)
		wrapped = append(wrapped, []byte("echo '")...)
		wrapped = append(wrapped, tagS...)
		wrapped = append(wrapped, []byte("';")...)
		wrapped = append(wrapped, req.Payload...)
		wrapped = append(wrapped, []byte("echo '")...)
		wrapped = append(wrapped, tagE...)
		wrapped = append(wrapped, []byte("';")...)
		req.Payload = wrapped
	} else {
		// 非可执行 payload 保留通用字节前后缀行为。
		req.Payload = append([]byte(tagS), append(req.Payload, []byte(tagE)...)...)
	}
	return req, nil
}

// Extract 从响应 body 中截取 tag_s 与 tag_e 之间的内容。
//
//	如果找不到标记，则原样返回 body，便于调用方排查问题。
func (e *Envelope) Extract(_ context.Context, resp *core.Response) (*core.Response, error) {
	if resp == nil {
		return nil, errors.New("marker.Envelope.Extract: 响应不能为空")
	}
	tagS := resp.Meta["marker.tag_s"]
	tagE := resp.Meta["marker.tag_e"]
	if tagS == "" || tagE == "" {
		return resp, nil
	}
	start := bytes.Index(resp.Body, []byte(tagS))
	if start < 0 {
		return resp, nil
	}
	start += len(tagS)
	end := bytes.Index(resp.Body[start:], []byte(tagE))
	if end < 0 {
		return resp, nil
	}
	resp.Body = resp.Body[start : start+end]
	return resp, nil
}
