package core

import "context"

// Operation 描述一个远端原子动作（Info、FileRead、Exec、…）。
//
//	适配器负责在 Operation 和 wire 层之间搭建映射。
type Operation interface {
	Name() string
	Build(ctx context.Context, sess *Session) (*Request, error)
	Parse(ctx context.Context, resp *Response) (Result, error)
}

// Capability 表示适配器支持的一种能力。
type Capability string

const (
	CapInfo     Capability = "info"      // 获取系统信息
	CapExec     Capability = "exec"      // 命令执行
	CapFileList Capability = "file.list" // 列目录
	CapFileRead Capability = "file.read" // 读文件
	//CapFileWrite  Capability = "file.write"  // 写文件
	//CapFileDelete Capability = "file.delete" // 删除文件
	CapFileUpload Capability = "file.upload" // 上传文件
)

// CapabilityAware 是可选接口，由需要声明前置能力的 Operation 实现。
type CapabilityAware interface {
	RequiredCapabilities() []Capability
}

// RiskLevel 标记一个操作的危险等级。
type RiskLevel string

const (
	RiskReadOnly    RiskLevel = "read_only"   // 只读
	RiskWrite       RiskLevel = "write"       // 写入
	RiskExec        RiskLevel = "exec"        // 命令执行
	RiskDestructive RiskLevel = "destructive" // 破坏性操作（删除等）
)

// RiskAware 是可选接口，由需要显式声明风险等级的 Operation 实现。
type RiskAware interface {
	RiskLevel() RiskLevel
}
