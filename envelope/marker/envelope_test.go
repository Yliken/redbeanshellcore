package marker

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestGenerate_Unique(t *testing.T) {
	e := New()
	a, err := e.generate()
	if err != nil {
		t.Fatalf("第一次 generate 出错: %v", err)
	}
	b, err := e.generate()
	if err != nil {
		t.Fatalf("第二次 generate 出错: %v", err)
	}
	if a == b {
		t.Fatalf("两次 generate 结果相同，期望随机唯一: %q", a)
	}
}

func TestGenerate_CustomLength(t *testing.T) {
	e := NewWithLength(8)
	tag, err := e.generate()
	if err != nil {
		t.Fatalf("generate 出错: %v", err)
	}
	if len(tag) != 8 {
		t.Fatalf("期望 tag 长度 8，实际 %d (%q)", len(tag), tag)
	}
}

func TestNewWithLength_DefaultOnZero(t *testing.T) {
	e := NewWithLength(0)
	if e.TagLen != 16 {
		t.Fatalf("TagLen=0 时应回退到默认 16，实际 %d", e.TagLen)
	}
}

func TestWrap_InjectsTags(t *testing.T) {
	e := NewWithLength(4)
	req := core.NewRequest("op")
	req.Payload = []byte("hello")

	out, err := e.Wrap(context.Background(), req)
	if err != nil {
		t.Fatalf("Wrap 出错: %v", err)
	}

	tagS := out.Meta["marker.tag_s"]
	tagE := out.Meta["marker.tag_e"]
	if tagS == "" || tagE == "" {
		t.Fatalf("Meta 里缺少 tag_s/tag_e: %+v", out.Meta)
	}

	want := tagS + "hello" + tagE
	if !bytes.Equal(out.Payload, []byte(want)) {
		t.Fatalf("payload 不符合 tagS+body+tagE，got=%q want prefix=%q suffix=%q",
			out.Payload, tagS, tagE)
	}
}

func TestWrap_PHPUsesEchoStatements(t *testing.T) {
	e := NewWithLength(4)
	req := core.NewRequest("info")
	req.Meta["adapter"] = "php"
	req.Payload = []byte(`echo "body";`)

	out, err := e.Wrap(context.Background(), req)
	if err != nil {
		t.Fatalf("Wrap 出错: %v", err)
	}
	tagS := out.Meta["marker.tag_s"]
	tagE := out.Meta["marker.tag_e"]
	payload := string(out.Payload)
	if !strings.HasPrefix(payload, "echo '"+tagS+"';") {
		t.Fatalf("PHP payload 应用 echo 输出 start marker: %q", payload)
	}
	if !strings.Contains(payload, `echo "body";`) {
		t.Fatalf("原 PHP payload 应保持不变: %q", payload)
	}
	if !strings.HasSuffix(payload, "echo '"+tagE+"';") {
		t.Fatalf("PHP payload 应用 echo 输出 end marker: %q", payload)
	}
	if strings.HasPrefix(payload, tagS) {
		t.Fatalf("PHP payload 不应以裸 marker 开头: %q", payload)
	}
}

func TestWrap_NilRequest(t *testing.T) {
	e := New()
	_, err := e.Wrap(context.Background(), nil)
	if err == nil {
		t.Fatal("Wrap(nil) 应返回错误")
	}
}

func TestExtract_FullWindow(t *testing.T) {
	e := New()
	req := core.NewRequest("op")
	req.Payload = []byte("real-body")
	wrapped, err := e.Wrap(context.Background(), req)
	if err != nil {
		t.Fatalf("Wrap 出错: %v", err)
	}

	resp := core.NewResponse()
	resp.Meta["marker.tag_s"] = wrapped.Meta["marker.tag_s"]
	resp.Meta["marker.tag_e"] = wrapped.Meta["marker.tag_e"]
	// 模拟远端输出：一些前缀 + tagS + 真实内容 + tagE + 一些后缀
	resp.Body = []byte("garbage" + string(wrapped.Payload) + "trailer")

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	if string(got.Body) != "real-body" {
		t.Fatalf("期望截取到 real-body，got=%q", got.Body)
	}
}

