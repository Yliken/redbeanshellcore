package php

import (
	"context"
	stdbase64 "encoding/base64"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)


const (
	remoteErrorPathUnavailable = "ERR:TEST:PATH_UNAVAILABLE"
	remoteErrorFileOpen       = "ERR:TEST:FILE_OPEN_FAILED"
	remoteErrorFileRead       = "ERR:TEST:FILE_READ_FAILED"
)

func TestParseInfo(t *testing.T) {
	// 模拟 PHP 输出：workdir \t drives \t os \t user
	body := "/var/www/html\t/\tLinux 5.4.0\twww-data"
	resp := core.NewResponse()
	resp.Body = []byte(body)

	op := NewPhpInfo()
	res, err := op.Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	info, ok := res.(*core.InfoResult)
	if !ok {
		t.Fatalf("期望 *core.InfoResult，got %T", res)
	}
	if info.Workdir != "/var/www/html" {
		t.Fatalf("Workdir 不符合预期: %q", info.Workdir)
	}
	if info.OS != "Linux 5.4.0" {
		t.Fatalf("OS 不符合预期: %q", info.OS)
	}
	if info.User != "www-data" {
		t.Fatalf("User 不符合预期: %q", info.User)
	}
}

func TestParseInfo_ShortParts(t *testing.T) {
	// 只有 1 个字段时退化为 OS
	body := "Linux"
	resp := core.NewResponse()
	resp.Body = []byte(body)

	res, err := NewPhpInfo().Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	info := res.(*core.InfoResult)
	if info.OS != "Linux" {
		t.Fatalf("OS 不符合预期: %q", info.OS)
	}
}

func TestParseInfo_NilResponse(t *testing.T) {
	_, err := NewPhpInfo().Parse(context.Background(), nil)
	if err == nil {
		t.Fatal("Parse(nil) 应返回错误")
	}
}

func TestParseFileList(t *testing.T) {
	// 每行：name \t modtime \t size \t mode
	body := "file1.txt\t2024-01-01 12:00:00\t1024\t0644\n" +
		"subdir/\t2024-01-01 12:00:00\t0\t0755\n"
	resp := core.NewResponse()
	resp.Body = []byte(body)

	op := NewPhpFileList("/tmp")
	res, err := op.Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	list, ok := res.(*core.FileListResult)
	if !ok {
		t.Fatalf("期望 *core.FileListResult，got %T", res)
	}
	if list.Path != "/tmp" {
		t.Fatalf("Path 不符合预期: %q", list.Path)
	}
	if len(list.Entries) != 2 {
		t.Fatalf("应解析出 2 个条目，got %d", len(list.Entries))
	}
	if list.Entries[0].Name != "file1.txt" {
		t.Fatalf("第一个条目名不符: %q", list.Entries[0].Name)
	}
	if !list.Entries[1].IsDir || list.Entries[1].Name != "subdir/" {
		t.Fatalf("第二个条目应为目录: %+v", list.Entries[1])
	}
}

func TestParseFileList_Empty(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("")

	res, err := NewPhpFileList("/empty").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	list := res.(*core.FileListResult)
	if len(list.Entries) != 0 {
		t.Fatalf("空目录应返回 0 个条目，got %d", len(list.Entries))
	}
}

func TestParseFileRead(t *testing.T) {
	content := []byte("hello world binary \x00\xff")
	resp := core.NewResponse()
	resp.Body = content

	op := NewPhpFileRead("/etc/passwd")
	res, err := op.Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	fr, ok := res.(*core.FileReadResult)
	if !ok {
		t.Fatalf("期望 *core.FileReadResult，got %T", res)
	}
	if fr.Path != "/etc/passwd" {
		t.Fatalf("Path 不符合预期: %q", fr.Path)
	}
	if string(fr.Data) != string(content) {
		t.Fatalf("Data 不符合预期: got=%q want=%q", fr.Data, content)
	}
}

