package jsp

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// Parser converts raw response bytes into structured core.Result values.
type Parser struct{}

// NewParser builds a jsp Parser instance.
func NewParser() *Parser { return &Parser{} }

// parseInfo parses Info responses from the JSP shell.
// JSP shell returns: base64(workdir)\tbase64(os)\tbase64(user)
func parseInfo(body []byte) core.Result {
	raw := string(body)
	parts := splitTab(raw)
	res := &core.InfoResult{BaseResult: core.NewBaseResult("info", body)}
	switch {
	case len(parts) >= 4:
		res.Workdir = fromB64(parts[0])
		res.OS = fromB64(parts[2])
		res.User = fromB64(parts[3])
	case len(parts) >= 3:
		res.Workdir = fromB64(parts[0])
		res.OS = fromB64(parts[1])
		res.User = fromB64(parts[2])
	case len(parts) >= 1:
		res.OS = fromB64(parts[0])
	}
	return res
}

// parseExec parses Exec responses. Compatible with PHP format: output + ret=<code>
func parseExec(body []byte) core.Result {
	s := string(body)
	out := &core.ExecResult{
		BaseResult: core.NewBaseResult("exec", body),
		Stdout:     s,
	}
	const prefix = "ret="
	if idx := strings.LastIndex(s, prefix); idx >= 0 {
		codeStr := strings.TrimSpace(s[idx+len(prefix):])
		if code, err := atoi(codeStr); err == nil {
			out.ExitCode = code
			out.Stdout = strings.TrimSpace(s[:idx])
		}
	}
	return out
}

// parseFileList parses FileList responses. Same tab-separated format as PHP.
func parseFileList(target string, body []byte) core.Result {
	lines := splitLines(string(body))
	var entries []core.FileEntry
	for _, line := range lines {
		line = trimRight(line, "\t\n")
		if len(line) == 0 {
			continue
		}
		name, rest, ok := cutTab(line)
		if !ok {
			continue
		}
		detail := splitTab(rest)
		if len(detail) < 3 {
			continue
		}
		entries = append(entries, core.FileEntry{
			Name:  name,
			IsDir: hasSuffix(name, "/"),
		})
	}
	return &core.FileListResult{
		BaseResult: core.NewBaseResult("file.list", body),
		Path:       target,
		Entries:    entries,
	}
}

// parseFileRead parses FileRead/FileDownload responses.
// JSP shell returns base64-encoded content; we decode back to raw bytes.
func parseFileRead(operation, target string, body []byte) core.Result {
	data := b64DecodeBytes(body)
	return &core.FileReadResult{
		BaseResult: core.NewBaseResult(operation, body),
		Path:       target,
		Data:       data,
	}
}

// parseRemoteError checks if the response indicates a remote error.
func parseRemoteError(operation string, resp *core.Response) error {
	if resp == nil {
		return nil
	}
	body := strings.TrimSpace(string(resp.Body))
	if strings.HasPrefix(body, "ERR:") {
		return core.NewOpError(core.ErrRemoteRuntime, operation, resp.NodeID,
			"remote error: "+body, nil)
	}
	return nil
}

// fromB64 decodes a base64 string to plain text.
func fromB64(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(decoded)
}

// b64DecodeBytes decodes base64 bytes to raw bytes.
func b64DecodeBytes(src []byte) []byte {
	dst := make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(dst, src)
	if err != nil {
		out := make([]byte, len(src))
		copy(out, src)
		return out
	}
	return dst[:n]
}

// parse helpers shared with the operations package.

func splitTab(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\t' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		if r == '\r' {
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func cutTab(s string) (string, string, bool) {
	for i, r := range s {
		if r == '\t' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

func trimRight(s string, cuts string) string {
loop:
	for len(s) > 0 {
		c := s[len(s)-1:]
		for _, r := range cuts {
			if string(r) == c {
				s = s[:len(s)-1]
				continue loop
			}
		}
		return s
	}
	return s
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// atoi returns 0 and an error if s contains non-numeric characters.
func atoi(s string) (int, error) {
	if s == "" {
		return 0, errors.New("empty string")
	}
	n := 0
	neg := false
	start := 0
	if s[0] == '-' {
		neg = true
		start = 1
	} else if s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return 0, errors.New("only sign")
	}
	for _, c := range s[start:] {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}
