# 适配器开发指南

适配器是把通用 `core.Operation` 映射为某种服务端语言或 Shell 协议的边界层。它负责模板、参数编码、错误标记和结果解析；`core` 只负责生命周期、组件编排和错误归一化。

## 1. 推荐目录结构

```text
adapter/<name>/
  adapter.go          # 模板、Parser、Capabilities 的组合
  capabilities.go    # 能力声明
  operations.go      # Name / Build / Parse
  templates.go       # 服务端源代码模板
  parser.go          # 响应解析和远端错误识别
  client_factory.go  # 从 NodeRecord 组装 Client
  obfuscate.go       # 可选：部署与客户端共用的命名状态
```

PHP、ASP、ASPX 和 JSP 都采用这一边界。`adapter/mock` 只用于测试。

## 2. Operation 契约

`Build` 应返回已经具备服务端语义的 `Request`：

- `Payload`：服务端待执行的代码或动作码；
- `Params`：额外参数，二进制内容先在适配器中编码；
- `Meta["adapter"]`：适配器名；
- `Meta["payload_form_field"]`：需要自定义主 payload 表单字段时设置；
- `RiskLevel`：声明只读、写入或执行风险。

`Parse` 必须先处理空响应和远端错误，再构造强类型 Result。文件内容使用 `[]byte` 保持二进制安全；不要在适配器层把下载内容强制转换为 UTF-8。

示例：

```go
type infoOperation struct{}

func (o *infoOperation) Name() string { return "info" }
func (o *infoOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (o *infoOperation) Build(context.Context, *core.Session) (*core.Request, error) {
    req := core.NewRequest(o.Name())
    req.Payload = []byte(renderInfo())
    req.Meta["adapter"] = "example"
    return req, nil
}

func (o *infoOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
    if resp == nil { return nil, errors.New("response is nil") }
    return &core.InfoResult{
        BaseResult: core.NewBaseResult(o.Name(), resp.Body),
        OS: string(resp.Body),
    }, nil
}
```

## 3. 能力声明

能力是 `core.Capability`：`CapInfo`、`CapExec`、`CapFileList`、`CapFileRead`、`CapFileUpload`。适配器应让 `Capabilities()` 与实际可构造的操作保持一致；不要声明尚未实现的能力。

```go
type Capabilities struct{}

func (Capabilities) All() []core.Capability {
    return []core.Capability{
        core.CapInfo, core.CapExec, core.CapFileList,
        core.CapFileRead, core.CapFileUpload,
    }
}
```

需要在调用前校验能力时，让 Operation 实现 `RequiredCapabilities()`，并在应用层根据 Session 能力快照拒绝不支持的请求。

## 4. ClientFactory

Factory 接收 `*core.NodeRecord`，创建 Transport、Session 和可选的 Envelope：

```go
type ClientFactory struct{}

func (f *ClientFactory) NewClient(_ context.Context, rec *core.NodeRecord) (*core.Client, error) {
    if rec == nil || rec.Config.Endpoint == "" {
        return nil, errors.New("endpoint is required")
    }
    tr := httpform.New(rec.Config.Endpoint)
    sess := core.NewSession(rec.Config.ID, rec.Config.Endpoint)
    sess.Adapter = rec.Config.Adapter
    sess.Transport = rec.Config.Transport
    sess.Metadata = clone(rec.Metadata)
    for k, v := range rec.Config.Auth { sess.Metadata[k] = v }
    return core.NewClient(
        core.WithSession(sess),
        core.WithTransport(tr),
    ), nil
}
```

生产 Factory 还应解析 `rec.Config.Options`，例如超时、TLS、Wire Protocol 和 HMAC key。内置行为：

- PHP、ASP、ASPX 的 `NewClientFactory` 支持默认 `httpform`；
- JSP 的 `NewClientFactory.NewClient` 明确不构建 Client，必须由调用方手动提供 Transport 和 Session；
- `core.DefaultClientFactory` 只提供选择器骨架，内置 `httpform` 和 `mock` 仍要求调用方接入自定义 Factory。

## 5. 显式 WrapOp

通用 `ops` 与语言专属 Operation 不能自动互换。Factory 可以提供 `WrapOp`：

```go
factory := php.NewClientFactory()
op, err := factory.WrapOp(ops.NewExec("whoami"))
if err != nil { /* ... */ }
result, err := client.Do(ctx, op)
```

`WrapOp` 不会被 `Client.Do` 隐式调用。上传操作会读取通用 `io.Reader`，因此调用方应明确承担一次性读取和内存占用。

## 6. Shell 与客户端的状态配对

ASP、ASPX、JSP 的混淆器会同时改变服务端字段名和客户端参数名。部署 Shell 后，客户端必须使用同一个 `Obfuscator`：

```go
obf := jsp.NewObfuscator()
shell := jsp.ShellSourceWith(obf) // 部署 shell
op := jsp.NewJspExec("whoami").WithObfuscator(obf)
```

如果只随机化 Shell 而没有保存 `Obfuscator`，后续请求无法匹配字段和动作码。PHP 的模板在每次 Build 时生成随机内部变量和参数字段，Transport 只负责原样提交 `Request.Params`。

JSP 有两种模式：

- `ShellStatic`：预部署全部动作码，兼容所有 JDK，默认推荐；
- `ShellDynamic`：通过 `ScriptEngine` 执行 JavaScript，需要 Nashorn（JDK 6–14），已弃用。

## 7. 错误协议和边界

远端错误应使用不会与合法文件内容混淆的前缀，例如 `ERR:<random>:`，解析时转换为 `core.ErrRemoteRuntime`。Marker Envelope 的开始/结束标记由客户端每次生成，适配器不得假设固定值。

如果适配器支持 Wire Protocol，需要让服务端输出版本、RID、时间戳、nonce、状态和可选 HMAC；客户端使用 `marker.NewWithWire()` 和 `httpform.Options.WireProtocol` 成对启用。

## 8. 测试清单

至少覆盖：

1. 每个 Operation 的 `Name`、Build 参数编码和风险等级；
2. 正常响应、空响应、格式错误响应和远端错误前缀；
3. 二进制文件读取不损坏字节；
4. `Capabilities()` 与实际操作集合一致；
5. 自定义字段名/混淆器在 Shell 和客户端之间一致；
6. Factory 对空记录、空 Endpoint 和未知 Transport 返回明确错误；
7. 与 `transport/mock` 的完整 Client 流程测试。
