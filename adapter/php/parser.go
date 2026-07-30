package php

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// ErrorPrefix 为每次请求生成唯一的错误前缀，替代固定的 ERROR://REDBEAN。
// 在 PHP 侧拼接到错误消息上，客户端按此前缀识别远端错误。
type ErrorPrefix struct {
	value string
}

// NewErrorPrefix 生成一个随机的 8 字节十六进制错误前缀。
func NewErrorPrefix() ErrorPrefix {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return ErrorPrefix{value: "ERR:" + hex.EncodeToString(buf) + ":"}
}

// String 返回错误前缀字符串。
func (e ErrorPrefix) String() string { return e.value }

// Parser 把原始响应字节转成结构化的 core.Result。
type Parser struct{}

// NewParser 构建一个 php Parser 实例。
func NewParser() *Parser { return &Parser{} }

// PayloadFor 暴露模板渲染管线，让测试 / adapter 客户端可以端到端地
// 跑一遍渲染流程，而不用真的起 HTTP 服务。
func (p *Parser) PayloadFor(name string, fn func() (string, map[string]string)) (string, map[string]string) {
	code, placeholders := fn()
	_ = name
	return code, placeholders
}

// ParseInfo 是给 demo 测试用的便捷入口。
func (p *Parser) ParseInfo(body []byte) *core.InfoResult {
	return &core.InfoResult{BaseResult: core.NewBaseResult("info", body)}
}

// RemoteErrorPrefix 返回当前请求的错误前缀，用于在响应中匹配远端错误。
// 需要在 Operation.Build 中注入 req.Meta["remote_error_prefix"]。
func RemoteErrorPrefix(resp *core.Response) string {
	if resp == nil && resp.Meta == nil {
		return ""
	}
	return resp.Meta["remote_error_prefix"]
}

func parseRemoteError(operation string, resp *core.Response) error {
	if resp == nil {
		return nil
	}
	body := strings.TrimSpace(string(resp.Body))

	// 兼容旧版固定错误标记
	if strings.HasPrefix(body, "ERROR://REDBEAN:") {
		msg := describeRedbeanError(body)
		return core.NewOpError(core.ErrRemoteRuntime, operation, resp.NodeID, msg+" ("+body+")", nil)
	}
	// 通用 ERR: 前缀（与其他 adapter 保持一致）
	if strings.HasPrefix(body, "ERR:") {
		return core.NewOpError(core.ErrRemoteRuntime, operation, resp.NodeID, "remote error: "+body, nil)
	}

	// 新版动态错误前缀
	prefix := resp.Meta["remote_error_prefix"]
	if prefix != "" && strings.HasPrefix(body, prefix) {
		return core.NewOpError(core.ErrRemoteRuntime, operation, resp.NodeID,
			"远端返回错误: "+body, nil)
	}

	return nil
}

func describeRedbeanError(token string) string {
	switch token {
	case "ERROR://REDBEAN:PATH_UNAVAILABLE":
		return "远端路径不存在或无权限"
	case "ERROR://REDBEAN:FILE_OPEN_FAILED":
		return "远端文件无法打开"
	case "ERROR://REDBEAN:FILE_READ_FAILED":
		return "远端文件读取失败"
	default:
		return "远端返回未知错误"
	}
}
