package file

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func newFileRegistry(t *testing.T) (*Registry, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "nodes.json")
	r, err := New(path)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	return r, path
}

func TestPutGetDelete(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()
	rec := &core.NodeRecord{Config: core.NodeConfig{ID: "n1", Adapter: "php", Tags: []string{"lab"}}}

	if err := r.Put(ctx, rec); err != nil {
		t.Fatalf("Put 失败: %v", err)
	}
	got, err := r.Get(ctx, "n1")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.Config.ID != "n1" {
		t.Fatalf("ID 不符合预期: %q", got.Config.ID)
	}

	list, err := r.List(ctx, core.NodeFilter{Tags: []string{"lab"}})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("应返回 1 条记录，实际返回 %d", len(list))
	}

	if err := r.Delete(ctx, "n1"); err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if _, err := r.Get(ctx, "n1"); !core.IsKind(err, core.ErrNotFound) {
		t.Fatalf("删除后应返回 ErrNotFound，实际返回 %v", err)
	}
}

func TestAtomicWrite_PersistedToDisk(t *testing.T) {
	r, path := newFileRegistry(t)
	ctx := context.Background()

	rec := &core.NodeRecord{Config: core.NodeConfig{ID: "persist-me", Adapter: "php"}}
	if err := r.Put(ctx, rec); err != nil {
		t.Fatalf("Put 失败: %v", err)
	}

	// 文件应存在且非空
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("文件为空")
	}

	// 同路径 New 一条新 Registry 应能加载回来
	r2, err := New(path)
	if err != nil {
		t.Fatalf("从磁盘重新加载失败: %v", err)
	}
	got, err := r2.Get(ctx, "persist-me")
	if err != nil {
		t.Fatalf("新 Registry 里找不到记录: %v", err)
	}
	if got.Config.ID != "persist-me" || got.Config.Adapter != "php" {
		t.Fatalf("字段不一致: %+v", got.Config)
	}
}

func TestList_FilterByID(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	for _, id := range []string{"a", "b", "c"} {
		if err := r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: id}}); err != nil {
			t.Fatalf("Put(%s) 失败: %v", id, err)
		}
	}

	list, err := r.List(ctx, core.NodeFilter{IDs: []string{"a", "c"}})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应返回 2 条，实际 %d", len(list))
	}
	gotIDs := map[string]bool{}
	for _, rec := range list {
		gotIDs[rec.Config.ID] = true
	}
	if !gotIDs["a"] || !gotIDs["c"] || gotIDs["b"] {
		t.Fatalf("过滤结果不符: %+v", gotIDs)
	}
}

func TestList_FilterByTag_ALL(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n1", Tags: []string{"lab", "php"}}})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n2", Tags: []string{"lab", "jsp"}}})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n3", Tags: []string{"production"}}})

	// 同时带 lab AND php
	list, err := r.List(ctx, core.NodeFilter{Tags: []string{"lab", "php"}})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 1 || list[0].Config.ID != "n1" {
		t.Fatalf("应只命中 n1，got %d 条", len(list))
	}
}

func TestList_FilterByStatus(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "ok"}, Status: core.NodeReady})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "bad"}, Status: core.NodeDown})

	list, err := r.List(ctx, core.NodeFilter{Status: core.NodeReady})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 1 || list[0].Config.ID != "ok" {
		t.Fatalf("应只命中 ok，got %d 条", len(list))
	}
}

func TestNewInMemory_Ephemeral(t *testing.T) {
	r := NewInMemory()
	ctx := context.Background()

	rec := &core.NodeRecord{Config: core.NodeConfig{ID: "volatile"}}
	if err := r.Put(ctx, rec); err != nil {
		t.Fatalf("Put 失败: %v", err)
	}
	got, err := r.Get(ctx, "volatile")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.Config.ID != "volatile" {
		t.Fatalf("ID 不符合预期")
	}
}

func TestCorruptFile_StartsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("写坏文件失败: %v", err)
	}

	_, err := New(path)
	if err == nil {
		t.Fatal("损坏文件应返回错误")
	}
}

func TestList_FilterByGroup(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n1", Group: "case-001"}})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n2", Group: "case-002"}})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n3", Group: "case-001"}})

	list, err := r.List(ctx, core.NodeFilter{Group: "case-001"})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应返回 2 条，实际 %d", len(list))
	}
}

func TestList_FilterByAdapter(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n1", Adapter: "php"}})
	r.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: "n2", Adapter: "jsp"}})

	list, err := r.List(ctx, core.NodeFilter{Adapter: "php"})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 1 || list[0].Config.ID != "n1" {
		t.Fatalf("应只命中 n1，got %d 条", len(list))
	}
}

