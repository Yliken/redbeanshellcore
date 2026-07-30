package core

import (
	"context"
	"errors"
	"fmt"
	"time"
	"sync"
)

// NodeStatus 标记已注册节点的可达性。
type NodeStatus string

const (
	NodeUnknown NodeStatus = "unknown" // 未知
	NodeReady   NodeStatus = "ready"   // 可达
	NodeDown    NodeStatus = "down"    // 不可达
	NodeError   NodeStatus = "error"   // 错误
	NodeFrozen  NodeStatus = "frozen"  // 冻结
)

// NodeConfig 是注册节点时输入的配置。
type NodeConfig struct {
	ID        string            // 节点唯一 ID
	Name      string            // 可读名称
	Endpoint  string            // 节点入口 URL
	Adapter   string            // 适配器类型（php / mock / …）
	Transport string            // 传输类型（httpform / mock / …）
	Codec     string            // 编码方式
	Envelope  string            // 边界协议
	Auth      map[string]string // 认证字段映射，常用键 "payload_form_field" 指定远端
	//   PHP 主 payload 的 POST 字段名（对应 AntSword 密码字段）
	Options map[string]string // adapter / transport 扩展配置
	Tags    []string          // 标签，用于查询 / 分组
	Group   string            // 逻辑分组（例如某次测试任务 / 靶场）
}

// NodePatch 用于局部更新一条 NodeRecord。
type NodePatch struct {
	Name     *string
	Endpoint *string
	Options  map[string]string
	Tags     []string
	Group    *string
	Status   NodeStatus
	Metadata map[string]string
}

// NodeRecord 是注册表中持久化存放的节点记录。
type NodeRecord struct {
	Config       NodeConfig
	Status       NodeStatus
	Capabilities []Capability
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastSeenAt   time.Time
	Metadata     map[string]string
}

// NodeFilter 用于从注册表查询一批节点。
type NodeFilter struct {
	IDs     []string // ID 列表（OR 任意匹配）
	Adapter string   // 适配器类型
	Tags    []string // 标签（AND 全部匹配）
	Group   string   // 分组
	Status  NodeStatus
}

// ClientFactory 根据 NodeRecord 创建一个可用的 Client。
//
//	Manager 把组装 Client 的工作委托给 ClientFactory。
type ClientFactory interface {
	NewClient(ctx context.Context, record *NodeRecord) (*Client, error)
}

// Manager 是轻量级多节点会话注册表，不会编排任务、不做 UI。
type Manager struct {
	registry Registry
	factory  ClientFactory
}

// NewManager 构建一个 Manager。factory 为 nil 时会使用 DefaultClientFactory。
func NewManager(r Registry, f ClientFactory) (*Manager, error) {
	if r == nil {
		return nil, fmt.Errorf("remote-node-core: registry cannot be nil")
	}
	m := &Manager{registry: r}
	if f != nil {
		m.factory = f
	} else {
		m.factory = DefaultClientFactory()
	}
	return m, nil
}

// Register 添加一个新节点。
func (m *Manager) Register(ctx context.Context, cfg NodeConfig) error {
	if cfg.ID == "" {
		return errors.New("remote-node-core: 节点 ID 不能为空")
	}
	now := time.Now().UTC()
	rec := &NodeRecord{
		Config:       cfg,
		Status:       NodeUnknown,
		Metadata:     copyMap(cfg.Options),
		Capabilities: make([]Capability, 0),
		CreatedAt:    now,
		UpdatedAt:    now,
		// LastSeenAt intentionally NOT set here — updated on first success
	}
	return m.registry.Put(ctx, rec)
}

// Unregister 从注册表删除一个节点。
func (m *Manager) Unregister(ctx context.Context, nodeID string) error {
	return m.registry.Delete(ctx, nodeID)
}

// Update 局部更新某节点的字段。
func (m *Manager) Update(ctx context.Context, nodeID string, patch NodePatch) error {
	rec, err := m.registry.Get(ctx, nodeID)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("remote-node-core: registry.Get(%q) 返回了 nil", nodeID)
	}
	if patch.Name != nil {
		rec.Config.Name = *patch.Name
	}
	if patch.Endpoint != nil {
		rec.Config.Endpoint = *patch.Endpoint
	}
	if patch.Group != nil {
		rec.Config.Group = *patch.Group
	}
	if patch.Tags != nil {
		rec.Config.Tags = patch.Tags
	}
	if patch.Options != nil {
		if rec.Config.Options == nil {
			rec.Config.Options = make(map[string]string)
		}
		for k, v := range patch.Options {
			rec.Config.Options[k] = v
		}
	}
	if patch.Metadata != nil {
		if rec.Metadata == nil {
			rec.Metadata = make(map[string]string)
		}
		for k, v := range patch.Metadata {
			rec.Metadata[k] = v
		}
	}
	if patch.Status != "" {
		rec.Status = patch.Status
	}
	rec.UpdatedAt = time.Now().UTC()
	return m.registry.Put(ctx, rec)
}

// Get 按 ID 拉取一条记录。
func (m *Manager) Get(ctx context.Context, nodeID string) (*NodeRecord, error) {
	return m.registry.Get(ctx, nodeID)
}

// List 按 filter 查询；空 filter 返回全部。
func (m *Manager) List(ctx context.Context, filter NodeFilter) ([]*NodeRecord, error) {
	return m.registry.List(ctx, filter)
}

