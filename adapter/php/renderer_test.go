package php

import (
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestRenderer_Info_ReturnsCode(t *testing.T) {
	tpl := NewPHPTemplates()
	code, placeholders := tpl.Info()
	if code == "" {
		t.Fatal("Info 模板不应返回空代码")
	}
	if placeholders != nil && len(placeholders) != 0 {
		t.Fatalf("Info 模板不应有占位符，got %v", placeholders)
	}
	// 应包含关键 PHP 函数
	for _, want := range []string{"dirname", "php_uname", "get_current_user"} {
		if !strings.Contains(code, want) {
			t.Fatalf("Info 模板缺少 %q", want)
		}
	}
}

func TestRenderer_FileList_ReturnsPlaceholders(t *testing.T) {
	tpl := NewPHPTemplates()
	code, placeholders := tpl.FileList()
	if code == "" {
		t.Fatal("FileList 模板不应返回空代码")
	}
	if len(placeholders) == 0 {
		t.Fatal("FileList 模板应至少有一个占位符")
	}
	// 占位符值应为 base64::path
	for _, v := range placeholders {
		if v != placeholderBase64Path {
			t.Fatalf("FileList 占位符值应为 %q，got %q", placeholderBase64Path, v)
		}
	}
}

func TestRenderer_FileRead_ReturnsPlaceholders(t *testing.T) {
	tpl := NewPHPTemplates()
	code, placeholders := tpl.FileRead()
	if code == "" {
		t.Fatal("FileRead 模板不应返回空代码")
	}
	if len(placeholders) == 0 {
		t.Fatal("FileRead 模板应至少有一个占位符")
	}
}

func TestRenderer_FileOperationsHaveDeterministicErrors(t *testing.T) {
	tpl := NewPHPTemplates()
	listCode, _ := tpl.FileList()
	if !strings.Contains(listCode, remoteErrorPathUnavailable) || !strings.Contains(listCode, "===false") {
		t.Fatalf("FileList 应包含确定性错误和严格失败判断: %q", listCode)
	}
	readCode, _ := tpl.FileRead()
	for _, want := range []string{"\"rb\"", "stream_get_contents", remoteErrorFileOpen, remoteErrorFileRead} {
		if !strings.Contains(readCode, want) {
			t.Fatalf("FileRead 缺少 %q: %q", want, readCode)
		}
	}
	downloadCode, placeholders := tpl.FileDownload()
	if len(placeholders) == 0 {
		t.Fatal("FileDownload 应包含路径占位符")
	}
	if strings.Contains(downloadCode, "fgetc") {
		t.Fatalf("FileDownload 不应使用 fgetc 真值判断: %q", downloadCode)
	}
	for _, want := range []string{"\"rb\"", "readfile", remoteErrorFileOpen, remoteErrorFileRead} {
		if !strings.Contains(downloadCode, want) {
			t.Fatalf("FileDownload 缺少 %q: %q", want, downloadCode)
		}
	}
}

func TestRenderer_Exec_ReturnsPlaceholders(t *testing.T) {
	tpl := NewPHPTemplates()
	code, placeholders := tpl.Exec()
	if code == "" {
		t.Fatal("Exec 模板不应返回空代码")
	}
	// Exec 模板有 3 个占位符：bin / cmd / env
	if len(placeholders) != 3 {
		t.Fatalf("Exec 模板应有 3 个占位符，got %d", len(placeholders))
	}
	expected := map[string]bool{
		placeholderBase64Bin: false,
		placeholderBase64Cmd: false,
		placeholderBase64Env: false,
	}
	for _, v := range placeholders {
		if _, ok := expected[v]; ok {
			expected[v] = true
		}
	}
	for k, found := range expected {
		if !found {
			t.Fatalf("Exec 模板缺少占位符 %q", k)
		}
	}
}

func TestFillPlaceholders_ReplacesAll(t *testing.T) {
	a := New()
	code := "header#{base64::path}middle#{base64::bin}tail"
	params := map[string][]byte{
		"path": []byte("/tmp"),
		"bin":  []byte("/bin/sh"),
	}
	out := a.FillPlaceholders(code, params)
	if strings.Contains(out, "#{base64::") {
		t.Fatalf("占位符未被替换: %q", out)
	}
	// 替换后应出现 base64 编码后的值
	if !strings.Contains(out, "L3RtcA==") { // base64("/tmp")
		t.Fatalf("path 未被 base64 编码: %q", out)
	}
}

func TestFillPlaceholders_MissingKey(t *testing.T) {
	a := New()
	code := "header#{base64::path}tail"
	// 不传 path 参数
	out := a.FillPlaceholders(code, map[string][]byte{})
	// 当前行为：缺失的占位符原样保留
	if !strings.Contains(out, "#{base64::path}") {
		t.Fatalf("缺失的占位符应原样保留，got=%q", out)
	}
}

func TestFillPlaceholders_EmptyParams(t *testing.T) {
	a := New()
	code := "no placeholders here"
	out := a.FillPlaceholders(code, map[string][]byte{})
	if out != code {
		t.Fatalf("无占位符时应原样返回，got=%q", out)
	}
}

func TestRandomVar_Uniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		v := randomVar6()
		if seen[v] {
			t.Fatalf("randomVar6 产生重复值: %q", v)
		}
		seen[v] = true
		if len(v) != 6 {
			t.Fatalf("randomVar6 长度应为 6，got %d (%q)", len(v), v)
		}
	}
}

func TestAdapter_Capabilities(t *testing.T) {
	a := New()
	caps := a.Capabilities()
	if len(caps) == 0 {
		t.Fatal("Capabilities 不应为空")
	}
	// 应包含 info / exec / file.list / file.read
	want := map[core.Capability]bool{
		core.CapInfo:     false,
		core.CapExec:     false,
		core.CapFileList: false,
		core.CapFileRead: false,
	}
	for _, c := range caps {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for k, found := range want {
		if !found {
			t.Fatalf("Capabilities 缺少 %q", k)
		}
	}
}

func TestAdapter_HasCapability(t *testing.T) {
	a := New()
	caps := a.Capabilities()
	for _, c := range caps {
		if !a.caps.HasCapability(c) {
			t.Fatalf("HasCapability(%q) 应为 true", c)
		}
	}
	// CapFileWrite 已被注释掉，适配器不应声明它
	if a.caps.HasCapability(core.Capability("file.write")) {
		t.Fatal("HasCapability(file.write) 应为 false（已注释掉）")
	}
}

// 防止 import 被误删
var _ = strings.HasPrefix
