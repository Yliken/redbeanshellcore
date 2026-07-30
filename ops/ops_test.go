package ops

import (
	"context"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestInfoOperation_Build(t *testing.T) {
	op := NewInfo()
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "info" {
		t.Fatalf("Operation 应为 info，got %q", req.Operation)
	}
	if string(req.Payload) != "info" {
		t.Fatalf("Payload 应为字面量 info，got %q", req.Payload)
	}
}

func TestInfoOperation_Parse(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("/var/www\t/\tLinux 5.0\twww-data")
	res, err := NewInfo().Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	info, ok := res.(*core.InfoResult)
	if !ok {
		t.Fatalf("期望 *core.InfoResult，got %T", res)
	}
	if info.Workdir != "/var/www" || info.OS != "Linux 5.0" || info.User != "www-data" {
		t.Fatalf("解析结果不符: workdir=%q os=%q user=%q", info.Workdir, info.OS, info.User)
	}
}

func TestInfoOperation_Parse_Short(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("Linux")
	info, _ := NewInfo().Parse(context.Background(), resp)
	if info.(*core.InfoResult).OS != "Linux" {
		t.Fatalf("短响应应只填 OS")
	}
}

func TestInfoOperation_Parse_Nil(t *testing.T) {
	_, err := NewInfo().Parse(context.Background(), nil)
	if err == nil {
		t.Fatal("Parse(nil) 应返回错误")
	}
}

func TestExecOperation_Build(t *testing.T) {
	op := NewExecWithBin("whoami", "/bin/bash")
	op.Env = map[string]string{"FOO": "bar", "PATH": "/usr/bin"}
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "exec" {
		t.Fatalf("Operation 应为 exec，got %q", req.Operation)
	}
	if string(req.Payload) != "whoami" {
		t.Fatalf("Payload 应为命令字面量，got %q", req.Payload)
	}
	bin, _ := req.GetParam("bin")
	if string(bin) != "/bin/bash" {
		t.Fatalf("bin 参数不符: %q", bin)
	}
	env, _ := req.GetParam("env")
	if !strings.Contains(string(env), "FOO=bar") || !strings.Contains(string(env), "|||asline|||") {
		t.Fatalf("env 参数格式不符: %q", env)
	}
}

func TestExecOperation_Parse(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("line1\nline2\nSTDERR://some error\nret=42")
	res, err := NewExec("x").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 42 {
		t.Fatalf("ExitCode 应为 42，got %d", exec.ExitCode)
	}
	if exec.Stdout != "line1\nline2" {
		t.Fatalf("Stdout 不符: %q", exec.Stdout)
	}
	if exec.Stderr != "some error" {
		t.Fatalf("Stderr 不符: %q", exec.Stderr)
	}
}

func TestExecOperation_Parse_CleanStdout(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("clean output")
	res, err := NewExec("x").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 0 {
		t.Fatalf("无 ret= 时 ExitCode 应为 0，got %d", exec.ExitCode)
	}
	if exec.Stdout != "clean output" {
		t.Fatalf("Stdout 不符: %q", exec.Stdout)
	}
}

func TestFileListOperation_Build(t *testing.T) {
	req, _ := NewFileList("/var/www").Build(context.Background(), nil)
	if req.Operation != "file.list" {
		t.Fatalf("Operation 不符")
	}
	path, _ := req.GetParam("path")
	if string(path) != "/var/www" {
		t.Fatalf("path 参数不符: %q", path)
	}
}

func TestFileListOperation_Parse(t *testing.T) {
	resp := core.NewResponse()
	// name \t modtime \t size \t mode
	resp.Body = []byte("file.txt\t2024-01-01 12:00:00\t1024\t0644\nsubdir/\t2024-01-01 12:00:00\t0\t0755\n")
	res, err := NewFileList("/tmp").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	list := res.(*core.FileListResult)
	if list.Path != "/tmp" {
		t.Fatalf("Path 不符: %q", list.Path)
	}
	if len(list.Entries) != 2 {
		t.Fatalf("应返回 2 个条目，got %d", len(list.Entries))
	}
	if list.Entries[0].Name != "file.txt" || list.Entries[0].Size != 1024 {
		t.Fatalf("第一个条目不符: %+v", list.Entries[0])
	}
	if !list.Entries[1].IsDir || list.Entries[1].Name != "subdir" {
		t.Fatalf("第二个条目应为目录: %+v", list.Entries[1])
	}
}

func TestFileListOperation_Parse_MalformedLines(t *testing.T) {
	resp := core.NewResponse()
	// 格式错误的行进 unparsed
	resp.Body = []byte("good\t2024-01-01 12:00:00\t0\t0644\nbad-line\n")
	res, err := NewFileList("/").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	fileListResult := res.(*core.FileListResult)
	if len(fileListResult.Entries) != 1 {
		t.Fatalf("应只有 1 个有效条目，got %d", len(fileListResult.Entries))
	}
	if fileListResult.Meta()["unparsed"] != "bad-line" {
		t.Fatalf("未解析行应记入 Meta[unparsed]: %q", fileListResult.Meta()["unparsed"])
	}
}

func TestFileReadOperation_BuildAndParse(t *testing.T) {
	content := []byte("binary \x00\xff data")
	req, _ := NewFileRead("/etc/passwd").Build(context.Background(), nil)
	path, _ := req.GetParam("path")
	if string(path) != "/etc/passwd" {
		t.Fatalf("path 不符: %q", path)
	}

	resp := core.NewResponse()
	resp.Body = content
	res, err := NewFileRead("/etc/passwd").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	fr := res.(*core.FileReadResult)
	if string(fr.Data) != string(content) {
		t.Fatalf("Data 不符合预期")
	}
	if fr.Path != "/etc/passwd" {
		t.Fatalf("Path 不符: %q", fr.Path)
	}
}

func TestFileUploadOperation_Build(t *testing.T) {
	content := []byte("upload data here")
	op := NewFileUpload("/tmp/dest.txt", strings.NewReader(string(content)))
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "file.upload" {
		t.Fatalf("Operation 不符")
	}
	path, _ := req.GetParam("path")
	if string(path) != "/tmp/dest.txt" {
		t.Fatalf("path 不符: %q", path)
	}
	gotContent, _ := req.GetParam("content")
	if string(gotContent) != string(content) {
		t.Fatalf("content 不符: %q", gotContent)
	}
	if req.Meta["chunk_size"] != "full" {
		t.Fatalf("chunk_size 应为 full: %q", req.Meta["chunk_size"])
	}
}

func TestFileUploadOperation_NilReader(t *testing.T) {
	op := NewFileUpload("/x", nil)
	_, err := op.Build(context.Background(), nil)
	if err == nil {
		t.Fatal("nil reader 应返回错误")
	}
}

func TestFileUploadOperation_DefaultChunkSize(t *testing.T) {
	op := NewFileUpload("/x", strings.NewReader("a"))
	if op.ChunkSize != 64*1024 {
		t.Fatalf("默认 ChunkSize 应为 64KiB，got %d", op.ChunkSize)
	}
}

func TestFileDownloadOperation_Build(t *testing.T) {
	req, _ := NewFileDownload("/etc/passwd").Build(context.Background(), nil)
	if req.Operation != "file.download" {
		t.Fatalf("Operation 不符")
	}
	path, _ := req.GetParam("path")
	if string(path) != "/etc/passwd" {
		t.Fatalf("path 不符: %q", path)
	}
}

func TestFileDownloadOperation_Parse(t *testing.T) {
	content := []byte("file content")
	resp := core.NewResponse()
	resp.Body = content
	res, err := NewFileDownload("/x").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	fr := res.(*core.FileReadResult)
	if string(fr.Data) != string(content) {
		t.Fatalf("Data 不符")
	}
}

// 防止 import 误删
var _ = context.Background
