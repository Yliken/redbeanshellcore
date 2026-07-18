package core

import "context"

// Registry 是节点注册表的存储接口。
// 框架目前提供 memory.Registry（进程内）和 file.Registry（磁盘 JSON 数组）。
// 内置实现会深复制输入和输出记录；调用方也可以接入加密 / sqlite / 远端实现。
type Registry interface {
	Put(ctx context.Context, record *NodeRecord) error
	Get(ctx context.Context, nodeID string) (*NodeRecord, error)
	Delete(ctx context.Context, nodeID string) error
	List(ctx context.Context, filter NodeFilter) ([]*NodeRecord, error)
}
