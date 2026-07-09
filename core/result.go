package core

import "time"

// Result 是每个结构化结果都必须实现的最小接口。
type Result interface {
	OperationName() string
	Raw() []byte
	Meta() map[string]string
}

// BaseResult 承载所有 Result 共有的字段。
type BaseResult struct {
	OpName string            // 操作名
	RawDat []byte            // 原始字节
	Metas  map[string]string // 元数据
}

// OperationName 返回操作名。
func (r *BaseResult) OperationName() string { return r.OpName }

// Raw 返回原始字节。
func (r *BaseResult) Raw() []byte { return r.RawDat }

// Meta 返回元数据 map。
func (r *BaseResult) Meta() map[string]string { return r.Metas }

// InfoResult 是系统 / 运行时信息解析后的结构。
type InfoResult struct {
	BaseResult
	OS      string // 操作系统信息
	User    string // 当前用户
	Workdir string // 工作目录
}

// ExecResult 是命令执行后的结构化输出。
type ExecResult struct {
	BaseResult
	Stdout   string // 标准输出
	Stderr   string // 标准错误
	ExitCode int    // 退出码
}

// FileEntry 是目录列表里的一条文件 / 目录项。
type FileEntry struct {
	Name    string    // 文件名
	Path    string    // 路径
	IsDir   bool      // 是否为目录
	Size    int64     // 字节大小
	Mode    string    // 权限字符串（如 0755）
	ModTime time.Time // 修改时间
}

// FileListResult 是目录列表的结果。
type FileListResult struct {
	BaseResult
	Path    string      // 查询的目录路径
	Entries []FileEntry // 目录条目
}

// FileReadResult 是文件读取的结果。
type FileReadResult struct {
	BaseResult
	Path string // 远端文件路径
	Data []byte // 文件内容（二进制安全）
}

// BoolResult 是 成功/失败 + 描述文本 的通用结果。
type BoolResult struct {
	BaseResult
	OK      bool   // 是否成功
	Message string // 远端返回的文本
}

// NewBaseResult 构建一个元数据已初始化的 BaseResult。
func NewBaseResult(op string, raw []byte) BaseResult {
	return BaseResult{
		OpName: op,
		RawDat: raw,
		Metas:  make(map[string]string),
	}
}
