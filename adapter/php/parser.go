package php

import (
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

const (
	remoteErrorPathUnavailable = "ERROR://REDBEAN:PATH_UNAVAILABLE"
	remoteErrorFileOpen        = "ERROR://REDBEAN:FILE_OPEN_FAILED"
	remoteErrorFileRead        = "ERROR://REDBEAN:FILE_READ_FAILED"
)

var remoteErrorMessages = map[string]string{
	remoteErrorPathUnavailable: "远端路径不存在或无权限",
	remoteErrorFileOpen:        "远端文件无法打开",
	remoteErrorFileRead:        "远端文件读取失败",
}

// Parser 把原始响应字节转成结构化的 core.Result。
// 逻辑对应 Python demo 的 Decoder.decode + 各 operation 的后处理。
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

// ParseInfo 是给 demo 测试用的便捷入口，避免直接 import ops。
func (p *Parser) ParseInfo(body []byte) *core.InfoResult {
	return &core.InfoResult{BaseResult: core.NewBaseResult("info", body)}
}

func parseRemoteError(operation string, resp *core.Response) error {
	if resp == nil {
		return nil
	}
	token := strings.TrimSpace(string(resp.Body))
	message, ok := remoteErrorMessages[token]
	if !ok {
		return nil
	}
	return core.NewOpError(core.ErrRemoteRuntime, operation, resp.NodeID, message+" ("+token+")", nil)
}
