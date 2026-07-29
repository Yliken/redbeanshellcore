// Package httpform 字段生成器：动态生成符合常见表单命名模式的随机字段名。
package httpform

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sync"
)

// FieldNamingPattern 定义表单字段名的生成风格。
type FieldNamingPattern int

const (
	PatternDefault       FieldNamingPattern = iota
	PatternShort
	PatternUnderscore
	PatternCamelCase
	PatternNumericPrefix
	PatternRandomWord
)

// FieldGenerator 生成随机的表单字段名，每次调用产生不同风格的字段名。
type FieldGenerator struct {
	mu      sync.RWMutex
	pattern FieldNamingPattern
	used    map[string]bool
}

// NewFieldGenerator 构建一个默认 FieldGenerator。
func NewFieldGenerator() *FieldGenerator {
	return &FieldGenerator{
		pattern: PatternDefault,
		used:    make(map[string]bool),
	}
}

// WithPattern 设置字段命名模式。
func (g *FieldGenerator) WithPattern(p FieldNamingPattern) *FieldGenerator {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pattern = p
	return g
}

// Generate 生成一个不重复的随机字段名。
func (g *FieldGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	pattern := g.pattern
	if pattern == PatternDefault {
		n, _ := rand.Int(rand.Reader, big.NewInt(5))
		pattern = FieldNamingPattern(int(n.Int64()) + 1)
	}

	for attempts := 0; attempts < 50; attempts++ {
		name := generateFieldName(pattern)
		if !g.used[name] {
			g.used[name] = true
			return name
		}
	}
	return generateFieldName(PatternShort)
}

// Reset 清除已使用字段名集合（可在每请求开始前调用）。
func (g *FieldGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.used = make(map[string]bool)
}

func generateFieldName(pattern FieldNamingPattern) string {
	switch pattern {
	case PatternShort:
		return pickShortName()
	case PatternUnderscore:
		return pickUnderscoreName()
	case PatternCamelCase:
		return pickCamelCaseName()
	case PatternNumericPrefix:
		return pickNumericPrefixName()
	case PatternRandomWord:
		return pickRandomWordName()
	default:
		return pickShortName()
	}
}

var shortNames = []string{
	"a", "pwd", "key", "data", "cmd", "id", "act", "do",
	"q", "s", "f", "t", "v", "x", "p", "k",
	"pass", "user", "file", "code", "input", "text", "val", "str",
}

var underscoreNames = []string{
	"auth_password", "user_token", "form_data", "input_code",
	"field_value", "request_key", "session_id", "action_type",
	"submit_data", "encrypted_payload", "command_string",
	"file_content", "upload_path", "parameter_list",
}

var camelCaseNames = []string{
	"authPassword", "userToken", "formData", "inputCode",
	"fieldValue", "requestKey", "sessionId", "actionType",
	"submitData", "encryptedPayload", "commandString",
	"fileContent", "uploadPath", "parameterList",
}

var wordPool = []string{
	"alpha", "beta", "delta", "gamma", "omega", "sigma",
	"token", "hash", "salt", "nonce", "proof", "chunk",
	"cloud", "edge", "core", "meta", "data", "info",
	"node", "grid", "light", "prime", "scope", "trace",
}

func pickShortName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(shortNames))))
	return shortNames[n.Int64()]
}

func pickUnderscoreName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(underscoreNames))))
	return underscoreNames[n.Int64()]
}

func pickCamelCaseName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(camelCaseNames))))
	return camelCaseNames[n.Int64()]
}

func pickNumericPrefixName() string {
	bases := []string{"p", "f", "i", "in", "k", "v", "param", "field", "input"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(bases))))
	base := bases[n.Int64()]
	suffixN, _ := rand.Int(rand.Reader, big.NewInt(90))
	suffix := int(suffixN.Int64()) + 10
	return base + "_" + itoa(suffix)
}

