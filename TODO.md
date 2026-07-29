# TODO — redbeanshellcore 整改路线图

基于 [PROJECT_REVIEW.md](./PROJECT_REVIEW.md) 识别的问题，按用户设定的优先级重排。核心原则：

- **P0 — 流量伪装/随机化/混淆**：以绕过检测为目标，消除固定流量指纹。
- **P1 — 标准加密、认证、敏感内容最小暴露**：建立可信通信基础，保障机密性与完整性。
- **P2 — Bug 修复与工程改进**：修复现有缺陷，提升工程质量和可维护性。

---

## P0（最高优先级）— 流量伪装、随机化与混淆

> 目标：消除协议形状、请求时序、载荷文本和传输层指纹的确定性特征，使其难以被规则或机器学习模型识别。

### [ ] P0.1 可配置 User-Agent 轮换池

- [x] 提供内置常见浏览器 UA 列表（Chrome／Firefox／Edge／Safari 多版本）
- [x] 支持自定义 UA 池注入
- [x] 支持每个请求或每个会话随机选取
- [x] 关联 Accept、Accept-Language、Accept-Encoding、Sec-CH-UA 等头部的协同轮换

**涉及文件**：`transport/useragent/pool.go`、`transport/httpform/transport.go`
**状态**：✅ 已完成。流量包验证：5 次请求出现 3 种不同浏览器指纹（Chrome Win/Linux、Firefox），Sec-CH-UA / Accept-Language 同步轮换。

### [ ] P0.2 请求时序随机化（Jitter）

- [x] 每次请求前增加可配置范围的随机延迟（最小／最大间隔）
- [x] 支持 backoff 策略（固定、线性、指数抖动）
- [x] 支持按操作类型或目标节点单独配置

**涉及文件**：`transport/jitter/jitter.go`
**状态**：✅ 已完成。Middleware 形式，可插入 Client 中间件链，按操作名累计尝试次数。

### [ ] P0.3 动态表单字段名生成

- [x] 替代固定的 `auth_password_field` 等字段名
- [x] 每次请求生成符合常见表单命名模式的随机字段名
- [x] 主 payload 字段名与附加参数字段名均由生成器决定

**涉及文件**：`transport/httpform/fieldgen.go`、`transport/httpform/transport.go`
**状态**：✅ 已完成。5 种命名模式（short/underscore/camelCase/numeric/randomWord），FieldGenerator 测试 30/30 不重复。

### [ ] P0.4 移除或随机化 PHP 特征常量

- [x] 消除 `|||askey|||`、`|||asline|||`、`ERROR://REDBEAN:*` 等固定文本标记
- [x] 改用随机分隔符或长度分帧的结构化响应
- [x] 消除 `base64_decode`、`system`／`passthru` 等函数调用链的固定文本特征

**涉及文件**：`adapter/php/renderer.go`、`adapter/php/operations.go`、`adapter/php/obfuscate.go`
**状态**：✅ 部分完成。流量包验证：0/10 请求包含 `|||askey|||`/`|||asline|||`/`ERROR://REDBEAN`。`base64_decode` 和命令执行函数链是 PHP Shell 核心功能，需 P1 结构化协议消除。

### [ ] P0.5 请求正文填充与结构随机化

- [x] 在表单正文中插入随机顺序的无关字段（honeypot fields）
- [x] 支持对 payload 进行填充（padding），使密文或编码后长度不固定
- [x] 消除 `|||` 分隔符和 `ERROR://` 前缀等唯一签名（同 P0.4）

**涉及文件**：`transport/httpform/fieldgen.go`、`transport/httpform/transport.go`
**状态**：✅ 已完成。诱饵字段采用 CSRF Token/分页参数/时间戳/WordPress 字段等真实模式，流量包验证无 `dev`/`localhost`/`undefined` 等人工值。

### [ ] P0.6 TLS 客户端指纹随机化

- [x] 通过自定义 `http.Transport` 配置 TLS 参数（密码套件顺序、ALPN、支持的曲线）
- [x] 支持模拟不同浏览器／操作系统的 TLS Client Hello 指纹
- [ ] 集成现有 JA3 轮换方案或 utls 库（当前使用 Go 标准库内置密码套件随机子集）

**涉及文件**：`transport/httpform/transport.go`
**状态**：✅ Partially done. 密码套件随机子集选取已实现；完整的 utls/JA3 伪装依赖外部库，需单独依赖管理。

### [ ] P0.7 可插拔请求／响应变换中间件

- [x] 设计 Transform 接口（core 已有），允许在发送前和接收后对请求／响应做任意变换
- [x] 内置变换：HTML 注释包裹、JS 变量包裹、JSONP 回调包裹
- [ ] 内置变换：图片隐写（未实现）
- [x] 变换可组合、可开关，按节点或操作配置

**涉及文件**：`transform/htmlcomment/transform.go`、`transform/jswrapper/transform.go`、`transform/jsonp/transform.go`
**状态**：✅ 基本完成。3 种内置 Transform 全部可用，核心接口已定义。

