package memory

import (
	"context"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestDeepCloneIsolation(t *testing.T) {
	registry := New()
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

	got, err := registry.Get(ctx, "n1")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if got.Config.Auth["field"] != "a" || got.Config.Options["tls"] != "false" || got.Config.Tags[0] != "lab" || got.Capabilities[0] != core.CapInfo || got.Metadata["os"] != "Linux" {
		t.Fatalf("Put 输入未隔离: %+v", got)
	}
	got.Metadata["os"] = "mutated"
	got.Config.Tags[0] = "mutated"
	again, _ := registry.Get(ctx, "n1")
	if again.Metadata["os"] != "Linux" || again.Config.Tags[0] != "lab" {
		t.Fatalf("Get 输出未隔离: %+v", again)
	}
	listed, _ := registry.List(ctx, core.NodeFilter{})
	listed[0].Config.Auth["field"] = "mutated"
	again, _ = registry.Get(ctx, "n1")
	if again.Config.Auth["field"] != "a" {
		t.Fatalf("List 输出未隔离: %+v", again)
	}
}

func TestPutGetDelete(t *testing.T) {
	r := New()
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