func pickRandomWordName() string {
	n1, _ := rand.Int(rand.Reader, big.NewInt(int64(len(wordPool))))
	n2, _ := rand.Int(rand.Reader, big.NewInt(int64(len(wordPool))))
	return wordPool[n1.Int64()] + "_" + wordPool[n2.Int64()]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func randomHex(n int) string {
	buf := make([]byte, (n+1)/2)
	_, _ = rand.Read(buf)
	out := hex.EncodeToString(buf)
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// HoneypotFields 生成一组模仿真实 Web 表单的诱饵字段。
// 字段名和值在形态上贴近实际 PHP 应用（CSRF Token、分页参数、时间戳、框架隐藏字段等）。
type HoneypotFields struct {
	mu        sync.RWMutex
	count     int
	generator *FieldGenerator
}

// NewHoneypotFields 构建一个 Honeypot 字段生成器。
func NewHoneypotFields(count int) *HoneypotFields {
	if count < 0 {
		count = 0
	}
	if count > 20 {
		count = 20
	}
	return &HoneypotFields{
		count:     count,
		generator: NewFieldGenerator(),
	}
}

// Generate 生成 count 个模仿真实表单的诱饵字段。
func (h *HoneypotFields) Generate() map[string]string {
	fields := make(map[string]string)
	if h.count <= 0 {
		return fields
	}

	templates := buildHoneypotTemplates()
	shuffled := shuffleTemplates(templates)

	count := h.count
	if count > len(shuffled) {
		count = len(shuffled)
	}

	for i := 0; i < count; i++ {
		name, value := shuffled[i]()
		fields[name] = value
	}
	return fields
}

func buildHoneypotTemplates() []func() (string, string) {
	return []func() (string, string){
		// CSRF 类
		func() (string, string) { return "_token", randomHex(40) },
		func() (string, string) { return "_csrf_token", randomHex(32) },
		func() (string, string) { return "csrfmiddlewaretoken", randomHex(64) },
		func() (string, string) { return "_csrf", randomHex(32) },
		func() (string, string) { return "token", randomHex(32) },

		// 时间戳 / Nonce 类
		func() (string, string) { return "_t", randomTimestamp() },
		func() (string, string) { return "_ts", randomTimestamp() },
		func() (string, string) { return "nonce", randomHex(24) },
		func() (string, string) { return "_nonce", randomHex(16) },

		// AJAX cachebuster
		func() (string, string) { return "_", randomTimestamp() },
		func() (string, string) { return "_dc", randomTimestamp() },

		// 表单控制类
		func() (string, string) { return "_method", pickHTTPMethod() },
		func() (string, string) { return "action", pickAction() },
		func() (string, string) { return "format", "json" },
		func() (string, string) { return "_format", "json" },

		// 分页 / 查询类
		func() (string, string) { return "page", randomIntStr(1, 50) },
		func() (string, string) { return "per_page", randomIntStr(10, 100) },
		func() (string, string) { return "limit", randomIntStr(10, 100) },
		func() (string, string) { return "offset", randomIntStr(0, 200) },
		func() (string, string) { return "sort", pickSort() },
		func() (string, string) { return "order", pickOrder() },

		// WordPress 类
		func() (string, string) { return "_wpnonce", randomHex(20) },
		func() (string, string) { return "_wp_http_referer", "/wp-admin/" },

		// 会话类
		func() (string, string) { return "session_id", randomHex(32) },
		func() (string, string) { return "s", randomHex(26) },

		// 真实应用常见字段
		func() (string, string) { return "remember", pickRemember() },
		func() (string, string) { return "terms", "1" },
		func() (string, string) { return "subscribe", "0" },
	}
}

func shuffleTemplates(t []func() (string, string)) []func() (string, string) {
	s := make([]func() (string, string), len(t))
	copy(s, t)
	for i := len(s) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		s[i], s[j.Int64()] = s[j.Int64()], s[i]
	}
	return s
}

func randomTimestamp() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(86400000))
	base := int64(1785369600000)
	return itoa64(base + n.Int64())
}

func pickHTTPMethod() string {
	opts := []string{"PUT", "PATCH", "DELETE"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(opts))))
	return opts[n.Int64()]
}

func pickAction() string {
	opts := []string{"save", "update", "delete", "publish", "draft", "preview"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(opts))))
	return opts[n.Int64()]
}

func pickSort() string {
	opts := []string{"id", "name", "date", "created_at", "updated_at", "title", "status"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(opts))))
	return opts[n.Int64()]
}

func pickOrder() string {
	opts := []string{"asc", "desc"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(opts))))
	return opts[n.Int64()]
}

func pickRemember() string {
	opts := []string{"0", "1", "on"}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(opts))))
	return opts[n.Int64()]
}

func randomIntStr(min, max int64) string {
	delta := max - min
	if delta <= 0 {
		delta = 1
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(delta))
	return itoa64(min + n.Int64())
}

// SetCount 设置诱饵字段数量。
func (h *HoneypotFields) SetCount(count int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if count < 0 {
		count = 0
	}
	if count > 20 {
		count = 20
	}
	h.count = count
}

// Count 返回当前诱饵字段数量。
func (h *HoneypotFields) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}