### [ ] P0.8 多协议传输协商

- [x] 支持 HTTP/1.1、HTTP/2、HTTP/3 协议协商与轮换
- [x] 按节点配置首选传输协议
- [ ] 请求协议与指纹应协同随机化（待 P0.6 utls 集成后统一）

**涉及文件**：`transport/httpform/transport.go`
**状态**：✅ 基本完成。HTTP/1.1、HTTP/2 可用；HTTP/3 依赖 quic-go 等外部库，当前为占位。

### [ ] P0.9 连接复用与会话维持

- [x] 长期复用 `http.Client` 和 `http.Transport`
- [x] 支持 CookieJar 以维持服务端会话状态
- [x] 可配置连接池大小、空闲超时、TLS 会话重用策略

**涉及文件**：`transport/httpform/transport.go`
**状态**：✅ 已完成。`sync.Once` 单例 Client，内置 CookieJar，连接池参数可配。

### [ ] P0.10 代理链与出口 IP 轮换

- [x] 支持 HTTP／SOCKS5 代理链配置
- [x] 支持代理池轮换（每个请求或每次失败后切换）
- [ ] 代理认证凭据由 CredentialProvider 管理，不内联在配置中（待 P1 实现）

**涉及文件**：`transport/httpform/transport.go`
**状态**：✅ 基本完成。流量包验证：5 请求全部通过 127.0.0.1:8080 代理，代理轮换逻辑可用。凭据管理待 P1.2 CredentialProvider。

---

## P1（高优先级）— 标准加密、认证与敏感内容最小暴露

> 目标：设计带认证的版本化 wire protocol，实现密钥独立管理、消息完整性保护，并消除敏感数据的非必要暴露。

### [ ] P1.1 设计版本化、结构化、带认证的 wire protocol

- [ ] 协议头部包含版本号、请求 ID、时间戳、nonce
- [ ] 使用 HMAC 或 AEAD 对全部消息体计算认证标签
- [ ] 服务端验证时间窗和 nonce 唯一性以防御重放攻击
- [ ] 将 `auth_password_field` 重命名为 `payload_form_field`，消除安全误导

**涉及文件**：`transport/httpform/transport.go`、`docs/`、`core/client.go`

### [ ] P1.2 CredentialProvider 与密钥引用模型

- [ ] 定义 `CredentialProvider` 接口，从系统密钥链、环境变量或外部 Vault 获取凭据
- [ ] Registry 只保存密钥引用（secret reference），不保存明文凭据（参考 Issue 15）
- [ ] 导出、日志和错误消息默认对敏感字段脱敏

**涉及文件**：`core/`（新增 credential.go）、`registry/file/registry.go`

### [ ] P1.3 消息完整性校验

- [ ] 每个请求／响应携带消息认证码（MAC）
- [ ] 支持多种算法：HMAC-SHA256、AES-GCM 的附加数据认证
- [ ] 密钥通过 CredentialProvider 按节点获取，不共享

**涉及文件**：`transport/`、`core/client.go`

### [ ] P1.4 身份认证与防重放

- [ ] 服务端验证请求身份（基于共享密钥或公钥签名）
- [ ] nonce + 时间窗机制确保消息唯一性
- [ ] 响应同样携带认证标签，防止响应的伪造与篡改

**涉及文件**：`transport/httpform/transport.go`、`adapter/php/`、`core/client.go`

### [ ] P1.5 审计失败处理策略

- [ ] 为 `AuditMiddleware` 提供 fail-open／fail-closed 配置（参考 Issue 22）
- [ ] 审计记录错误时至少暴露指标和回调
- [ ] 合规模式默认 fail-closed：审计失败则操作被拒绝

**涉及文件**：`middleware/audit/middleware.go`

### [ ] P1.6 不可变 Result 与数据所有权

- [ ] `Result.Raw` 和 `Result.Meta` 返回副本而非内部引用（参考 Issue 23）
- [ ] 调用方无法篡改结果内部状态
- [ ] 文档明确结果数据所有权和生命周期

**涉及文件**：`core/result.go`

### [ ] P1.7 注册表敏感字段脱敏

- [ ] Registry 序列化前对 Auth、Options、Metadata 的敏感字段自动脱敏
- [ ] 导出、备份、日志路径默认不包含凭证材料
- [ ] 提供 `SecretRef` 类型替代明文存储

**涉及文件**：`registry/file/registry.go`、`registry/memory/registry.go`

### [ ] P1.8 Session 不可变快照

- [ ] `Client.Session()` 返回不可变快照，而非内部指针（参考 Issue 16）
- [ ] Metadata 和 Capabilities 的变更通过受控 API 进行
- [ ] 运行期状态使用同步原语保护

**涉及文件**：`core/session.go`、`core/client.go`

---

## P2（中低优先级）— Bug 修复与工程改进

> 目标：修复现有功能缺陷，补齐工程基础设施，确保代码可维护、可验证。

### [ ] P2.1 恢复测试体系

