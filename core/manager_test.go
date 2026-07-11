package core_test

import (
	"context"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/registry/memory"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

// fakeFactory 让 Manager.Client() 走得到“成功”路径。
type fakeFactory struct {
	client *core.Client
	err    error
}

func (f *fakeFactory) NewClient(_ context.Context, _ *core.NodeRecord) (*core.Client, error) {
	return f.client, f.err
}

func newManagerWithFakeFactory(t *testing.T, c *core.Client) *core.Manager {
	t.Helper()
	m := core.NewManager(memory.New(), nil)
	// 用 factory 字段是没法直接设的（private），所以回退到使用默认工厂。
	// 这里直接构造一个空 Manager，用反射没意义——改用 NewClient 走"节点不存在"路径
	return m
}

func TestManager_RegisterAndGet(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	err := m.Register(ctx, core.NodeConfig{ID: "n1", Adapter: "php", Endpoint: "http://x/shell.php"})
	if err != nil {
		t.Fatalf("Register 失败: %v", err)
	}

	rec, err := m.Get(ctx, "n1")
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	if rec.Status != core.NodeUnknown {
		t.Fatalf("注册后状态应为 NodeUnknown，got %q", rec.Status)
	}
	if rec.Config.ID != "n1" || rec.Config.Adapter != "php" {
		t.Fatalf("Config 字段不一致: %+v", rec.Config)
	}
}

func TestManager_RegisterDuplicate_Overwrites(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1", Adapter: "php"})
	m.Register(ctx, core.NodeConfig{ID: "n1", Adapter: "jsp"})

	rec, _ := m.Get(ctx, "n1")
	if rec.Config.Adapter != "jsp" {
		t.Fatalf("重复注册应覆盖，got adapter=%q", rec.Config.Adapter)
	}
}

func TestManager_Register_EmptyID(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	err := m.Register(context.Background(), core.NodeConfig{ID: ""})
	if err == nil {
		t.Fatal("空 ID 注册应返回错误")
	}
}

func TestManager_Unregister(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1"})
	if err := m.Unregister(ctx, "n1"); err != nil {
		t.Fatalf("Unregister 失败: %v", err)
	}
	if _, err := m.Get(ctx, "n1"); !core.IsKind(err, core.ErrNotFound) {
		t.Fatalf("删除后应 ErrNotFound，got %v", err)
	}
}

func TestManager_Unregister_NotFound(t *testing.T) {
	// 当前 memory registry 的 Delete 不报 NotFound（幂等），
	// 所以 Unregister 也不会报。行为锁定在此。
	m := core.NewManager(memory.New(), nil)
	err := m.Unregister(context.Background(), "never-existed")
	if err != nil {
		t.Fatalf("当前实现对不存在的节点删除应返回 nil（幂等），got %v", err)
	}
}

func TestManager_Update_NameAndGroup(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1", Group: "g1"})
	name := "renamed"
	group := "g2"
	tags := []string{"lab"}

	err := m.Update(ctx, "n1", core.NodePatch{Name: &name, Group: &group, Tags: tags})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}

	rec, _ := m.Get(ctx, "n1")
	if rec.Config.Name != "renamed" {
		t.Fatalf("Name 未更新: %q", rec.Config.Name)
	}
	if rec.Config.Group != "g2" {
		t.Fatalf("Group 未更新: %q", rec.Config.Group)
	}
	if len(rec.Config.Tags) != 1 || rec.Config.Tags[0] != "lab" {
		t.Fatalf("Tags 未更新: %v", rec.Config.Tags)
	}
}

func TestManager_Update_StatusOnly(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1"})
	if err := m.Update(ctx, "n1", core.NodePatch{Status: core.NodeReady}); err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Status != core.NodeReady {
		t.Fatalf("Status 未更新: %q", rec.Status)
	}
}

func TestManager_Update_MetadataMerge(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1"})
	err := m.Update(ctx, "n1", core.NodePatch{Metadata: map[string]string{"os": "Linux", "user": "root"}})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Metadata["os"] != "Linux" || rec.Metadata["user"] != "root" {
		t.Fatalf("Metadata 未合并: %+v", rec.Metadata)
	}
}

