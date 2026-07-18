# 完整使用手册

## 目录

- [1. 核心概念](#1-核心概念)
- [2. Client](#2-client)
- [3. Operation](#3-operation)
- [4. Transport](#4-transport)
- [5. Codec](#5-codec)
- [6. Envelope](#6-envelope)
- [7. Middleware](#7-middleware)
- [8. Error](#8-error)
- [9. Result](#9-result)
- [10. Manager](#10-manager)
- [11. Registry](#11-registry)
- [12. Adapter](#12-adapter)

---

## 1. 核心概念

```
Operation
  -> Build Request
  -> Encode Request         (Codec)
  -> Wrap Envelope          (Envelope)
  -> Apply Transform        (Transform)
  -> Middleware Chain       (Middleware)
  -> Round Trip             (Transport)
  -> Apply Response         (Transform)
  -> Extract Envelope       (Envelope)
  -> Decode Response        (Codec)
  -> Parse Result           (Operation)
```

---

## 2. Client

核心入口，功能选项模式配置：

```go
client := core.NewClient(
    core.WithSession(&core.Session{
        NodeID:   "lab-01",
        Endpoint: "https://lab.example/shell.php",
        Adapter:  "php",
        Metadata: map[string]string{"auth_password_field": "antpwd"},
    }),
    core.WithTransport(httpform.New("https://lab.example/shell.php")),
    // PHP eval 端默认使用 plain codec；只有远端明确支持对应协议时才启用其他 Codec。
    core.WithEnvelope(marker.New()),
    core.WithMiddleware(logging.Middleware()),
    core.WithTransforms(noop.New()),
)

res, err := client.Do(ctx, operation)
```

### Session Metadata

| Key | 说明 |
|-----|------|
| `auth_password_field` | 密码 POST 字段名（默认 `antpwd`）。Transport 会把主 payload 写入这个字段 |

---

## 3. Operation

### PHP 适配器专属（PHP Shell 必须用）

```go
phpshell.NewPhpInfo()                      // 系统信息
phpshell.NewPhpFileList("/var/www")        // 列目录
phpshell.NewPhpFileRead("/etc/passwd")     // 读文件
phpshell.NewPhpFileDownload("/tmp/data.bin") // 下载二进制文件
phpshell.NewPhpFileUpload("/tmp/x.go", data) // 上传文件
phpshell.NewPhpExec("whoami")              // 执行命令
```

链式调用：

```go
// Exec：指定 shell 路径
phpshell.NewPhpExec("dir").WithBin("C:\\Windows\\system32\\cmd.exe")

// Exec：注入环境变量
phpshell.NewPhpExec("make").WithEnv("CC", "gcc")

// Upload：追加模式（默认覆盖）
phpshell.NewPhpFileUpload("/tmp/x.go", data).WithAppend(true)
```

### core 通用操作（仅作抽象层，PHP 环境不要直接用）

```go
ops.NewInfo()
ops.NewExec("cmd")
ops.NewFileList("/path")
ops.NewFileRead("/path")
ops.NewFileUpload("/remote", reader)
ops.NewFileDownload("/path")
```

> ⚠️ 通用 ops 的 `Build` 只生成字面 payload，对 PHP Shell 没用。
> PHP 环境应使用 `phpshell.NewPhp*()`；也可以显式调用 `phpshell.NewClientFactory().WrapOp(op)` 转换受支持的通用 Info/List/Read/Download/Exec。`WrapOp` 不会被 Client 自动调用，通用 Upload 也不会隐式消费 reader。

### 自定义 Operation

```go
type MyOp struct{}
func (o *MyOp) Name() string { return "my.op" }
func (o *MyOp) Build(ctx context.Context, sess *core.Session) (*core.Request, error) { /* ... */ }
func (o *MyOp) Parse(ctx context.Context, resp *core.Response) (core.Result, error) { /* ... */ }
```

---

## 4. Transport

只负责发送/接收，不理解业务。`Request.Params` 会作为表单字段原样提交；PHP 模板需要的 Base64 等编码由 PHP Operation 在 Build 阶段完成。响应 body 超过 64MiB 时返回 `ErrProtocol`，不会静默截断。

```go
// httpform（真实环境）
tr := httpform.New("https://lab.example/shell.php")
tr.Timeout = 30 * time.Second
tr.InsecureTLS = false
tr.ExtraHeaders = map[string]string{"X-Custom": "value"}

// mock（测试用）
tr := transportmock.New(func(ctx context.Context, req *core.Request) (*core.Response, error) {
    resp := core.NewResponse()
    resp.Body = req.Payload
    return resp, nil
})
```

---

## 5. Codec

```go
core.WithCodec(base64.New()) // base64 编解码
// nil 表示 plain（无变换）
```

---

## 6. Envelope

tag_s / tag_e 响应边界协议：

```go
core.WithEnvelope(marker.New())        // 默认 16 字节
core.WithEnvelope(marker.NewWithLength(32))
```

---

## 7. Middleware

按注册顺序包裹请求链。Middleware 包裹 Transport、HTTP 状态映射、响应 Transform、Envelope、Codec 和 Operation.Parse，因此 logging/audit/retry 能观察最终响应错误；Operation.Build 和请求编码仍发生在链外。

```go
core.WithMiddleware(
    logging.Middleware(),                    // 日志
    logging.Middleware(logging.WithLogger(l)), // 自定义 logger
    audit.Middleware(),                      // 审计
    audit.Middleware(audit.WithSink(sink)),  // 自定义 sink
    timeout.Middleware(timeout.Options{Timeout: 15 * time.Second}),
    retry.Middleware(retry.Options{MaxAttempts: 3}),
    readonly.Middleware(),                   // 只读策略
)
```

### Readonly 拦截的操作

`Exec / FileUpload`

---

## 8. Error

```go
type OpError struct {
    Kind      ErrorKind // network_error / timeout / auth_error / ...
    Operation string
    NodeID    string
    Message   string
    Cause     error
}
```

判断错误：

```go
if core.IsKind(err, core.ErrPolicyDenied) { /* 被 readonly 拦截 */ }
if core.IsKind(err, core.ErrNetwork) { /* 网络问题 */ }
```

---

## 9. Result

```go
type InfoResult struct { BaseResult; OS, User, Workdir string }
type ExecResult struct { BaseResult; Stdout, Stderr string; ExitCode int }
type FileListResult struct { BaseResult; Path string; Entries []FileEntry }
type FileReadResult struct { BaseResult; Path string; Data []byte }
type BoolResult struct { BaseResult; OK bool; Message string }
```

通用接口：

```go
res.OperationName() // 操作名
res.Raw()           // 原始字节
res.Meta()          // 元数据
```

类型断言：

```go
switch r := res.(type) {
case *core.InfoResult:    fmt.Println(r.OS, r.User)
case *core.ExecResult:    fmt.Println(r.Stdout, r.ExitCode)
case *core.FileListResult: for _, e := range r.Entries { fmt.Println(e.Name) }
case *core.FileReadResult: os.WriteFile("out", r.Data, 0644)
case *core.BoolResult:    fmt.Println(r.OK, r.Message)
}
```

---

## 10. Manager

多节点注册、索引、Client 创建。

```go
mgr := core.NewManager(registry, factory)

// CRUD
mgr.Register(ctx, NodeConfig{ID, Endpoint, Adapter, Transport, Auth, Options, Tags, Group})
mgr.Unregister(ctx, "lab-a")
mgr.Update(ctx, "lab-a", NodePatch{Status: NodeReady})
mgr.Get(ctx, "lab-a")
mgr.List(ctx, NodeFilter{Tags: []string{"lab"}, Status: NodeReady})

// Client 创建
mgr.Client(ctx, "lab-a")

// 健康检查
mgr.Ping(ctx, "lab-a", phpshell.NewPhpInfo())
mgr.Refresh(ctx, "lab-a", phpshell.NewPhpInfo())

// 批量操作（仅限读）
mgr.DoEach(ctx, filter, func(rec *NodeRecord) Operation {
    return phpshell.NewPhpInfo()
})
```

### NodeConfig 字段

| 字段 | 说明 |
|------|------|
| ID | 唯一标识 |
| Endpoint | 节点 URL |
| Adapter | `php` / `mock` / 自定义 |
| Transport | `httpform` / `mock` / 自定义 |
| Auth | 认证字段 `{param: 密码字段名}` |
| Options | 扩展配置 `{auth_password_field, insecure_tls}` |
| Tags | 标签列表 |
| Group | 分组 |

---

## 11. Registry

```go
// 内存注册表
reg := memory.New()

// JSON 数组文件持久化；内置 Registry 会深复制 NodeRecord 的 map/slice
reg, err := file.New("./nodes.json")

// 接口
reg.Put(ctx, record)
reg.Get(ctx, nodeID)
reg.Delete(ctx, nodeID)
reg.List(ctx, filter)
```

---

## 12. Adapter

自定义适配器需要：

1. `adapter/<name>/adapter.go` — 主适配器
2. `adapter/<name>/renderer.go` — PHP 模板渲染
3. `adapter/<name>/operations.go` — 专属 Operation
4. `adapter/<name>/capabilities.go` — 能力声明

注册 `ClientFactory`，在 Manager 中使用。

示例见 `adapter/php/`。
