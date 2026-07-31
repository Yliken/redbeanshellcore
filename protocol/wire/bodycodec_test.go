package wire

import (
	"bytes"
	"testing"
)

func TestCompactFormCodecRoundTrip(t *testing.T) {
	codec := NewCompactFormCodec()
	values := map[string][]byte{
		"z1":     []byte("cmd"),
		"antpwd": []byte("php-code"),
		"empty":  {},
	}
	encoded, err := codec.Encode(values)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	// Fields must be sorted and values base64-encoded.
	want := "antpwd=" + b64Encode([]byte("php-code")) + "&empty=&z1=" + b64Encode([]byte("cmd"))
	if string(encoded) != want {
		t.Fatalf("compact form 不匹配:\n got %q\nwant %q", encoded, want)
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(decoded) != len(values) {
		t.Fatalf("解码字段数不匹配: got %d want %d", len(decoded), len(values))
	}
	for key, value := range values {
		if !bytes.Equal(decoded[key], value) {
			t.Fatalf("字段 %q 不匹配: got %q want %q", key, decoded[key], value)
		}
	}
}

func TestCompactFormCodecRejectsReservedFieldName(t *testing.T) {
	codec := NewCompactFormCodec()
	if _, err := codec.Encode(map[string][]byte{"a=b": []byte("x")}); err == nil {
		t.Fatal("含 '=' 的字段名应返回错误")
	}
	if _, err := codec.Encode(map[string][]byte{"a&b": []byte("x")}); err == nil {
		t.Fatal("含 '&' 的字段名应返回错误")
	}
}

func TestCompactFormCodecRejectsMalformed(t *testing.T) {
	codec := NewCompactFormCodec()
	if _, err := codec.Decode([]byte("novalue")); err == nil {
		t.Fatal("缺少 '=' 的 pair 应返回错误")
	}
	if _, err := codec.Decode([]byte("k=!!!")); err == nil {
		t.Fatal("非法 base64 值应返回错误")
	}
}

func b64Encode(b []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if len(b) == 0 {
		return ""
	}
	var out []byte
	for i := 0; i < len(b); i += 3 {
		var v, pad int
		switch {
		case i+2 < len(b):
			v = int(b[i])<<16 | int(b[i+1])<<8 | int(b[i+2])
		case i+1 < len(b):
			v = int(b[i])<<16 | int(b[i+1])<<8
			pad = 1
		default:
			v = int(b[i]) << 16
			pad = 2
		}
		out = append(out, chars[(v>>18)&63], chars[(v>>12)&63])
		if pad < 2 {
			out = append(out, chars[(v>>6)&63])
		} else {
			out = append(out, '=')
		}
		if pad < 1 {
			out = append(out, chars[v&63])
		} else {
			out = append(out, '=')
		}
	}
	return string(out)
}