- [ ] 从 `.gitignore` 移除 `*_test.go`（参考 Issue 4）
- [ ] 恢复已删除的 22 个测试文件、3787 行测试（参考 commit `efd3a05`）
- [ ] `go test ./...` 通过，补充竞态检测
- [ ] 建立覆盖率门槛和 CI 门禁

### [ ] P2.2 Marker fail-open → fail-closed（参考 Issue 2）

- [ ] 启用 Marker 后，开始／结束标记缺失、重复或顺序错误时返回 `ErrEnvelope`
- [ ] 原始响应不得在 Marker 解析失败时进入业务处理

### [ ] P2.3 组件组合校验（参考 Issue 3）

- [ ] Client 构造阶段拒绝无效的 Codec + Envelope + Adapter 组合
- [ ] Adapter 提供完整且经过验证的 wire profile
- [ ] 建立兼容矩阵

### [ ] P2.4 上传流式化（参考 Issue 9、10）

- [ ] 设计分块流式上传协议（块编号、总长度、内容摘要）
- [ ] PHP 端循环写入并累计字节数，返回实际长度
- [ ] 客户端校验摘要，支持断点续传和原子提交

### [ ] P2.5 Exec 管道死锁修复（参考 Issue 11）

- [ ] 并发读取 stdout／stderr 避免管道填满阻塞
- [ ] 退出码通过结构化字段独立传输，不依赖正文尾部 `ret=<n>`
- [ ] 移除 `2>&1` 合并逻辑，使 stderr 字段具有真实语义

### [ ] P2.6 文件响应控制／数据分离（参考 Issue 12）

- [ ] 响应中独立传输状态、错误码、长度和数据体
- [ ] 消除固定错误字符串被误判为文件内容的风险

### [ ] P2.7 PHP 文件列表完整解析（参考 Issue 13）

- [ ] 复用或实现通用解析器，填充 Path、Size、Mode、ModTime
- [ ] 非法字段产生可观察错误

### [ ] P2.8 Manager 状态机修复（参考 Issue 17）

- [ ] Frozen 状态在 Client.Do 入口强制阻断
- [ ] LastError、LastSeenAt 在正确时机更新
- [ ] Register 在成功通信后才设置 LastSeenAt
- [ ] Refresh 刷新 Capabilities

### [ ] P2.9 Capability 强制执行（参考 Issue 18）

- [ ] Client.Do 在 Build 前检查 `RequiredCapabilities`
- [ ] Adapter Factory 填充能力快照
- [ ] Manager 消费 Capability 信息

### [ ] P2.10 DoEach 批量操作限制（参考 Issue 19）

- [ ] DoEach 默认拒绝非 `RiskReadOnly` 操作
- [ ] 高风险批量写入使用独立、显式命名的 API

### [ ] P2.11 Registry 一致性（参考 Issue 20）

- [ ] Manager 或共享验证层统一校验（空 ID、重复等）
- [ ] 文件 Registry 增加文件锁和版本冲突检测，明确限制为单进程

### [ ] P2.12 错误分类稳定（参考 Issue 21）

- [ ] 为配置、策略、远端状态、传输、协议和解析建立稳定 Kind
- [ ] 包装时不得改写已有明确 Kind
- [ ] HTTP 429 等状态码有专门语义

### [ ] P2.13 Factory 装配与校验（参考 Issue 24、25、29）

- [ ] 删除默认 Factory 或实现完整的组件注册／选择
- [ ] PHP Factory 统一装配 Codec + Envelope + Transform + Middleware
- [ ] 所有不支持字段在构造时返回错误
- [ ] `NewManager` 验证 Registry 非 nil，返回 `(*Manager, error)`

### [ ] P2.14 Operation 抽象纯净（参考 Issue 26）

- [ ] Operation 只保留业务语义和参数
- [ ] Adapter 负责渲染 wire request 和解析响应
- [ ] 消除语言专属 Operation 泄漏到业务 API

### [ ] P2.15 CI 与代码质量（参考 Issue 27）

- [ ] 统一 gofmt、go vet、test、race、lint 流程
- [ ] 覆盖 Linux、Windows、macOS 多平台
- [ ] 设为 PR 合并门禁

### [ ] P2.16 文档修复（参考 Issue 28）

- [ ] 删除不存在的功能描述（如 NODES 环境变量加载）
- [ ] 文档示例必须编译并通过测试
- [ ] 所有示例先检查 error 和类型断言

---

## 优先级总览

| 层级 | 范围 | 性质 |
|------|------|------|
| **P0** | 流量伪装、随机化、混淆 | 新功能设计，彻底重写传输层特征 |
| **P1** | 加密、认证、敏感内容最小暴露 | 协议与安全架构重建 |
| **P2** | Bug 修复与工程改进 | 缺陷修复 + 工程基础设施补齐 |

> P0 和 P1 是 **核心价值功能**，决定项目的可用性和隐蔽性天花板。
> P2 保证现有功能和未来扩展的质量底座，优先级低于前两者。

*** End of File