func TestManager_Update_Options(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1"})
	err := m.Update(ctx, "n1", core.NodePatch{Options: map[string]string{"insecure_tls": "true"}})
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Config.Options["insecure_tls"] != "true" {
		t.Fatalf("Options 未更新: %+v", rec.Config.Options)
	}
}

func TestManager_List_FilterByTag(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "a", Tags: []string{"lab"}})
	m.Register(ctx, core.NodeConfig{ID: "b", Tags: []string{"lab"}})
	m.Register(ctx, core.NodeConfig{ID: "c", Tags: []string{"prod"}})

	list, err := m.List(ctx, core.NodeFilter{Tags: []string{"lab"}})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应返回 2 条，got %d", len(list))
	}
	gotIDs := map[string]bool{}
	for _, r := range list {
		gotIDs[r.Config.ID] = true
	}
	if !gotIDs["a"] || !gotIDs["b"] || gotIDs["c"] {
		t.Fatalf("过滤结果不符: %+v", gotIDs)
	}
}

func TestManager_List_FilterByGroup(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "n1", Group: "case-001"})
	m.Register(ctx, core.NodeConfig{ID: "n2", Group: "case-002"})
	m.Register(ctx, core.NodeConfig{ID: "n3", Group: "case-001"})

	list, err := m.List(ctx, core.NodeFilter{Group: "case-001"})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应返回 2 条，got %d", len(list))
	}
}

func TestManager_List_FilterByStatus(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "ok"})
	m.Update(ctx, "ok", core.NodePatch{Status: core.NodeReady})
	m.Register(ctx, core.NodeConfig{ID: "bad"})
	m.Update(ctx, "bad", core.NodePatch{Status: core.NodeDown})

	list, err := m.List(ctx, core.NodeFilter{Status: core.NodeReady})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 1 || list[0].Config.ID != "ok" {
		t.Fatalf("应只命中 ok，got %d 条", len(list))
	}
}

func TestManager_List_FilterByIDs(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "a"})
	m.Register(ctx, core.NodeConfig{ID: "b"})
	m.Register(ctx, core.NodeConfig{ID: "c"})

	list, err := m.List(ctx, core.NodeFilter{IDs: []string{"a", "c"}})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("应返回 2 条，got %d", len(list))
	}
}

func TestManager_List_EmptyFilterReturnsAll(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "a"})
	m.Register(ctx, core.NodeConfig{ID: "b"})

	list, err := m.List(ctx, core.NodeFilter{})
	if err != nil {
		t.Fatalf("List 失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("空 filter 应返回全部 2 条，got %d", len(list))
	}
}

func TestManager_Client_NodeNotFound(t *testing.T) {
	m := core.NewManager(memory.New(), nil)
	_, err := m.Client(context.Background(), "never-existed")
	if err == nil {
		t.Fatal("获取不存在的节点应返回错误")
	}
}