func TestParseFileList_RemoteError(t *testing.T) {
	resp := core.NewResponse()
	resp.NodeID = "node-1"
	resp.Body = []byte(remoteErrorPathUnavailable)

	res, err := NewPhpFileList("/missing").Parse(context.Background(), resp)
	if res != nil || !core.IsKind(err, core.ErrRemoteRuntime) {
		t.Fatalf("远端目录错误应返回 ErrRemoteRuntime: res=%v err=%v", res, err)
	}
}

func TestParseFileRead_RemoteError(t *testing.T) {
	for _, token := range []string{remoteErrorFileOpen, remoteErrorFileRead} {
		resp := core.NewResponse()
		resp.Body = []byte(token)
		res, err := NewPhpFileRead("/missing").Parse(context.Background(), resp)
		if res != nil || !core.IsKind(err, core.ErrRemoteRuntime) {
			t.Fatalf("token=%q 应返回 ErrRemoteRuntime: res=%v err=%v", token, res, err)
		}
	}
}

func TestPhpFileDownload_BuildAndParse(t *testing.T) {
	path := "/tmp/a b.bin"
	op := NewPhpFileDownload(path)
	if op.Name() != "file.download" || op.RiskLevel() != core.RiskReadOnly {
		t.Fatalf("download 名称/风险不正确")
	}
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	for _, value := range req.Params {
		decoded, decodeErr := stdbase64.StdEncoding.DecodeString(string(value))
		if decodeErr != nil || string(decoded) != path {
			t.Fatalf("download 路径 base64 不正确: %q %v", decoded, decodeErr)
		}
	}
	body := []byte{0x00, '0', 0xff}
	resp := core.NewResponse()
	resp.Body = body
	result, err := op.Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	file := result.(*core.FileReadResult)
	if file.OperationName() != "file.download" || string(file.Data) != string(body) {
		t.Fatalf("download 结果不正确: %+v", file)
	}
}

func TestPhpFileDownload_RemoteError(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte(remoteErrorFileOpen)
	res, err := NewPhpFileDownload("/missing").Parse(context.Background(), resp)
	if res != nil || !core.IsKind(err, core.ErrRemoteRuntime) {
		t.Fatalf("download 错误应返回 ErrRemoteRuntime: res=%v err=%v", res, err)
	}
}

func TestParseExec_StdoutOnly(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("command output line1\nline2")

	res, err := NewPhpExec("ls").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec, ok := res.(*core.ExecResult)
	if !ok {
		t.Fatalf("期望 *core.ExecResult，got %T", res)
	}
	if exec.ExitCode != 0 {
		t.Fatalf("纯 stdout 时 ExitCode 应为 0，got %d", exec.ExitCode)
	}
	if exec.Stdout != "command output line1\nline2" {
		t.Fatalf("Stdout 不符合预期: %q", exec.Stdout)
	}
}

func TestParseExec_WithExitCode(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("some outputret=1")

	res, err := NewPhpExec("false").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 1 {
		t.Fatalf("期望 ExitCode=1，got %d", exec.ExitCode)
	}
	if exec.Stdout != "some output" {
		t.Fatalf("Stdout 应去掉尾部 ret=，got=%q", exec.Stdout)
	}
}

func TestParseExec_OnlyExitCode(t *testing.T) {
	resp := core.NewResponse()
	resp.Body = []byte("ret=127")

	res, err := NewPhpExec("nope").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 127 {
		t.Fatalf("期望 ExitCode=127，got %d", exec.ExitCode)
	}
	if exec.Stdout != "" {
		t.Fatalf("Stdout 应为空，got=%q", exec.Stdout)
	}
}

