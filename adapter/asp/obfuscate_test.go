package asp

import (
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestNewObfuscator_GeneratesUniqueNames(t *testing.T) {
	o1 := NewObfuscator()
	o2 := NewObfuscator()
	if o1.Param1() == o2.Param1() {
		t.Error("expected unique Param1")
	}
	if o1.Param2() == o2.Param2() {
		t.Error("expected unique Param2")
	}
	if o1.HelperDecode() == o2.HelperDecode() {
		t.Error("expected unique HelperDecode")
	}
	if o1.ErrPrefix() == o2.ErrPrefix() {
		t.Error("expected unique ErrPrefix")
	}
}

func TestDefaultObfuscator_FixedNames(t *testing.T) {
	o := DefaultObfuscator()
	if o.Param1() != "z1" {
		t.Errorf("expected Param1=z1, got %q", o.Param1())
	}
	if o.Param2() != "z2" {
		t.Errorf("expected Param2=z2, got %q", o.Param2())
	}
	if o.HelperDecode() != "b64d" {
		t.Errorf("expected HelperDecode=b64d, got %q", o.HelperDecode())
	}
	if o.HelperEncode() != "b64e" {
		t.Errorf("expected HelperEncode=b64e, got %q", o.HelperEncode())
	}
}

func TestObfuscator_HelperCode_ContainsFunctionNames(t *testing.T) {
	o := NewObfuscator()
	code := o.HelperCode()
	if len(code) == 0 {
		t.Fatal("HelperCode returned empty string")
	}
	if !strings.Contains(code, o.HelperDecode()) {
		t.Errorf("HelperCode should contain HelperDecode %q", o.HelperDecode())
	}
	if !strings.Contains(code, o.HelperEncode()) {
		t.Errorf("HelperCode should contain HelperEncode %q", o.HelperEncode())
	}
	if !strings.Contains(code, o.HelperBTS()) {
		t.Errorf("HelperCode should contain HelperBTS %q", o.HelperBTS())
	}
}

func TestTemplateSubst_ReplacesNames(t *testing.T) {
	code := `Function b64d(s):BytesToStr(.NodeTypedValue):Response.Write b64e(wd):Request.Form("z1"):Dim x=Request.Form("z2")`
	o := NewObfuscator()
	result := templateSubst(code, o)
	if strings.Contains(result, `b64d(`) {
		t.Errorf("templateSubst should replace b64d(, got %q", result)
	}
	if strings.Contains(result, `b64e(`) {
		t.Errorf("templateSubst should replace b64e(, got %q", result)
	}
	if strings.Contains(result, `Request.Form("z1")`) {
		t.Errorf(`templateSubst should replace Request.Form("z1"), got %q`, result)
	}
	if strings.Contains(result, `Request.Form("z2")`) {
		t.Errorf(`templateSubst should replace Request.Form("z2"), got %q`, result)
	}
	if !strings.Contains(result, o.HelperDecode()+"(") {
		t.Errorf("result should contain obfuscated decode call, got %q", result)
	}
}

func TestAspParseInfo_ReturnsInfoResult(t *testing.T) {
	body := []byte("d29ya2Rpcg==\tT1NfbmFtZQ==\tVXNlck5hbWU=")
	result := parseInfo(body)
	_, ok := result.(*core.InfoResult)
	if !ok {
		t.Fatalf("expected *core.InfoResult, got %T", result)
	}
}

func TestAspParseRemoteError_DetectsError(t *testing.T) {
	resp := &core.Response{
		Body:   []byte("ERR:TEST:error"),
		NodeID: "n1",
	}
	err := parseRemoteError("test", resp)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	errOp, ok := err.(*core.OpError)
	if !ok {
		t.Fatalf("expected *core.OpError, got %T", err)
	}
	if errOp.Kind != core.ErrRemoteRuntime {
		t.Errorf("expected Kind %v, got %v", core.ErrRemoteRuntime, errOp.Kind)
	}
}

func TestAspParseRemoteError_NilResponse(t *testing.T) {
	err := parseRemoteError("test", nil)
	if err != nil {
		t.Errorf("expected nil for nil response, got %v", err)
	}
}

func TestAspParseRemoteError_NoErrorPrefix(t *testing.T) {
	resp := &core.Response{
		Body:   []byte("normal output"),
		NodeID: "n1",
	}
	err := parseRemoteError("test", resp)
	if err != nil {
		t.Errorf("expected nil for normal output, got %v", err)
	}
}
