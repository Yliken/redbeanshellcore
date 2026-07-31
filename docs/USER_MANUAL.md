# 完整使用手册

本文档以当前源码为准，描述 `core`、内置传输、编解码、边界协议、中间件、注册表和适配器之间的契约。

## 1. 分层和请求生命周期

```text
Operation.Build
  -> Codec.EncodeRequest                (可选)
  -> Envelope.Wrap                      (可选)
  -> Transform.ApplyRequest             (可选)
  -> Middleware 链
       -> Crypto.Encrypt                (可选)
       -> Transport.RoundTrip
       -> Crypto.Decrypt                 (可选)
       -> HTTP 状态码检查
       -> Envelope.Extract               (可选)
       -> Transform.ApplyResponse       (逆序，可选)
       -> Codec.DecodeResponse           (可选)
       -> Operation.Parse
```

`Middleware` 包裹完整的响应处理流程，因此 logging、audit、retry 可以观察 Transport、Envelope、Decode 和 Parse 的最终结果。`Operation.Build` 和请求编码发生在 Middleware 之前。

## 2. Client 和 Session

```go
sess := core.NewSession("node-1", endpoint)
sess.Adapter = "php"
sess.Metadata["payload_form_field"] = "antpwd"

client := core.NewClient(
    core.WithSession(sess),
    core.WithTransport(httpform.New(endpoint)),
    core.WithMiddleware(logging.Middleware()),
)
result, err := client.Do(ctx, php.NewPhpInfo())
```

`Client` 至少需要 `Transport`。`Session` 只保存单节点运行上下文，不负责长期持久化；常用字段如下：

| 字段 | 用途 |
| --- | --- |
| `NodeID` | 日志、错误和请求关联 ID |
| `Endpoint` | 节点入口地址 |
| `Adapter` | `php`、`jsp`、`asp`、`aspx` 等适配器名称 |
| `Metadata` | 传输字段名、HMAC key、适配器参数等 |
| `Capabilities` | 节点能力快照，供上层查询 |

`Client.Do` 会为请求补充 `ID`、`operation`、`node_id`、UTC 时间戳和风险等级元数据。Operation 返回 `nil`、Transport 未配置或组件返回 `nil` 都会转换为 `OpError`。

## 3. Operation、风险和结果

自定义操作只需实现三个方法：

```go
type MyOperation struct{}
func (o *MyOperation) Name() string { return "custom.read" }
func (o *MyOperation) Build(ctx context.Context, s *core.Session) (*core.Request, error) { /* ... */ }
func (o *MyOperation) Parse(ctx context.Context, r *core.Response) (core.Result, error) { /* ... */ }
```

需要能力或策略控制时可额外实现：

```go
func (o *MyOperation) RequiredCapabilities() []core.Capability { return []core.Capability{core.CapFileRead} }
func (o *MyOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }
```

内置风险等级为 `read_only`、`write`、`exec`、`destructive`。`readonly.Middleware` 会拦截 `exec`、`file.upload` 以及写入/破坏性风险。

标准结果类型：

| 类型 | 主要字段 |
| --- | --- |
| `InfoResult` | `OS`、`User`、`Workdir` |
| `ExecResult` | `Stdout`、`Stderr`、`ExitCode` |
| `FileListResult` | `Path`、`Entries` |
| `FileReadResult` | `Path`、二进制 `Data` |
| `BoolResult` | `OK`、`Message` |

所有结果都实现 `OperationName()`、`Raw()` 和 `Meta()`。

## 4. Request、Response 和 Transport

`Request.Payload` 是主 payload，`Request.Params` 是额外的字节字段，`Headers` 是传输头，`Meta` 是组件间元数据。Transport 不解析业务，也不负责 Base64；适配器在 `Build` 阶段完成参数编码。

内置 `httpform.Transport` 使用 `application/x-www-form-urlencoded` POST：

```go
opts := httpform.DefaultOptions()
opts.Timeout = 30 * time.Second
opts.InsecureTLS = false
opts.EnableCookieJar = true
tr := httpform.NewWithOptions(endpoint, opts)
tr.ExtraHeaders = map[string]string{"X-Test": "lab"}
```

