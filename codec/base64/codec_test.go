package base64

import (
	"context"
	"testing"

	"github.com/Yliken/redbeanshellcore/core"
)

func TestCodecRoundTrip(t *testing.T) {
	c := New()
	ctx := context.Background()

	in := core.NewRequest("test")
	in.Payload = []byte("hello world")
	encoded, err := c.EncodeRequest(ctx, in)
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	resp := core.NewResponse()
	resp.Body = encoded.Payload
	decoded, err := c.DecodeResponse(ctx, resp)
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}

	if string(decoded.Body) != "hello world" {
		t.Fatalf("解码结果不符合预期: %q", string(decoded.Body))
	}
}
