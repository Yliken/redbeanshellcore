# redbeanshellcore

> 远程节点通信与操作抽象中间件 / SDK Core（Go）

[![Go](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

*"红豆生南国，春来发几枝。"*

---

## 简介

`redbeanshellcore` 是一个 Go 语言实现的远程节点操作 SDK Core。它把底层远程节点交互中的通用逻辑沉淀为稳定、可复用、可扩展的框架。

核心能力：

- **模板生成**：内置 AntSword PHP 模板移植，payload 自动生成
- **函数名混淆**：strrev/substr/chr 等多策略，消除流量中函数名特征
- **Wire Protocol**：版本化结构化协议（RBS1.0），HMAC-SHA256 签名与完整性校验
- **请求封装**：base64 编码内联、参数归一化
- **编解码**：plain / base64 codec，可扩展
- **传输层**：HTTP form POST，可扩展自定义 Transport
- **边界协议**：tag_s / tag_e 响应截取
- **结果解析**：强类型 Result，不让上层解析字符串
- **错误归一化**：12 种 ErrorKind，上层按类型判断
- **会话管理**：Session + Client + Manager + Registry
- **多节点支持**：注册、索引、分组、批量操作
- **中间件链**：logging / audit / readonly / timeout / retry

---

## 快速开始

```bash
go get github.com/Yliken/redbeanshellcore
```

```go
client := core.NewClient(
    core.WithSession(&core.Session{
        NodeID: "lab-01",
        Endpoint: "https://lab.example/shell.php",
        Adapter: "php",
        Metadata: map[string]string{"payload_form_field": "antpwd"},
    }),
    core.WithTransport(httpform.New("https://lab.example/shell.php")),
)

res, _ := client.Do(ctx, phpshell.NewPhpInfo())
info := res.(*core.InfoResult)
fmt.Printf("workdir=%s\nos=%s\nuser=%s\n", info.Workdir, info.OS, info.User)
```

详见 [QUICKSTART.md](docs/QUICKSTART.md)。

---

## 文档

| 文档 | 说明 |
|------|------|
| [QUICKSTART.md](docs/QUICKSTART.md) | 快速开始、常见问题 |
| [USER_MANUAL.md](docs/USER_MANUAL.md) | 完整 API 参考 |
| [ADAPTER.md](docs/ADAPTER.md) | 编写自定义适配器 |

---

## TODO

目前仅支持 **PHP** 语言的 WebShell，计划添加对其他语言的支持：

- [x] **JSP** — Java Web Shell
- [ ] **ASP / ASPX** — IIS 环境
- [ ] 其他语言

欢迎贡献代码！

---

## 安全提示

⚠️ 本 SDK 仅用于已获得书面授权的靶场 / 自有环境。请勿对未经授权的目标使用。请遵守《网络安全法》等法律法规。

---

## License

本项目基于 [MIT License](LICENSE) 开源。

您可以自由使用、修改、分发本软件，但需遵守以下义务：
- 保留原始版权声明和许可声明
- 不对软件提供任何担保

⚠️ **免责声明**：本工具仅用于已获得授权的安全测试。使用者需自行承担使用后果，作者不对任何滥用行为负责。

---





## 致谢

感谢 **中国蚁剑（AntSword）** 项目。


项目地址：https://github.com/AntSwordProject/antSword