func TestExtract_MissingMetaTags(t *testing.T) {
	e := New()
	resp := core.NewResponse()
	resp.Body = []byte("untouched body")

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	if string(got.Body) != "untouched body" {
		t.Fatalf("Meta 缺失时应原样返回 body，got=%q", got.Body)
	}
}

func TestExtract_MissingStartTag(t *testing.T) {
	e := New()
	resp := core.NewResponse()
	resp.Meta["marker.tag_s"] = "notexiststart"
	resp.Meta["marker.tag_e"] = "notexistend"
	resp.Body = []byte("body without tags")

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	if string(got.Body) != "body without tags" {
		t.Fatalf("找不到 start tag 时应原样返回 body，got=%q", got.Body)
	}
}

func TestExtract_MissingEndTag(t *testing.T) {
	e := New()
	req := core.NewRequest("op")
	wrapped, _ := e.Wrap(context.Background(), req)

	resp := core.NewResponse()
	resp.Meta["marker.tag_s"] = wrapped.Meta["marker.tag_s"]
	resp.Meta["marker.tag_e"] = "never-appear-end"
	resp.Body = []byte(wrapped.Meta["marker.tag_s"] + "some-stuff-but-no-end")

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	// 有 start 但找不到 end → 当前行为原样返回
	if !bytes.Equal(got.Body, resp.Body) {
		t.Fatalf("找不到 end tag 时应原样返回，got=%q", got.Body)
	}
}

func TestExtract_NestedTags_TakesFirstWindow(t *testing.T) {
	e := New()
	req := core.NewRequest("op")
	req.Payload = []byte("first")
	wrapped1, _ := e.Wrap(context.Background(), req)

	req2 := core.NewRequest("op")
	req2.Payload = []byte("second")
	wrapped2, _ := e.Wrap(context.Background(), req2)

	// body 里有两个完整的 tagS/tagE 窗口
	full := string(wrapped1.Payload) + "middle" + string(wrapped2.Payload)
	resp := core.NewResponse()
	resp.Meta["marker.tag_s"] = wrapped1.Meta["marker.tag_s"]
	resp.Meta["marker.tag_e"] = wrapped1.Meta["marker.tag_e"]
	resp.Body = []byte(full)

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	// 用第一个 tagS 和它之后出现的第一个 tagE 截取
	if string(got.Body) != "first" {
		t.Fatalf("期望取第一个窗口 first，got=%q", got.Body)
	}
}

func TestExtract_NilResponse(t *testing.T) {
	e := New()
	_, err := e.Extract(context.Background(), nil)
	if err == nil {
		t.Fatal("Extract(nil) 应返回错误")
	}
}

func TestRoundTrip_WrapThenExtract(t *testing.T) {
	e := New()
	payload := []byte("the-precious-payload")

	req := core.NewRequest("op")
	req.Payload = payload
	wrapped, err := e.Wrap(context.Background(), req)
	if err != nil {
		t.Fatalf("Wrap 出错: %v", err)
	}

	resp := core.NewResponse()
	resp.Meta["marker.tag_s"] = wrapped.Meta["marker.tag_s"]
	resp.Meta["marker.tag_e"] = wrapped.Meta["marker.tag_e"]
	resp.Body = wrapped.Payload // 远端原样输出

	got, err := e.Extract(context.Background(), resp)
	if err != nil {
		t.Fatalf("Extract 出错: %v", err)
	}
	if !bytes.Equal(got.Body, payload) {
		t.Fatalf("往返后 payload 不一致: got=%q want=%q", got.Body, payload)
	}
}

func TestTagContainsOnlyHex(t *testing.T) {
	e := NewWithLength(16)
	tag, err := e.generate()
	if err != nil {
		t.Fatalf("generate 出错: %v", err)
	}
	if !isHex(tag) {
		t.Fatalf("tag 应只含十六进制字符，got=%q", tag)
	}
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// 防止未来重构时误删 import 的保守检查
var _ = strings.HasPrefix