func TestParseExec_TrailingGarbage(t *testing.T) {
	// ret= 后面不是数字 → 当无退出码处理
	resp := core.NewResponse()
	resp.Body = []byte("helloret=abc")

	res, err := NewPhpExec("x").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 0 {
		t.Fatalf("ret=abc 时 ExitCode 应为 0，got %d", exec.ExitCode)
	}
	if exec.Stdout != "helloret=abc" {
		t.Fatalf("无法解析时应原样保留 Stdout，got=%q", exec.Stdout)
	}
}

func TestParseExec_LastIndexWins(t *testing.T) {
	// 多个 ret= 时取最后一个
	resp := core.NewResponse()
	resp.Body = []byte("ret=1 middle ret=2")

	res, err := NewPhpExec("x").Parse(context.Background(), resp)
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	exec := res.(*core.ExecResult)
	if exec.ExitCode != 2 {
		t.Fatalf("应取最后一个 ret=2，got %d", exec.ExitCode)
	}
}

// phpFileUpload 的测试

func TestPhpFileUpload_NameAndRisk(t *testing.T) {
	op := NewPhpFileUpload("/tmp/x.txt", []byte("data"))
	if op.Name() != "file.upload" {
		t.Fatalf("Name 应为 file.upload，got %q", op.Name())
	}
	if op.RiskLevel() != core.RiskWrite {
		t.Fatalf("RiskLevel 应为 RiskWrite，got %q", op.RiskLevel())
	}
}

func TestPhpFileUpload_Build_SelfContained(t *testing.T) {
	op := NewPhpFileUpload("/tmp/go.php", []byte("<?php echo 1;"))
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "file.upload" {
		t.Fatalf("Operation 应为 file.upload，got %q", req.Operation)
	}
	code := string(req.Payload)
	// 自包含方案不应引用 $_POST
	if contains(code, "$_POST") {
		t.Fatalf("自包含方案不应引用 $_POST: %q", code)
	}
	// 应包含关键 PHP 函数和写文件逻辑
	for _, want := range []string{"fopen", "fwrite", "fclose", "base64_decode"} {
		if !contains(code, want) {
			t.Fatalf("Payload 缺少 %q: %q", want, code)
		}
	}
	// 默认是覆盖模式（"w"）
	if !contains(code, `"w"`) && !contains(code, `'w'`) {
		t.Fatalf("默认应为写模式 w: %q", code)
	}
}

func TestPhpFileUpload_Build_AppendMode(t *testing.T) {
	// 默认覆盖模式
	opDefault := NewPhpFileUpload("/tmp/x", []byte("data"))
	reqDefault, _ := opDefault.Build(context.Background(), nil)
	codeDefault := string(reqDefault.Payload)
	if !contains(codeDefault, `'w'`) {
		t.Fatalf("默认应为写模式 'w': %q", codeDefault)
	}

	// 追加模式
	opAppend := NewPhpFileUpload("/tmp/x", []byte("data"))
	opAppend.WithAppend(true)
	reqAppend, _ := opAppend.Build(context.Background(), nil)
	codeAppend := string(reqAppend.Payload)
	if !contains(codeAppend, `'a'`) {
		t.Fatalf("追加模式应为 'a': %q", codeAppend)
	}
	if contains(codeAppend, `'w'`) {
		t.Fatalf("追加模式不应出现 'w': %q", codeAppend)
	}
}

func TestPhpFileUpload_Build_Base64EncodesContent(t *testing.T) {
	// 验证内容确实被 base64 编码后内联
	content := []byte("hello world")
	op := NewPhpFileUpload("/tmp/x", content)
	req, _ := op.Build(context.Background(), nil)
	code := string(req.Payload)
	// base64("hello world") = "aGVsbG8gd29ybGQ="
	if !contains(code, "aGVsbG8gd29ybGQ=") {
		t.Fatalf("Payload 应包含 base64 编码后的内容: %q", code)
	}
}

