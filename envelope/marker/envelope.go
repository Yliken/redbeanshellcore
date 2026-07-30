// Package marker 实现了 Python demo 里的 tag_s / tag_e 边界协议
//
//	（Packet.wrap / Packet.extract）。当 payload 需要嵌在 HTML / 文本
//	响应里时，用这一层把真实内容截出来。
//
// 扩展：支持 wire protocol 头部，在 PHP 代码中嵌入 RBS1.0 协议头输出，
// 并解析响应协议头以提供版本、请求 ID、时间戳、nonce、状态码和 HMAC。
package marker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/protocol/wire"
)

type Envelope struct {
	TagLen       int
	WireProtocol bool
}

func New() *Envelope { return &Envelope{TagLen: 16, WireProtocol: false} }

func NewWithWire() *Envelope { return &Envelope{TagLen: 16, WireProtocol: true} }

func NewWithLength(n int) *Envelope {
	if n <= 0 {
		n = 16
	}
	return &Envelope{TagLen: n, WireProtocol: false}
}

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

	if req.Meta["adapter"] != "php" {
		req.Payload = append([]byte(tagS), append(req.Payload, []byte(tagE)...)...)
		return req, nil
	}

	var wrapped string
	if e.WireProtocol {
		key := req.Meta["hmac_key"]
		rid := req.ID
		if rid == "" {
			rid = req.Meta["req_id"]
		}
		ts := req.Meta["ts"]
		nonce := req.Meta["_nonce"]
		if nonce == "" {
			nonce = randomHex(8)
		}

		protoHeader := fmt.Sprintf(
			"echo '%s';"+
				"echo \"\\n\";"+
				"echo 'RID=%s';"+
				"echo \"\\n\";"+
				"echo 'TS=%s';"+
				"echo \"\\n\";"+
				"echo 'NONCE=%s';"+
				"echo \"\\n\";"+
				"echo 'STATUS=0';"+
				"echo \"\\n\";"+
				"echo '"+wire.HeaderDelim+"';"+
				"echo \"\\n\";",
			wire.ResponsePrefix, rid, ts, nonce)

		// 无 HMAC key 时，直接使用简单包裹
		if key == "" {
			wrapped = "echo '" + tagS + "';" + protoHeader + string(req.Payload) + "echo '" + tagE + "';"
		} else {
			// 有 HMAC key 时：先输出协议头，再用 ob_start 捕获 payload 输出用于签名
			bodyVar := phpVar6()
			sigCloser := fmt.Sprintf(
				"$%s=ob_get_clean();"+
					"echo $%s;"+
					"echo \"\\n\";"+
					"echo 'SIG=';"+
					"echo hash_hmac('sha256',$%s,'%s');"+
					"echo '%s';",
				bodyVar, bodyVar, bodyVar, key, tagE)
			wrapped = "echo '" + tagS + "';" + protoHeader + "ob_start();" + string(req.Payload) + sigCloser
		}
	} else {
		wrapped = "echo '" + tagS + "';" + string(req.Payload) + "echo '" + tagE + "';"
	}

	req.Payload = []byte(wrapped)
	return req, nil
}

func (e *Envelope) Extract(_ context.Context, resp *core.Response) (*core.Response, error) {
	if resp == nil {
		return nil, errors.New("marker.Envelope.Extract: 响应不能为空")
	}
	tagS := resp.Meta["marker.tag_s"]
	tagE := resp.Meta["marker.tag_e"]
	if tagS == "" || tagE == "" {
		return resp, fmt.Errorf("marker: missing tag_s or tag_e (tag_s=%q, tag_e=%q)", tagS, tagE)
	}

	start := bytes.Index(resp.Body, []byte(tagS))
	if start < 0 {
		return resp, fmt.Errorf("marker: start tag %q not found", tagS)
	}
	start += len(tagS)
	end := bytes.Index(resp.Body[start:], []byte(tagE))
	if end < 0 {
		return resp, fmt.Errorf("marker: end tag %q not found", tagE)
	}
	rawBody := resp.Body[start : start+end]

	if e.WireProtocol {
		body, header := wire.ParseResponse(rawBody)
		if header != nil {
			resp.SetMeta("proto_version", header.Version)
			resp.SetMeta("proto_rid", header.RID)
			resp.SetMeta("proto_ts", header.TS)
			resp.SetMeta("proto_nonce", header.Nonce)
			resp.SetMeta("proto_status", header.Status)
			resp.SetMeta("proto_sig", header.Sig)

			key := resp.Meta["hmac_key"]
			if key != "" && header.Sig != "" {
				if !wire.VerifyResponseHMAC(body, header.Sig, key) {
					return resp, errors.New("marker: 响应 HMAC 验证失败，响应可能被篡改")
				}
			}

			resp.Body = body
		} else {
			resp.Body = rawBody
		}
	} else {
		resp.Body = rawBody
	}

	return resp, nil
}

func randomHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func phpVar6() string {
	buf := make([]byte, 3)
	_, _ = rand.Read(buf)
	return "x" + hex.EncodeToString(buf)
}