主 payload 字段解析顺序是 `Request.Meta["payload_form_field"]`、兼容字段 `auth_password_field`、动态字段名，最后才是默认值 `antpwd`。响应 body 上限为 64 MiB，超限返回 `ErrProtocol`。

### HTTP Transport 选项

`httpform.Options` 还支持：

- `UARotation` / `UAPool`：User-Agent 轮换；
- `DynamicFieldNames` / `FieldGen`：请求字段名生成；
- `EnablePadding` / `HoneypotCount`：增加诱饵表单字段，数量最多 20；
- `TLSFingerprint`：TLS 版本、曲线和 cipher suite 配置；
- `Protocol`：自动、HTTP/1.1、HTTP/2 或 HTTP/3（HTTP/3 需要底层运行环境支持）；
- 连接池、Cookie Jar、代理链和代理轮换。

这些选项只改变 HTTP 传输行为，不会改变 Operation 的参数语义。启用 `InsecureTLS` 会关闭证书校验，只应在隔离测试环境使用。

`transport/mock` 提供 `New`、`EchoHandler`、`StaticHandler`、`DispatchHandler` 和 `FailAlways`，适合单元测试，不会发送网络请求。

## 5. Codec、Envelope、Transform 和 Crypto

### Codec

内置 `codec/plain` 不做变换，`codec/base64.New()` 对 Request payload 和 Response body 做 Base64 编解码。适配器必须明确支持对应 Codec；PHP、ASP、ASPX、JSP 的已知 profile 默认不声明额外 Codec。

### Marker Envelope

`envelope/marker` 为每次请求生成随机 `tag_s` / `tag_e`，并在响应中截取两者之间的内容：

```go
core.WithEnvelope(marker.New())
core.WithEnvelope(marker.NewWithLength(32))
```

缺少开始标记、结束标记或标记元数据会返回 `ErrEnvelope`，不会把未识别的错误页当成成功结果。成功提取后 `Response.EnvelopeOK` 为 `true`。

Marker 对 PHP payload 有专门的 `echo` 包装；非 PHP 适配器只会把标记拼到 payload 外层，远端必须自行实现对应协议。

### Transform

可用实现包括 `noop`、`htmlcomment`、`jsonp` 和 `jswrapper`。响应 Transform 在 Envelope 提取之后、Codec 解码之前按逆序执行；因此响应 Transform 不应假设自己能看到完整的 marker 包装。Transform 是扩展接口，不承担业务解析或策略判断。

### Crypto

`core.WithCrypto` 把加密放在 Transport 前、解密放在响应收到后。`crypto/aesgcm` 使用 AES-GCM，线格式为 `base64(nonce || ciphertext || tag)`；`crypto/noop` 仅用于测试。

Crypto 必须与远端 Shell 实现保持同一协议。现有 PHP/ASP/ASPX 普通 Shell 不会自动解密 AES-GCM；JSP 提供 `CryptoShellSource` / `CryptoDynamicShellSource` 作为配套 Shell 生成器。不要只在客户端启用 Crypto 而不部署兼容的服务端。

## 6. Wire Protocol

`httpform.Options.WireProtocol` 为表单增加 `_v`、`_rid`、`_ts`、`_nonce`、`_sig` 字段。`marker.NewWithWire()` 解析响应中的结构化头：

```text
<tag_s>
RBS1.0
RID=<request id>
TS=<unix milliseconds>
NONCE=<nonce>
STATUS=<status>
BODY
<body>
SIG=<optional HMAC-SHA256>
<tag_e>
```

HMAC key 由 `Request.Meta["hmac_key"]` 或 Session Metadata 提供。空 key 表示不签名；配置 key 后客户端会验证响应 body 的 HMAC。Wire Protocol 还会附带 nonce 和时间戳，服务端应自行实施时间窗口和重放检查。

通过适配器工厂配置时使用：

```go
Options: map[string]string{
    "wire_protocol": "true",
    "hmac_key": "change-me",
}
```

`protocol/wire.ResponsePrefix` 默认是 `RBS1.0`，可以在部署协议一致的前提下改为自定义前缀。

