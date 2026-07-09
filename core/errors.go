package core

import (
	"errors"
	"fmt"
)

// ErrorKind 表示 SDK 中某一类错误的分类。
type ErrorKind string

const (
	ErrNetwork       ErrorKind = "network_error"        // 网络层错误
	ErrTimeout       ErrorKind = "timeout"              // 超时
	ErrAuth          ErrorKind = "auth_error"           // 认证失败
	ErrProtocol      ErrorKind = "protocol_error"       // 协议 / 构建阶段错误
	ErrEnvelope      ErrorKind = "envelope_error"       // 边界标记错误
	ErrEncode        ErrorKind = "encode_error"         // 编码错误
	ErrDecode        ErrorKind = "decode_error"         // 解码错误
	ErrParse         ErrorKind = "parse_error"          // 结果解析错误
	ErrPermission    ErrorKind = "permission_denied"    // 远端拒绝
	ErrNotFound      ErrorKind = "not_found"            // 资源不存在
	ErrRemoteRuntime ErrorKind = "remote_runtime_error" // 远端运行时异常
	ErrPolicyDenied  ErrorKind = "policy_denied"        // 策略拒绝（例如 readonly）
)

// OpError 是 SDK 返回的统一错误类型。
type OpError struct {
	Kind      ErrorKind // 错误分类
	Operation string    // 触发错误的操作名
	NodeID    string    // 节点 ID（若已知）
	Message   string    // 人类可读的错误描述
	Cause     error     // 底层错误（若存在）
}

// error 接口实现。
func (e *OpError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] op=%s node=%s: %s (cause: %v)", e.Kind, e.Operation, e.NodeID, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] op=%s node=%s: %s", e.Kind, e.Operation, e.NodeID, e.Message)
}

// Unwrap 让 errors.Is / errors.As 能继续向下解包。
func (e *OpError) Unwrap() error {
	return e.Cause
}

// NewOpError 是 OpError 的构造函数。
func NewOpError(kind ErrorKind, op, nodeID, message string, cause error) *OpError {
	return &OpError{
		Kind:      kind,
		Operation: op,
		NodeID:    nodeID,
		Message:   message,
		Cause:     cause,
	}
}

// IsKind 判断 err 是否为指定分类的 *OpError。
func IsKind(err error, kind ErrorKind) bool {
	var oe *OpError
	if errors.As(err, &oe) {
		return oe.Kind == kind
	}
	return false
}
