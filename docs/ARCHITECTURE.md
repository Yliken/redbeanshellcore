# 架构与实现边界

## 包结构

```text
core/                 生命周期、组件接口、Client、Manager、错误和结果
adapter/<language>/   服务端语言模板、参数协议、解析器、Factory
transport/            HTTP form、mock、UA、jitter
codec/                plain、base64
envelope/marker/      tag_s/tag_e 和可选 Wire Protocol
protocol/wire/        Wire Protocol 头、HMAC 和 nonce
middleware/            logging、audit、timeout、retry、readonly
registry/              memory、JSON file
crypto/                noop、AES-GCM
transform/             noop、HTML comment、JSONP、JS wrapper
ops/                   不带语言语义的通用 Operation
```

## 组件职责

`core.Client` 是唯一的单节点执行入口。它不理解 PHP、JSP 或 ASP 的代码格式，只调用 Operation 和各组件接口。适配器负责把参数变成目标语言能消费的 payload；Transport 只负责把 Request 传输到 Endpoint。

`Manager` 是节点注册和批量读取的轻量协调层，不是任务调度器。批量 `DoEach` 仅接受只读操作，最多同时运行 10 个节点，并为每个节点设置 30 秒上下文。

`Registry` 的实现可以替换。内置 memory 适合短生命周期，file 适合简单 CLI；两者都会深复制 NodeRecord。

## 兼容性矩阵

| 适配器 | Transport | Codec | Envelope | Crypto | 工厂 |
| --- | --- | --- | --- | --- | --- |
| PHP | httpform | plain | marker / wire | 需配套 Shell | 支持 |
| ASP | httpform | plain | 需服务端配合 | 需配套 Shell | 支持 |
| ASPX | httpform | plain | 需服务端配合 | 需配套 Shell | 支持 |
| JSP static | httpform | plain | 需服务端配合 | 可使用 JSP CryptoShell | 手动 Client |
| JSP dynamic | httpform | plain | 需服务端配合 | 可使用 CryptoDynamicShell | 手动 Client |

已知 Adapter Profile 会在 Client 执行前拒绝不兼容的 Codec/Envelope 组合。未知适配器不会自动获得能力或协议实现，应由调用方提供完整组件。

## 重要实现约束

1. `core.DefaultClientFactory` 不是开箱即用的 HTTP 工厂。真实节点应使用语言适配器 Factory，或实现自己的 `ClientFactory`。
2. `WrapOp` 是显式转换 API。Client 不会自动把 `ops.NewExec` 转换为 PHP/JSP/ASP/ASPX Operation。
3. Crypto 在 Envelope 之后、Transport 之前运行。加密后的字节必须由服务端 Shell 解密，否则客户端无法完成 RoundTrip。
4. Marker 缺少任一边界时会失败关闭并返回 `ErrEnvelope`。HTTP 200 不能单独证明操作成功。
5. `retry` 默认只覆盖幂等读操作；不要把写入和命令执行加入默认重试。
6. `InsecureTLS`、代理、动态字段和诱饵字段属于传输配置，不能替代认证、授权或服务端访问控制。

## 维护建议

新增适配器时先补齐：Shell 生成示例、Factory、能力声明、六类操作的 Build/Parse 测试，再把兼容性加入本文档。修改协议字段或默认值时，应同步更新 Quickstart、User Manual、示例和 Wire/Envelope 测试。
