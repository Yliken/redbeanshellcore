// Package useragent 提供可配置的 User-Agent 轮换池。
//
// 支持内置常见浏览器 UA 列表、自定义 UA 池注入、每个请求或每个会话随机选取，
// 并关联 Accept / Accept-Language / Accept-Encoding / Sec-CH-UA 等头部的协同轮换。
package useragent

import (
	"crypto/rand"
	"math/big"
	"sync"
)

// Profile 描述一个浏览器指纹的完整 HTTP 头部组合。
type Profile struct {
	UserAgent       string
	Accept          string
	AcceptLanguage  string
	AcceptEncoding  string
	SecCHUA         string // Sec-CH-UA
	SecCHUAMobile   string // Sec-CH-UA-Mobile
	SecCHUAPlatform string // Sec-CH-UA-Platform
}

// Selector 决定从池中选取 Profile 的策略。
type Selector int

const (
	SelectorRandom    Selector = iota // 每次随机选
	SelectorRoundRobin               // 轮询
)

// Pool 管理一组浏览器 UA Profile，支持线程安全的选取。
type Pool struct {
	mu       sync.RWMutex
	profiles []Profile
	selector Selector
	next     int
}

// New 构建一个含默认浏览器 UA 列表的 Pool。
func New() *Pool {
	return &Pool{
		profiles: defaultProfiles(),
		selector: SelectorRandom,
	}
}

// NewWithProfiles 用自定义 Profile 列表构建 Pool。
func NewWithProfiles(profiles []Profile) *Pool {
	p := &Pool{
		profiles: make([]Profile, len(profiles)),
		selector: SelectorRandom,
	}
	copy(p.profiles, profiles)
	return p
}

// WithSelector 设置选取策略。
func (p *Pool) WithSelector(s Selector) *Pool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.selector = s
	return p
}

// Append 追加自定义 Profile 到池中。
func (p *Pool) Append(profiles ...Profile) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles = append(p.profiles, profiles...)
}

// Reset 清空池并替换为新的 Profile 列表。
func (p *Pool) Reset(profiles []Profile) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.profiles = make([]Profile, len(profiles))
	copy(p.profiles, profiles)
	p.next = 0
}

// Pick 按当前 Selector 策略返回一个 Profile。
func (p *Pool) Pick() Profile {
	p.mu.RLock()
	profiles := p.profiles
	selector := p.selector
	p.mu.RUnlock()

	if len(profiles) == 0 {
		return defaultProfile()
	}

	switch selector {
	case SelectorRandom:
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(profiles))))
		if err != nil {
			return profiles[0]
		}
		return profiles[n.Int64()]
	case SelectorRoundRobin:
		p.mu.Lock()
		defer p.mu.Unlock()
		idx := p.next
		p.next = (p.next + 1) % len(profiles)
		return profiles[idx]
	default:
		return profiles[0]
	}
}

// Len 返回池中的 Profile 数量。
func (p *Pool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.profiles)
}

func defaultProfile() Profile {
	return Profile{
		UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		AcceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8",
		AcceptEncoding:  "gzip, deflate, br",
		SecCHUA:         "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"",
		SecCHUAMobile:   "?0",
		SecCHUAPlatform: "\"Windows\"",
	}
}

func defaultProfiles() []Profile {
	return []Profile{
		{
			UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"",
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: "\"Windows\"",
		},
		{
			UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage:  "en-US,en;q=0.9",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"",
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: "\"macOS\"",
		},
		{
			UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage:  "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3",
			AcceptEncoding:  "gzip, deflate, br",
		},
		{
			UserAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage:  "zh-CN,zh;q=0.9,en;q=0.8",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Microsoft Edge\";v=\"120\"",
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: "\"Windows\"",
		},
		{
			UserAgent:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			AcceptLanguage:  "en-US,en;q=0.9",
			AcceptEncoding:  "gzip, deflate, br",
		},
		{
			UserAgent:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
			Accept:          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
			AcceptLanguage:  "en-US,en;q=0.9",
			AcceptEncoding:  "gzip, deflate, br",
			SecCHUA:         "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"118\", \"Google Chrome\";v=\"118\"",
			SecCHUAMobile:   "?0",
			SecCHUAPlatform: "\"Linux\"",
		},
	}
}