// Client 基于最新记录构造一个可用 Client。
func (m *Manager) Client(ctx context.Context, nodeID string) (*Client, error) {
	record, err := m.registry.Get(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("remote-node-core: 无法获取节点 %q: %w", nodeID, err)
	}
	if record == nil {
		return nil, fmt.Errorf("remote-node-core: registry.Get(%q) 返回了 nil", nodeID)
	}
	if record.Status == NodeFrozen {
		return nil, fmt.Errorf("remote-node-core: node %q is frozen, cannot create client", nodeID)
	}
	if isNilInterface(m.factory) {
		return nil, fmt.Errorf("remote-node-core: client factory 未配置")
	}
	client, err := m.factory.NewClient(ctx, record)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, fmt.Errorf("remote-node-core: client factory 为节点 %q 返回了 nil", nodeID)
	}
	return client, nil
}

// Ping 是轻量健康检查，成功置 Ready、失败置 Down，但不会主动删除节点。
func (m *Manager) Ping(ctx context.Context, nodeID string, op Operation) error {
	c, err := m.Client(ctx, nodeID)
	if err != nil {
		return err
	}
	_, err = c.Do(ctx, op)

	rec, getErr := m.registry.Get(ctx, nodeID)
	if getErr != nil || rec == nil {
		return errors.Join(err, fmt.Errorf("remote-node-core: get node %q failed: %w", nodeID, getErr))
	}

	now := time.Now().UTC()
	rec.LastSeenAt = now
	rec.UpdatedAt = now

	if err != nil {
		rec.Status = NodeDown
		rec.LastError = err.Error()
		if putErr := m.registry.Put(ctx, rec); putErr != nil {
			return errors.Join(err, fmt.Errorf("remote-node-core: update node %q to down failed: %w", nodeID, putErr))
		}
		return err
	}

	rec.Status = NodeReady
	rec.LastError = ""
	if putErr := m.registry.Put(ctx, rec); putErr != nil {
		return fmt.Errorf("remote-node-core: update node %q to ready failed: %w", nodeID, putErr)
	}
	return nil
}
// Refresh 重新拉取基础信息 / capabilities 并刷新 NodeRecord。
func (m *Manager) Refresh(ctx context.Context, nodeID string, op Operation) (*NodeRecord, error) {
	c, err := m.Client(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(ctx, op)
	if err != nil {
		if updateErr := m.Update(ctx, nodeID, NodePatch{Status: NodeError}); updateErr != nil {
			return nil, errors.Join(err, fmt.Errorf("remote-node-core: 更新节点 %q 为 error 失败: %w", nodeID, updateErr))
		}
		return nil, err
	}
	rec, err := m.registry.Get(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("remote-node-core: registry.Get(%q) 返回了 nil", nodeID)
	}
	if rec.Metadata == nil {
		rec.Metadata = make(map[string]string)
	}
	if ir, ok := res.(*InfoResult); ok {
		rec.Metadata["os"] = ir.OS
		rec.Metadata["user"] = ir.User
		rec.Metadata["workdir"] = ir.Workdir
	}
	rec.LastSeenAt = time.Now().UTC()
	rec.LastError = ""
	rec.Status = NodeReady
	if err := m.registry.Put(ctx, rec); err != nil {
		return nil, fmt.Errorf("remote-node-core: 持久化节点 %q 刷新结果失败: %w", nodeID, err)
	}
	return rec, nil
}

// BatchResult 是 Manager.DoEach 每条节点的结果。
type BatchResult struct {
	NodeID string
	Result Result
	Error  error
}

// DoEach 对 filter 命中的每个节点执行一次 opFactory 返回的操作。
//
// 注意：DoEach 仅适用于 Info / FileList / FileRead 这类低风险读操作。
//
//	不会重试，也不会主动规避写入类副作用。
func (m *Manager) DoEach(ctx context.Context, filter NodeFilter, opFactory func(*NodeRecord) Operation) ([]BatchResult, error) {
	if opFactory == nil {
		return nil, fmt.Errorf("remote-node-core: opFactory cannot be nil")
	}
	recs, err := m.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// P1.11: Reject non-readonly operations
	if len(recs) > 0 {
		test := opFactory(recs[0])
		if aware, ok := test.(RiskAware); ok && aware.RiskLevel() != RiskReadOnly {
			return nil, fmt.Errorf("remote-node-core: DoEach rejects non-readonly operation (risk=%s)", aware.RiskLevel())
		}
	}

	out := make([]BatchResult, 0, len(recs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // max 10 concurrent

	for index, rec := range recs {
		if rec == nil {
			return nil, fmt.Errorf("remote-node-core: registry.List returned nil at index %d", index)
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(r *NodeRecord) {
			defer wg.Done()
			defer func() { <-sem }()
			c, cerr := m.Client(ctx, r.Config.ID)
			if cerr != nil {
				mu.Lock()
				out = append(out, BatchResult{NodeID: r.Config.ID, Error: cerr})
				mu.Unlock()
				return
			}
			op := opFactory(r)
			// Per-node timeout (P1.11)
			nodeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			res, derr := c.Do(nodeCtx, op)
			mu.Lock()
			out = append(out, BatchResult{NodeID: r.Config.ID, Result: res, Error: derr})
			mu.Unlock()
		}(rec)
	}
	wg.Wait()
	return out, nil
}
func copyMap(in map[string]string) map[string]string {
	if in == nil {
		return make(map[string]string)
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