func TestCloneRecord_IsolatesMutations(t *testing.T) {
	r, _ := newFileRegistry(t)
	ctx := context.Background()

	rec := &core.NodeRecord{Config: core.NodeConfig{ID: "mut", Tags: []string{"x"}}}
	r.Put(ctx, rec)

	got, _ := r.Get(ctx, "mut")
	got.Config.Tags = append(got.Config.Tags, "y")

	again, _ := r.Get(ctx, "mut")
	if len(again.Config.Tags) != 1 {
		t.Fatalf("cloneRecord 没有隔离可变字段，Tags 被改成 %v", again.Config.Tags)
	}
}

func TestPut_NilRecord(t *testing.T) {
	r, _ := newFileRegistry(t)
	err := r.Put(context.Background(), nil)
	if err == nil {
		t.Fatal("Put(nil) 应返回错误")
	}
}

func TestDeepCloneIsolation(t *testing.T) {
	registry, _ := newFileRegistry(t)
	ctx := context.Background()
	record := &core.NodeRecord{
		Config: core.NodeConfig{
			ID:      "n1",
			Auth:    map[string]string{"field": "a"},
			Options: map[string]string{"tls": "false"},
			Tags:    []string{"lab"},
		},
		Capabilities: []core.Capability{core.CapInfo},
		Metadata:     map[string]string{"os": "Linux"},
	}
	if err := registry.Put(ctx, record); err != nil {
		t.Fatalf("Put 失败: %v", err)
	}
	record.Config.Auth["field"] = "changed"
	record.Config.Options["tls"] = "changed"
	record.Config.Tags[0] = "changed"
	record.Capabilities[0] = core.CapExec
	record.Metadata["os"] = "changed"

	got, _ := registry.Get(ctx, "n1")
	if got.Config.Auth["field"] != "a" || got.Config.Options["tls"] != "false" || got.Config.Tags[0] != "lab" || got.Capabilities[0] != core.CapInfo || got.Metadata["os"] != "Linux" {
		t.Fatalf("Put 输入未隔离: %+v", got)
	}
	got.Metadata["os"] = "mutated"
	got.Config.Tags[0] = "mutated"
	again, _ := registry.Get(ctx, "n1")
	if again.Metadata["os"] != "Linux" || again.Config.Tags[0] != "lab" {
		t.Fatalf("Get 输出未隔离: %+v", again)
	}
}

func TestLoadRejectsNullRecord(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "nodes.json")
	if err := os.WriteFile(path, []byte(`[null]`), 0o600); err != nil {
		t.Fatalf("写测试文件失败: %v", err)
	}
	if _, err := New(path); err == nil {
		t.Fatal("包含 null record 的 JSON 应返回错误")
	}
}

func TestConcurrentPutPersistsAllRecords(t *testing.T) {
	registry, path := newFileRegistry(t)
	ctx := context.Background()
	const count = 20
	var wait sync.WaitGroup
	errorsCh := make(chan error, count)
	for index := 0; index < count; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			errorsCh <- registry.Put(ctx, &core.NodeRecord{Config: core.NodeConfig{ID: string(rune('a' + index))}})
		}(index)
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("并发 Put 失败: %v", err)
		}
	}
	reloaded, err := New(path)
	if err != nil {
		t.Fatalf("重新加载失败: %v", err)
	}
	records, err := reloaded.List(ctx, core.NodeFilter{})
	if err != nil || len(records) != count {
		t.Fatalf("持久化记录不完整: count=%d err=%v", len(records), err)
	}
	leftovers, err := filepath.Glob(path + ".tmp-*")
	if err != nil || len(leftovers) != 0 {
		t.Fatalf("不应残留临时文件: %v err=%v", leftovers, err)
	}
}

func TestPutRollsBackWhenFlushFails(t *testing.T) {
	directory := t.TempDir()
	registry := &Registry{
		path:    filepath.Join(directory, "missing", "nodes.json"),
		records: make(map[string]*core.NodeRecord),
	}
	err := registry.Put(context.Background(), &core.NodeRecord{Config: core.NodeConfig{ID: "n1"}})
	if err == nil {
		t.Fatal("临时目录不存在时 Put 应失败")
	}
	if _, getErr := registry.Get(context.Background(), "n1"); !core.IsKind(getErr, core.ErrNotFound) {
		t.Fatalf("flush 失败后内存修改应回滚，got %v", getErr)
	}
}
