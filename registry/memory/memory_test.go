package memory

import (
	"context"
	"testing"

	"github.com/yliken/redbeanshellcore/core"
)

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