func TestManager_Ping_Success(t *testing.T) {
	// 用 mock transport 构造一个成功 ping 的 client
	tr := transportmock.New(transportmock.EchoHandler)
	c := core.NewClient(core.WithSession(&core.Session{NodeID: "n1"}), core.WithTransport(tr))

	m := core.NewManager(memory.New(), &fakeFactory{client: c})
	ctx := context.Background()
	m.Register(ctx, core.NodeConfig{ID: "n1"})

	err := m.Ping(ctx, "n1", &stubOp{name: "info", build: func() (*core.Request, error) {
		return core.NewRequest("info"), nil
	}})
	if err != nil {
		t.Fatalf("Ping 应成功: %v", err)
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Status != core.NodeReady {
		t.Fatalf("Ping 成功后应 NodeReady，got %q", rec.Status)
	}
}

func TestManager_Ping_Failure(t *testing.T) {
	tr := transportmock.New(transportmock.FailAlways(nil))

	c := core.NewClient(
		core.WithSession(&core.Session{NodeID: "n1"}),
		core.WithTransport(tr))
	m := core.NewManager(memory.New(), &fakeFactory{client: c})
	ctx := context.Background()
	m.Register(ctx, core.NodeConfig{ID: "n1"})

	err := m.Ping(ctx, "n1", &stubOp{name: "info", build: func() (*core.Request, error) {
		return core.NewRequest("info"), nil
	}})
	if err == nil {
		t.Fatal("Ping 应失败")
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Status != core.NodeDown {
		t.Fatalf("Ping 失败后应 NodeDown，got %q", rec.Status)
	}
}

func TestManager_Refresh(t *testing.T) {
	tr := transportmock.New(func(_ context.Context, _ *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		// 模拟 PHP info 输出
		resp.Body = []byte("/var/www\t/\tLinux 5.0\twww-data")
		return resp, nil
	})
	c := core.NewClient(core.WithSession(&core.Session{NodeID: "n1"}), core.WithTransport(tr))

	m := core.NewManager(memory.New(), &fakeFactory{client: c})
	ctx := context.Background()
	m.Register(ctx, core.NodeConfig{ID: "n1"})

	rec, err := m.Refresh(ctx, "n1", &stubOp{
		name:  "info",
		build: func() (*core.Request, error) { return core.NewRequest("info"), nil },
		parse: func(_ *core.Response) (core.Result, error) {
			// 模拟 phpInfo 的 Parse
			return &core.InfoResult{
				BaseResult: core.NewBaseResult("info", nil),
				OS:         "Linux 5.0",
				User:       "www-data",
				Workdir:    "/var/www",
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Refresh 失败: %v", err)
	}
	if rec.Metadata["os"] != "Linux 5.0" {
		t.Fatalf("Refresh 应填充 Metadata[os]: %+v", rec.Metadata)
	}
	if rec.Metadata["user"] != "www-data" {
		t.Fatalf("Refresh 应填充 Metadata[user]: %+v", rec.Metadata)
	}
	if rec.Metadata["workdir"] != "/var/www" {
		t.Fatalf("Refresh 应填充 Metadata[workdir]: %+v", rec.Metadata)
	}
	if rec.Status != core.NodeReady {
		t.Fatalf("Refresh 成功后应 NodeReady，got %q", rec.Status)
	}
}

func TestManager_Refresh_Failure(t *testing.T) {
	tr := transportmock.New(transportmock.FailAlways(nil))
	c := core.NewClient(core.WithSession(&core.Session{NodeID: "n1"}), core.WithTransport(tr))

	m := core.NewManager(memory.New(), &fakeFactory{client: c})
	ctx := context.Background()
	m.Register(ctx, core.NodeConfig{ID: "n1"})

	_, err := m.Refresh(ctx, "n1", &stubOp{name: "info", build: func() (*core.Request, error) {
		return core.NewRequest("info"), nil
	}})
	if err == nil {
		t.Fatal("Refresh 应失败")
	}
	rec, _ := m.Get(ctx, "n1")
	if rec.Status != core.NodeError {
		t.Fatalf("Refresh 失败后应 NodeError，got %q", rec.Status)
	}
}

func TestManager_DoEach(t *testing.T) {
	tr := transportmock.New(transportmock.EchoHandler)
	c := core.NewClient(core.WithSession(&core.Session{NodeID: "n1"}), core.WithTransport(tr))

	m := core.NewManager(memory.New(), &fakeFactory{client: c})
	ctx := context.Background()

	m.Register(ctx, core.NodeConfig{ID: "a", Group: "g1"})
	m.Register(ctx, core.NodeConfig{ID: "b", Group: "g1"})
	m.Register(ctx, core.NodeConfig{ID: "c", Group: "g2"})

	results, err := m.DoEach(ctx, core.NodeFilter{Group: "g1"}, func(_ *core.NodeRecord) core.Operation {
		return &stubOp{
			name:  "info",
			build: func() (*core.Request, error) { return core.NewRequest("info"), nil },
		}
	})
	if err != nil {
		t.Fatalf("DoEach 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("应对 2 个节点执行，got %d", len(results))
	}
	gotIDs := map[string]bool{}
	for _, r := range results {
		gotIDs[r.NodeID] = true
		if r.Error != nil {
			t.Fatalf("节点 %s 出错: %v", r.NodeID, r.Error)
		}
	}
	if !gotIDs["a"] || !gotIDs["b"] || gotIDs["c"] {
		t.Fatalf("DoEach 结果不符: %+v", gotIDs)
	}
}

func TestManager_Client_NodeNotFound_WithFakeFactory(t *testing.T) {
	m := core.NewManager(memory.New(), &fakeFactory{client: &core.Client{}, err: nil})
	_, err := m.Client(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("节点不存在时 Client() 应报错")
	}
}
