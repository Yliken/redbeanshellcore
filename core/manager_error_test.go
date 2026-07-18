package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/registry/memory"
	transportmock "github.com/Yliken/redbeanshellcore/transport/mock"
)

type failingPutRegistry struct {
	base   core.Registry
	failAt int
	puts   int
	err    error
}

func (r *failingPutRegistry) Put(ctx context.Context, record *core.NodeRecord) error {
	r.puts++
	if r.puts == r.failAt {
		return r.err
	}
	return r.base.Put(ctx, record)
}

func (r *failingPutRegistry) Get(ctx context.Context, nodeID string) (*core.NodeRecord, error) {
	return r.base.Get(ctx, nodeID)
}

func (r *failingPutRegistry) Delete(ctx context.Context, nodeID string) error {
	return r.base.Delete(ctx, nodeID)
}

func (r *failingPutRegistry) List(ctx context.Context, filter core.NodeFilter) ([]*core.NodeRecord, error) {
	return r.base.List(ctx, filter)
}

func successfulClient(result core.Result) *core.Client {
	return core.NewClient(core.WithTransport(transportmock.New(func(context.Context, *core.Request) (*core.Response, error) {
		resp := core.NewResponse()
		resp.Body = []byte("ok")
		return resp, nil
	})), core.WithSession(&core.Session{NodeID: "n1"}), core.WithMiddleware(func(next core.Handler) core.Handler {
		return next
	}))
}

func TestManagerPingReturnsStatusPersistenceError(t *testing.T) {
	persistenceErr := errors.New("put failed")
	registry := &failingPutRegistry{base: memory.New(), failAt: 2, err: persistenceErr}
	client := successfulClient(nil)
	manager := core.NewManager(registry, &fakeFactory{client: client})
	if err := manager.Register(context.Background(), core.NodeConfig{ID: "n1"}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	err := manager.Ping(context.Background(), "n1", op)
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("Ping 应返回状态持久化错误，got %v", err)
	}
}

func TestManagerPingJoinsOperationAndPersistenceErrors(t *testing.T) {
	operationErr := errors.New("transport failed")
	persistenceErr := errors.New("put failed")
	registry := &failingPutRegistry{base: memory.New(), failAt: 2, err: persistenceErr}
	client := core.NewClient(core.WithTransport(transportmock.New(transportmock.FailAlways(operationErr))))
	manager := core.NewManager(registry, &fakeFactory{client: client})
	if err := manager.Register(context.Background(), core.NodeConfig{ID: "n1"}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	err := manager.Ping(context.Background(), "n1", op)
	if !errors.Is(err, operationErr) || !errors.Is(err, persistenceErr) {
		t.Fatalf("Ping 应同时保留两个错误，got %v", err)
	}
}

func TestManagerRefreshInitializesMetadata(t *testing.T) {
	registry := memory.New()
	if err := registry.Put(context.Background(), &core.NodeRecord{Config: core.NodeConfig{ID: "n1"}}); err != nil {
		t.Fatalf("seed 失败: %v", err)
	}
	client := successfulClient(nil)
	manager := core.NewManager(registry, &fakeFactory{client: client})
	op := &stubOp{
		name:  "info",
		build: func() (*core.Request, error) { return core.NewRequest("info"), nil },
		parse: func(*core.Response) (core.Result, error) {
			return &core.InfoResult{BaseResult: core.NewBaseResult("info", nil), OS: "Linux"}, nil
		},
	}

	record, err := manager.Refresh(context.Background(), "n1", op)
	if err != nil || record.Metadata["os"] != "Linux" {
		t.Fatalf("Refresh 应初始化 metadata: record=%+v err=%v", record, err)
	}
}

func TestManagerRefreshReturnsFinalPersistenceError(t *testing.T) {
	persistenceErr := errors.New("put failed")
	registry := &failingPutRegistry{base: memory.New(), failAt: 2, err: persistenceErr}
	client := successfulClient(nil)
	manager := core.NewManager(registry, &fakeFactory{client: client})
	if err := manager.Register(context.Background(), core.NodeConfig{ID: "n1"}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	op := &stubOp{name: "info", build: func() (*core.Request, error) { return core.NewRequest("info"), nil }}

	record, err := manager.Refresh(context.Background(), "n1", op)
	if record != nil || !errors.Is(err, persistenceErr) {
		t.Fatalf("Refresh 持久化失败不应返回成功记录: record=%v err=%v", record, err)
	}
}

func TestManagerRejectsNilFactoryClientAndOpFactory(t *testing.T) {
	registry := memory.New()
	manager := core.NewManager(registry, &fakeFactory{})
	if err := manager.Register(context.Background(), core.NodeConfig{ID: "n1"}); err != nil {
		t.Fatalf("Register 失败: %v", err)
	}
	if _, err := manager.Client(context.Background(), "n1"); err == nil {
		t.Fatal("nil client 应返回错误")
	}
	if _, err := manager.DoEach(context.Background(), core.NodeFilter{}, nil); err == nil {
		t.Fatal("nil opFactory 应返回错误")
	}
}
