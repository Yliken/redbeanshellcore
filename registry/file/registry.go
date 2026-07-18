// Package file 提供 JSON 文件持久化的注册表，适用于 CLI 工具和简单 SDK 调用。
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	registry := &Registry{
		path:    path,
		records: make(map[string]*core.NodeRecord),
	}
	if err := registry.load(); err != nil {
		return nil, err
	}
	return registry, nil
}

// NewInMemory 构建一个不落盘的 Registry，便于测试复用同样的代码路径。
func NewInMemory() *Registry {
	return &Registry{
		inMemory: true,
		records:  make(map[string]*core.NodeRecord),
	}
}

// Put 插入 / 覆盖一条记录，并持久化到磁盘。
func (r *Registry) Put(_ context.Context, record *core.NodeRecord) error {
	if record == nil {
		return fmt.Errorf("file.Registry: 记录不能为空")
	}
	if record.Config.ID == "" {
		return fmt.Errorf("file.Registry: 节点 ID 不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	previous, existed := r.records[record.Config.ID]
	r.records[record.Config.ID] = core.CloneNodeRecord(record)
	if err := r.flushLocked(); err != nil {
		if existed {
			r.records[record.Config.ID] = previous
		} else {
			delete(r.records, record.Config.ID)
		}
		return err
	}
	return nil
}

// Get 按 ID 读取记录。
func (r *Registry) Get(_ context.Context, nodeID string) (*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, ok := r.records[nodeID]
	if !ok {
		return nil, &core.OpError{Kind: core.ErrNotFound, NodeID: nodeID, Message: "节点未注册"}
	}
	return core.CloneNodeRecord(record), nil
}

// Delete 按 ID 删除记录。
func (r *Registry) Delete(_ context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	previous, existed := r.records[nodeID]
	delete(r.records, nodeID)
	if err := r.flushLocked(); err != nil {
		if existed {
			r.records[nodeID] = previous
		}
		return err
	}
	return nil
}

// List 按 filter 返回命中记录。
func (r *Registry) List(_ context.Context, filter core.NodeFilter) ([]*core.NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*core.NodeRecord, 0, len(r.records))
	for _, record := range r.records {
		if matchFilter(record, filter) {
			out = append(out, core.CloneNodeRecord(record))
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
	var records []*core.NodeRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("file.Registry: 解析文件失败 %s: %w", r.path, err)
	}
	for index, record := range records {
		if record == nil {
			return fmt.Errorf("file.Registry: 第 %d 条记录为 null", index)
		}
		if record.Config.ID == "" {
			return fmt.Errorf("file.Registry: 第 %d 条记录的节点 ID 为空", index)
		}
		r.records[record.Config.ID] = core.CloneNodeRecord(record)
	}
	return nil
}

// flushLocked 通过“写入唯一临时文件 + rename”把数据刷到磁盘。
// 调用方必须持有 r.mu 的写锁。
func (r *Registry) flushLocked() error {
	if r.inMemory {
		return nil
	}
	ids := make([]string, 0, len(r.records))
	for id := range r.records {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	records := make([]*core.NodeRecord, 0, len(ids))
	for _, id := range ids {
		records = append(records, r.records[id])
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("file.Registry: 序列化失败: %w", err)
	}

	directory := filepath.Dir(r.path)
	tempFile, err := os.CreateTemp(directory, filepath.Base(r.path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("file.Registry: 创建临时文件失败 %s: %w", directory, err)
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if err := tempFile.Chmod(0o600); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("file.Registry: 设置临时文件权限失败 %s: %w", tempPath, err)
	}
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("file.Registry: 写入临时文件失败 %s: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("file.Registry: 关闭临时文件失败 %s: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, r.path); err != nil {
		return fmt.Errorf("file.Registry: 覆盖文件失败 %s: %w", r.path, err)
	}
	cleanup = false
	return nil
}

func matchFilter(record *core.NodeRecord, filter core.NodeFilter) bool {
	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if id == record.Config.ID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if filter.Adapter != "" && record.Config.Adapter != filter.Adapter {
		return false
	}
	if filter.Group != "" && record.Config.Group != filter.Group {
		return false
	}
	if filter.Status != "" && record.Status != filter.Status {
		return false
	}
	if len(filter.Tags) > 0 && !hasAllTags(record.Config.Tags, filter.Tags) {
		return false
	}
	return true
}

func hasAllTags(have, want []string) bool {
	for _, expected := range want {
		found := false
		for _, actual := range have {
			if actual == expected {
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