## 7. Middleware

```go
core.WithMiddleware(
    logging.Middleware(),
    audit.Middleware(audit.WithSink(sink)),
    timeout.Middleware(timeout.Options{Timeout: 15 * time.Second}),
    retry.Middleware(retry.Options{MaxAttempts: 3}),
    readonly.Middleware(),
)
```

- `logging` 使用 `slog`，默认不记录完整 payload；可用 `WithPayloadLogging` 显式开启截断日志。
- `audit` 记录 request ID、节点、操作、耗时、成功状态和错误类别；`FailClosed` 可让 Sink 写入失败阻断请求。
- `timeout` 为完整处理链设置 deadline，并将网络 deadline 归类为 `ErrTimeout`。
- `retry` 默认只重试幂等读操作（`info`、`file.list`、`file.read`、`file.download`）的网络错误和超时，带指数退避。
- `readonly` 在 Transport 之前拒绝写入和执行操作。

## 8. 错误分类

统一错误类型是 `*core.OpError`，分类包括：`ErrNetwork`、`ErrTimeout`、`ErrAuth`、`ErrProtocol`、`ErrEnvelope`、`ErrEncode`、`ErrDecode`、`ErrParse`、`ErrPermission`、`ErrNotFound`、`ErrRemoteRuntime`、`ErrPolicyDenied` 和 `ErrCrypto`。

```go
var opErr *core.OpError
if errors.As(err, &opErr) {
    fmt.Println(opErr.Kind, opErr.Operation, opErr.NodeID, opErr.Message)
}
```

HTTP 401/403/404/5xx 会分别映射到认证、权限、资源不存在和远端运行时错误。HTTP 200 不代表业务成功：Envelope、Codec 和 Operation.Parse 仍可能失败。

## 9. Manager 和 Registry

`Manager` 负责节点注册、更新、筛选、Client 创建、Ping、Refresh 和只读批量操作：

```go
mgr, _ := core.NewManager(registry, php.NewClientFactory())
_ = mgr.Register(ctx, core.NodeConfig{
    ID: "node-1", Endpoint: endpoint, Adapter: "php", Transport: "httpform",
    Tags: []string{"lab"}, Group: "case-001",
})
nodes, _ := mgr.List(ctx, core.NodeFilter{Group: "case-001", Tags: []string{"lab"}})
record, _ := mgr.Refresh(ctx, "node-1", php.NewPhpInfo())
```

`NodeFilter.IDs` 是 OR，`Tags` 是 AND；空过滤器返回全部记录。节点状态包括 `unknown`、`ready`、`down`、`error`、`frozen`。冻结节点不能创建 Client。`DoEach` 最多并发 10 个节点，并固定 30 秒的单节点超时，只接受 `RiskReadOnly` 操作。

内置注册表：

- `registry/memory.New()`：线程安全的进程内存储；
- `registry/file.New(path)`：JSON 文件持久化，写入采用临时文件 + rename，文件权限为 0600；
- `registry/file.NewInMemory()`：复用文件注册表代码路径但不落盘。

注册表会深复制记录，调用方修改返回值不会直接改变内部状态。

## 10. 适配器兼容性

| 适配器 | 工厂创建 Client | 默认操作模式 | Shell 生成 |
| --- | --- | --- | --- |
| PHP | 支持 | 每次请求生成 PHP 代码 | `php.NewPHPTemplates` / `adapter/php` |
| ASP | 支持 | VBScript 模板 + Base64 参数 | `asp.ShellSource` |
| ASPX | 支持 | C# CodeDom 模板 + Base64 参数 | `aspx.ShellSource` |
| JSP | 不支持，需手动组装 | 静态动作码 | `jsp.ShellSource` |
| JSP Dynamic | 不支持，需手动组装 | Nashorn JavaScript | `jsp.DynamicShellSource`，已弃用 |

`WrapOp` 是工厂提供的显式转换方法，不会被 `Client.Do` 自动调用。通用 `ops` 适合抽象和测试；真实语言 Shell 应使用对应适配器的 `NewPhp*`、`NewAsp*`、`NewAspx*` 或 `NewJsp*`。
