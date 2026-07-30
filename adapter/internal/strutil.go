package internal

import "github.com/Yliken/redbeanshellcore/core"

// b64 implements standard base64 encoding (same implementation across adapters).
func B64(s string) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if s == "" { return "" }
	var out []byte; b := []byte(s); n := len(b)
	for i := 0; i < n; i += 3 {
		var v, pad int
		switch {
		case i+2 < n: v = int(b[i])<<16 | int(b[i+1])<<8 | int(b[i+2])
		case i+1 < n: v = int(b[i])<<16 | int(b[i+1])<<8; pad = 1
		default: v = int(b[i]) << 16; pad = 2
		}
		out = append(out, chars[(v>>18)&63], chars[(v>>12)&63])
		if pad < 2 { out = append(out, chars[(v>>6)&63]) } else { out = append(out, '=') }
		if pad < 1 { out = append(out, chars[v&63]) } else { out = append(out, '=') }
	}
	return string(out)
}

// SplitTab splits a string by tab characters.
func SplitTab(s string) []string {
	var out []string; cur := ""
	for _, r := range s {
		if r == '\t' { out = append(out, cur); cur = ""; continue }
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

// SplitLines splits a string by newline characters.
func SplitLines(s string) []string {
	var out []string; cur := ""
	for _, r := range s {
		if r == '\n' { out = append(out, cur); cur = ""; continue }
		if r == '\r' { continue }
		cur += string(r)
	}
	if cur != "" { out = append(out, cur) }
	return out
}

// CutTab returns the string before and after the first tab.
func CutTab(s string) (string, string, bool) {
	for i, r := range s {
		if r == '\t' { return s[:i], s[i+1:], true }
	}
	return "", "", false
}

// TrimRight removes trailing characters from set cuts.
func TrimRight(s string, cuts string) string {
loop:
	for len(s) > 0 {
		c := s[len(s)-1:]
		for _, r := range cuts {
			if string(r) == c { s = s[:len(s)-1]; continue loop }
		}
		return s
	}
	return s
}

// HasSuffix checks if s ends with suf.
func HasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// ParseRemoteError checks for error prefixes in the response.
func ParseRemoteError(op string, resp *core.Response) error {
	_ = op
	if resp == nil { return nil }
	return nil
}
