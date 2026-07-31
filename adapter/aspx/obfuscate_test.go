package aspx

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
	if o1.HelperDecode() == o2.HelperDecode() {
		t.Error("expected unique HelperDecode")
	}
	if o1.ClassName() == o2.ClassName() {
		t.Error("expected unique ClassName")
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
	if o.HelperDecode() != "d" {
		t.Errorf("expected HelperDecode=d, got %q", o.HelperDecode())
	}
	if o.HelperEncode() != "e" {
		t.Errorf("expected HelperEncode=e, got %q", o.HelperEncode())
	}
	if o.ClassName() != "E" {
		t.Errorf("expected ClassName=E, got %q", o.ClassName())
	}
}

func TestObfuscator_CsWrap_ContainsNames(t *testing.T) {
	o := NewObfuscator()
	code := o.CsWrap("testcode;")
	if !strings.Contains(code, o.HelperDecode()) {
		t.Errorf("CsWrap should contain HelperDecode %q", o.HelperDecode())
	}
	if !strings.Contains(code, o.HelperEncode()) {
		t.Errorf("CsWrap should contain HelperEncode %q", o.HelperEncode())
	}
	if !strings.Contains(code, o.ClassName()) {
		t.Errorf("CsWrap should contain ClassName %q", o.ClassName())
	}
}

func TestTemplateSubst_ReplacesNames(t *testing.T) {
	code := `Request.Form["z1"]:Request.Form["z2"]:d(:e(:"E"`
	o := NewObfuscator()
	result := templateSubst(code, o)
	if strings.Contains(result, `Request.Form["z1"]`) {
		t.Error("templateSubst should replace z1")
	}
	if strings.Contains(result, `"d(`) && !strings.Contains(result, `"`+o.HelperDecode()+`(`) {
		t.Error("templateSubst should replace d( with obfuscated name")
	}
}

func TestAspxParseRemoteError_DetectsError(t *testing.T) {
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

func TestAspxParseRemoteError_NilResponse(t *testing.T) {
	err := parseRemoteError("test", nil)
	if err != nil {
		t.Errorf("expected nil for nil response, got %v", err)
	}
}

func TestAspxParseRemoteError_NoErrorPrefix(t *testing.T) {
	resp := &core.Response{
		Body:   []byte("normal output"),
		NodeID: "n1",
	}
	err := parseRemoteError("test", resp)
	if err != nil {
		t.Errorf("expected nil for normal output, got %v", err)
	}
}
