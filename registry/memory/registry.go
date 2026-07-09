// Package memory 提供进程内注册表，适用于 CLI 会话、短生命周期测试等不需要持久化的场景。
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Yliken/redbeanshellcore/core"
)

// Registry 是线程安全的进程内节点注册表。
type Registry struct {
	mu      sync.RWMutex
	records map[string]*core.NodeRecord
}

// New 构建一个空 Registry。
func New() *Registry {
	return &Registry{records: make(map[string]*core.NodeRecord)}
}

// Put 插入或覆盖一条记录。
func (r *Registry) Put(_ context.Context, rec *core.NodeRecord) error {
	if rec == nil {
		return fmt.Errorf("memory.Registry: 记录不能为空")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	rp := *rec
	r.records[rp.Config.ID] = &rp
	return nil
}

// Get 按 ID 读取记录；未注册时返回错误。
func (r *Registry) Get(_ context.Context, nodeID string) (*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[nodeID]
	if !ok {
		return nil, &core.OpError{Kind: core.ErrNotFound, NodeID: nodeID, Message: "节点未注册"}
	}
	rp := *rec
	return &rp, nil
}

// Delete 删除一条记录。
func (r *Registry) Delete(_ context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.records, nodeID)
	return nil
}

// List 按 filter 返回全部命中记录。
func (r *Registry) List(_ context.Context, filter core.NodeFilter) ([]*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*core.NodeRecord, 0, len(r.records))
	for _, rec := range r.records {
		if matchFilter(rec, filter) {
			rp := *rec
			out = append(out, &rp)
		}
	}
	return out, nil
}

func matchFilter(rec *core.NodeRecord, filter core.NodeFilter) bool {
	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if id == rec.Config.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Adapter != "" && rec.Config.Adapter != filter.Adapter {
		return false
	}
	if filter.Group != "" && rec.Config.Group != filter.Group {
		return false
	}
	if filter.Status != "" && rec.Status != filter.Status {
		return false
	}
	if len(filter.Tags) > 0 {
		if !hasAllTags(rec.Config.Tags, filter.Tags) {
			return false
		}
	}
	return true
}

func hasAllTags(have, want []string) bool {
	for _, w := range want {
		found := false
		for _, h := range have {
			if h == w {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