func TestPhpFileUpload_Parse(t *testing.T) {
	cases := []struct {
		body   string
		wantOK bool
	}{
		{"1", true},
		{"0", false},
		{"ok", true},
		{"OK", false}, // 只认小写 "ok"
		{"anything-else", false},
	}
	for _, c := range cases {
		resp := core.NewResponse()
		resp.Body = []byte(c.body)
		res, err := NewPhpFileUpload("/x", nil).Parse(context.Background(), resp)
		if err != nil {
			t.Fatalf("Parse(%q) 出错: %v", c.body, err)
		}
		br := res.(*core.BoolResult)
		if br.OK != c.wantOK {
			t.Fatalf("body=%q 期望 OK=%v，got %v", c.body, c.wantOK, br.OK)
		}
	}
}

func TestPhpFileUpload_Parse_NilResponse(t *testing.T) {
	_, err := NewPhpFileUpload("/x", nil).Parse(context.Background(), nil)
	if err == nil {
		t.Fatal("Parse(nil) 应返回错误")
	}
}

func TestPhpInfo_Build(t *testing.T) {
	op := NewPhpInfo()
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "info" {
		t.Fatalf("Operation 应为 info，got %q", req.Operation)
	}
	if len(req.Payload) == 0 {
		t.Fatal("Payload 不应为空")
	}
	if req.Meta["adapter"] != "php" {
		t.Fatalf("Meta[adapter] 应为 php，got %q", req.Meta["adapter"])
	}
}

func TestPhpFileList_Build(t *testing.T) {
	op := NewPhpFileList("/var/www")
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "file.list" {
		t.Fatalf("Operation 应为 file.list，got %q", req.Operation)
	}
	if len(req.Params) == 0 {
		t.Fatal("Params 不应为空（应有占位符键）")
	}
	for k, v := range req.Params {
		if string(v) != stdbase64.StdEncoding.EncodeToString([]byte("/var/www")) {
			t.Fatalf("占位符 %q 应填充 base64 路径，got %q", k, v)
		}
		if !contains(string(req.Payload), `$_POST["`+k+`"]`) {
			t.Fatalf("payload 应引用同一个随机字段 %q", k)
		}
	}
}

func TestPhpFileList_DefaultPath(t *testing.T) {
	op := NewPhpFileList("")
	req, _ := op.Build(context.Background(), nil)
	for _, v := range req.Params {
		if string(v) != "Lw==" {
			t.Fatalf("空路径应默认为 / 的 base64，got %q", v)
		}
	}
}

func TestPhpFileRead_Build(t *testing.T) {
	path := `C:\\临时目录\\a b.txt`
	op := NewPhpFileRead(path)
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "file.read" {
		t.Fatalf("Operation 应为 file.read，got %q", req.Operation)
	}
	for _, value := range req.Params {
		decoded, decodeErr := stdbase64.StdEncoding.DecodeString(string(value))
		if decodeErr != nil || string(decoded) != path {
			t.Fatalf("路径 base64 不正确: decoded=%q err=%v", decoded, decodeErr)
		}
	}
}

func TestPhpExec_Build(t *testing.T) {
	op := NewPhpExec("whoami").WithBin("/bin/bash").WithEnv("FOO", "bar")
	req, err := op.Build(context.Background(), nil)
	if err != nil {
		t.Fatalf("Build 出错: %v", err)
	}
	if req.Operation != "exec" {
		t.Fatalf("Operation 应为 exec，got %q", req.Operation)
	}
	if len(req.Payload) == 0 {
		t.Fatal("Payload 不应为空")
	}
	// 自包含方案：payload 里不应出现 $_POST
	if contains(string(req.Payload), "$_POST") {
		t.Fatalf("自包含方案不应引用 $_POST，got=%q", req.Payload)
	}
}

func TestPhpExec_DefaultBin(t *testing.T) {
	op := NewPhpExec("x")
	req, _ := op.Build(context.Background(), nil)
	if !contains(string(req.Payload), "base64_decode") {
		t.Fatal("Exec payload 应包含 base64_decode")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
