package php

import (
	"context"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestWrapOp_Info(t *testing.T) {
	f := NewClientFactory()
	out, err := f.WrapOp(NewPhpInfo())
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if _, ok := out.(*phpInfo); !ok {
		t.Fatalf("info 应被替换为 *phpInfo，got %T", out)
	}
}

func TestWrapOp_FileList(t *testing.T) {
	f := NewClientFactory()
	out, err := f.WrapOp(NewPhpFileList("/tmp"))
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if _, ok := out.(*phpFileList); !ok {
		t.Fatalf("file.list 应被替换为 *phpFileList，got %T", out)
	}
}

func TestWrapOp_FileRead(t *testing.T) {
	f := NewClientFactory()
	out, err := f.WrapOp(NewPhpFileRead("/etc/passwd"))
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if _, ok := out.(*phpFileRead); !ok {
		t.Fatalf("file.read 应被替换为 *phpFileRead，got %T", out)
	}
}

func TestWrapOp_Exec(t *testing.T) {
	f := NewClientFactory()
	out, err := f.WrapOp(NewPhpExec("ls"))
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if _, ok := out.(*phpExec); !ok {
		t.Fatalf("exec 应被替换为 *phpExec，got %T", out)
	}
}

func TestWrapOp_UnknownOp_Passthrough(t *testing.T) {
	f := NewClientFactory()
	// 一个不在 WrapOp switch 里的操作应原样返回
	stub := &passthroughOp{name: "custom.unknown"}
	out, err := f.WrapOp(stub)
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if out != stub {
		t.Fatal("未知操作应原样返回")
	}
}

func TestWrapOp_ApplyTemplateDisabled(t *testing.T) {
	f := &ClientFactory{ApplyTemplate: false}
	stub := &passthroughOp{name: "info"}
	out, err := f.WrapOp(stub)
	if err != nil {
		t.Fatalf("WrapOp 出错: %v", err)
	}
	if out != stub {
		t.Fatal("ApplyTemplate=false 时应原样返回")
	}
}

func TestBuildTransport_HttpForm(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{
		Config: core.NodeConfig{ID: "n1", Endpoint: "http://example.com/shell.php", Transport: "httpform"},
	}
	tr, err := f.buildTransport(rec)
	if err != nil {
		t.Fatalf("buildTransport 出错: %v", err)
	}
	if tr == nil {
		t.Fatal("httpform transport 不应为 nil")
	}
}

func TestBuildTransport_Default(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{
		Config: core.NodeConfig{ID: "n1", Endpoint: "http://example.com/shell.php", Transport: ""},
	}
	tr, err := f.buildTransport(rec)
	if err != nil {
		t.Fatalf("buildTransport 出错: %v", err)
	}
	if tr == nil {
		t.Fatal("空 transport 应默认走 httpform")
	}
}

func TestBuildTransport_InsecureTLS(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{
		Config: core.NodeConfig{
			ID:        "n1",
			Endpoint:  "https://example.com/shell.php",
			Transport: "httpform",
			Options:   map[string]string{"insecure_tls": "true"},
		},
	}
	tr, err := f.buildTransport(rec)
	if err != nil {
		t.Fatalf("buildTransport 出错: %v", err)
	}
	if tr == nil {
		t.Fatal("transport 不应为 nil")
	}
}

func TestBuildTransport_Unsupported(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{
		Config: core.NodeConfig{ID: "n1", Transport: "grpc"},
	}
	_, err := f.buildTransport(rec)
	if err == nil {
		t.Fatal("不支持的 transport 应返回错误")
	}
}

func TestBuildCodec_Plain(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{Config: core.NodeConfig{Codec: "plain"}}
	codec, err := f.buildCodec(rec)
	if err != nil {
		t.Fatalf("buildCodec 出错: %v", err)
	}
	if codec != nil {
		t.Fatal("plain codec 应返回 nil（no-op）")
	}
}

func TestBuildCodec_Unsupported(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{Config: core.NodeConfig{Codec: "xor"}}
	_, err := f.buildCodec(rec)
	if err == nil {
		t.Fatal("不支持的 codec 应返回错误")
	}
}

func TestNewClient_NoEndpoint(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{Config: core.NodeConfig{ID: "n1"}}
	_, err := f.NewClient(context.Background(), rec)
	if err == nil {
		t.Fatal("没有 endpoint 应返回错误")
	}
}

func TestNewClient_Success(t *testing.T) {
	f := NewClientFactory()
	rec := &core.NodeRecord{
		Config: core.NodeConfig{
			ID:        "n1",
			Endpoint:  "http://example.com/shell.php",
			Transport: "httpform",
			Codec:     "plain",
			Adapter:   "php",
			Auth:      map[string]string{"auth_password_field": "antpwd"},
		},
		Metadata: map[string]string{"existing": "meta"},
	}
	c, err := f.NewClient(context.Background(), rec)
	if err != nil {
		t.Fatalf("NewClient 出错: %v", err)
	}
	if c == nil {
		t.Fatal("Client 不应为 nil")
	}
	// session 应被挂上
	if c.Session() == nil {
		t.Fatal("Session 不应为 nil")
	}
	if c.Session().NodeID != "n1" {
		t.Fatalf("Session.NodeID 应为 n1，got %q", c.Session().NodeID)
	}
	// Auth 应被合并到 Session.Metadata
	if c.Session().Metadata["auth_password_field"] != "antpwd" {
		t.Fatalf("Auth 未合并到 Session.Metadata: %+v", c.Session().Metadata)
	}
	// 原有 Metadata 不应丢失
	if c.Session().Metadata["existing"] != "meta" {
		t.Fatalf("原有 Metadata 丢失: %+v", c.Session().Metadata)
	}
}

// passthroughOp 是测试用的简单 Operation 实现。
type passthroughOp struct {
	name string
}

func (p *passthroughOp) Name() string { return p.name }
func (p *passthroughOp) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	return core.NewRequest(p.name), nil
}
func (p *passthroughOp) Parse(_ context.Context, _ *core.Response) (core.Result, error) {
	return &core.BoolResult{BaseResult: core.NewBaseResult(p.name, nil), OK: true}, nil
}
