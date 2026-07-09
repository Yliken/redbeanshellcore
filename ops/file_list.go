package ops

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// FileListOperation 列出远端目录。
type FileListOperation struct {
	Path string
}

// NewFileList 构建一个 FileList 操作。
func NewFileList(path string) *FileListOperation {
	if path == "" {
		path = "/"
	}
	return &FileListOperation{Path: path}
}

// Name 返回操作名。
func (op *FileListOperation) Name() string { return "file.list" }

// Build 生成带路径参数的请求。
func (op *FileListOperation) Build(_ context.Context, _ *core.Session) (*core.Request, error) {
	req := core.NewRequest(op.Name())
	req.SetParamString("path", op.Path)
	return req, nil
}

// Parse 把 AntSword 风格的 tab 分隔列表切分成 FileEntry。
//
// AntSword / Python demo 输出形如：
//
//	<name>/<tab><modtime>\t<size>\t<mode>\n     （目录）
//	<name><tab><modtime>\t<size>\t<mode>\n      （文件）
//
// 解析失败的行会写入 Metadata["unparsed"]，让 adapter 后续补充处理。
func (op *FileListOperation) Parse(_ context.Context, resp *core.Response) (core.Result, error) {
	if resp == nil {
		return nil, errors.New("file.list.Parse: 响应为空")
	}
	lines := strings.Split(strings.ReplaceAll(string(resp.Body), "\r\n", "\n"), "\n")
	entries := make([]core.FileEntry, 0, len(lines))
	var unparsed []string
	for _, line := range lines {
		line = strings.TrimRight(line, "\t\n")
		if line == "" {
			continue
		}
		name, rest, ok := strings.Cut(line, "\t")
		if !ok {
			unparsed = append(unparsed, line)
			continue
		}
		detail := strings.Split(rest, "\t")
		if len(detail) < 3 {
			unparsed = append(unparsed, line)
			continue
		}
		modstr := detail[0]
		sizestr := detail[1]
		modestr := detail[2]
		size, _ := strconv.ParseInt(sizestr, 10, 64)
		modTime, _ := time.Parse("2006-01-02 15:04:05", modstr)
		isDir := strings.HasSuffix(name, "/")
		if isDir {
			name = strings.TrimSuffix(name, "/")
		}
		entries = append(entries, core.FileEntry{
			Name:    name,
			Path:    name,
			IsDir:   isDir,
			Size:    size,
			Mode:    modestr,
			ModTime: modTime,
		})
	}
	res := &core.FileListResult{
		BaseResult: core.NewBaseResult(op.Name(), resp.Body),
		Path:       op.Path,
		Entries:    entries,
	}
	if len(unparsed) > 0 {
		res.Metas["unparsed"] = strings.Join(unparsed, "\n")
	}
	return res, nil
}

// RiskLevel 把本操作归类为只读。
func (op *FileListOperation) RiskLevel() core.RiskLevel { return core.RiskReadOnly }
