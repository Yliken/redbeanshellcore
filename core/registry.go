package core

import "context"

// Registry 是节点注册表的存储接口。
//  框架目前提供了两个内置实现：memory.Registry（进程内）和
//  file.Registry（磁盘 JSONL）。调用方也可以接入加密 / sqlite / 远端实现。
type Registry interface {
	Put(ctx context.Context, record *NodeRecord) error
	Get(ctx context.Context, nodeID string) (*NodeRecord, error)
	Delete(ctx context.Context, nodeID string) error
	List(ctx context.Context, filter NodeFilter) ([]*NodeRecord, error)
}
