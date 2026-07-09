// Package file 提供 JSON 文件持久化的注册表，适用于 CLI 工具和简单 SDK 调用。
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Yliken/redbeanshellcore/core"
)

// Registry 把节点记录持久化到单个 JSON 文件。
type Registry struct {
	mu       sync.RWMutex
	path     string
	records  map[string]*core.NodeRecord
	inMemory bool
}

// New 从磁盘路径构建 Registry。文件不存在时首次 Put 会创建它。
func New(path string) (*Registry, error) {
	r := &Registry{
		path:    path,
		records: make(map[string]*core.NodeRecord),
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

// NewInMemory 构建一个不落盘的 Registry，便于测试复用同样的代码路径。
func NewInMemory() *Registry {
	return &Registry{
		inMemory: true,
		records:  make(map[string]*core.NodeRecord),
	}
}

// Put 插入 / 覆盖一条记录，并持久化到磁盘。
func (r *Registry) Put(_ context.Context, rec *core.NodeRecord) error {
	if rec == nil {
		return fmt.Errorf("file.Registry: 记录不能为空")
	}
	r.mu.Lock()
	r.records[rec.Config.ID] = cloneRecord(rec)
	r.mu.Unlock()
	return r.flush()
}

// Get 按 ID 读取记录。
func (r *Registry) Get(_ context.Context, nodeID string) (*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.records[nodeID]
	if !ok {
		return nil, &core.OpError{Kind: core.ErrNotFound, NodeID: nodeID, Message: "节点未注册"}
	}
	return cloneRecord(rec), nil
}

// Delete 按 ID 删除记录。
func (r *Registry) Delete(_ context.Context, nodeID string) error {
	r.mu.Lock()
	delete(r.records, nodeID)
	r.mu.Unlock()
	return r.flush()
}

// List 按 filter 返回命中记录。
func (r *Registry) List(_ context.Context, filter core.NodeFilter) ([]*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*core.NodeRecord, 0, len(r.records))
	for _, rec := range r.records {
		if matchFilter(rec, filter) {
			out = append(out, cloneRecord(rec))
		}
	}
	return out, nil
}

func (r *Registry) load() error {
	if r.inMemory {
		return nil
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("file.Registry: 读取文件失败 %s: %w", r.path, err)
	}
	var recs []*core.NodeRecord
	if err := json.Unmarshal(data, &recs); err != nil {
		return fmt.Errorf("file.Registry: 解析文件失败 %s: %w", r.path, err)
	}
	for _, rec := range recs {
		r.records[rec.Config.ID] = rec
	}
	return nil
}

// flush 通过"写入临时文件 + rename"原子地把数据刷到磁盘。
func (r *Registry) flush() error {
	if r.inMemory {
		return nil
	}
	r.mu.RLock()
	recs := make([]*core.NodeRecord, 0, len(r.records))
	for _, rec := range r.records {
		recs = append(recs, rec)
	}
	r.mu.RUnlock()
	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return fmt.Errorf("file.Registry: 序列化失败: %w", err)
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("file.Registry: 写入临时文件失败 %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		return fmt.Errorf("file.Registry: 覆盖文件失败 %s: %w", r.path, err)
	}
	return nil
}

func cloneRecord(rec *core.NodeRecord) *core.NodeRecord {
	if rec == nil {
		return nil
	}
	cp := *rec
	return &cp
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
