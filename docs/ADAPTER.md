
# 如何编写自定义适配器

## 目录

- [1. 适配器结构](#1-适配器结构)
- [2. 最小适配器示例](#2-最小适配器示例)
- [3. 专属 Operation](#3-专属-operation)
- [4. ClientFactory](#4-clientfactory)
- [5. 注册到 Manager](#5-注册到-manager)

---

## 1. 适配器结构

```
adapter/
└── jsp/                          # JSP 适配器
    ├── templates.go              # Java 模板 + ShellSource
    ├── js_templates.go          # JS 模板（Dynamic 模式）
    ├── operations.go             # 6 个操作实现
    ├── mode.go                  # ShellStatic / ShellDynamic
    ├── obfuscate.go             # Obfuscator
    ├── crypto_shell.go           # Crypto Shell 生成
    ├── shell_dynamic.go         # Dynamic Shell 源码
    ├── parser.go                # 响应解析
    ├── client_factory.go        # ClientFactory
    ├── capabilities.go          # 能力声明
    └── adapter.go               # 主适配器
```

---

## 2. 最小适配器示例

```go
package jsp

import "github.com/Yliken/redbeanshellcore/core"

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Capabilities() []core.Capability {
    return []core.Capability{
        core.CapInfo,
        core.CapExec,
        core.CapFileList,
        core.CapFileRead,
    }
}
```

---

## 3. 专属 Operation

核心：Build 阶段生成可执行代码，Parse 阶段结构化结果。

```go
// 模板
func renderJspInfo() string {
    return `<%
    String workdir = new java.io.File(".").getAbsolutePath();
    String os = System.getProperty("os.name") + " " + System.getProperty("os.version");
    String user = System.getProperty("user.name");
    out.print(workdir + "\t/\t" + os + "\t" + user);
    %>`
}

// Operation
type jspInfo struct{}

func NewJspInfo() *jspInfo { return &jspInfo{} }
func (o *jspInfo) Name() string { return "info" }

func (o *jspInfo) Build(ctx context.Context, sess *core.Session) (*core.Request, error) {
    req := core.NewRequest(o.Name())
    req.Payload = []byte(renderJspInfo())
    return req, nil
}

func (o *jspInfo) Parse(ctx context.Context, resp *core.Response) (core.Result, error) {
    parts := strings.Split(string(resp.Body), "\t")
    res := &core.InfoResult{BaseResult: core.NewBaseResult(o.Name(), resp.Body)}
    if len(parts) >= 4 {
        res.Workdir = parts[0]
        res.OS = parts[2]
        res.User = parts[3]
    }
    return res, nil
}
```

---

## 4. ClientFactory

```go
type ClientFactory struct{}

func NewClientFactory() *ClientFactory { return &ClientFactory{} }

func (f *ClientFactory) NewClient(ctx context.Context, rec *core.NodeRecord) (*core.Client, error) {
    tr := httpform.New(rec.Config.Endpoint)
    tr.Timeout = 30 * time.Second

    sess := &core.Session{
        NodeID:    rec.Config.ID,
        Endpoint:  rec.Config.Endpoint,
        Adapter:   rec.Config.Adapter,
        Metadata:  rec.Metadata,
    }

    return core.NewClient(
        core.WithSession(sess),
        core.WithTransport(tr),
    ), nil
}
```

---

## 5. 注册到 Manager

方法一：自定义 Factory（推荐）：

```go
// 使用 PHP 适配器内置的 Factory
mgr := core.NewManager(registry, php.NewClientFactory())

// 或接入自己的 JSP / ASP Factory
mgr := core.NewManager(registry, jsp.NewClientFactory())
```

方法二：默认 Factory（仅限跑测试 / 接自定义 selector）：

```go
// ⚠️ 默认工厂对所有内置 transport/codec/envelope 都返回错误或 nil，
// 无法直接用于真实环境。必须提供自定义 ClientFactory。
mgr := core.NewManager(registry, core.DefaultClientFactory())
```

使用：

```go
client, _ := mgr.Client(ctx, "jsp-node")
res, _ := client.Do(ctx, jsp.NewJspInfo())
```

---

## 参考：PHP 适配器实现

编写 JSP / ASP 适配器前，先读 `adapter/php/` 作为完整参考：

- `renderer.go` — 生成可执行的 PHP 源码模板
- `operations.go` — Build 阶段按模板协议编码参数，Parse 阶段生成结构化结果
- `client_factory.go` — `WrapOp` 可显式、按具体类型转换部分通用 Operation，不会被 Client 自动调用
- `capabilities.go` — 声明适配器支持的 Capability；当前 Capability 仅用于描述

PHP Exec/Upload 使用 Base64 参数内联；FileList/FileRead/FileDownload 使用随机 `$_POST` 字段，并由 Operation 预先 Base64 编码字段值。Transport 只负责原样提交 `Request.Params`。

---

## 注意事项

1. **二进制安全**：文件读取/下载使用二进制模式，不要进行字符串转码
2. **协议错误**：为远端错误定义稳定且带命名空间的标记，避免与合法文件内容混淆
3. **参数编码**：编码责任属于 Adapter/Operation，不应由通用 Transport 猜测
4. **显式适配**：通用 Operation 到语言专属 Operation 的转换应保持参数语义并显式调用
