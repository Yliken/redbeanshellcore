// Package jitter 提供每次请求前的随机延迟中间件。
//
// 支持可配置的随机延迟范围、多种 backoff 策略，以及按操作类型分类配置。
package jitter

import (
	"context"
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/Yliken/redbeanshellcore/core"
)

// Strategy 定义抖动延时的计算方式。
type Strategy int

const (
	StrategyFixed    Strategy = iota // 固定范围均匀分布
	StrategyLinear                   // 线性增长（按尝试次数）
	StrategyExponential              // 指数退避 + 随机 jitter
)

// Options 配置 Jitter 中间件的行为。
type Options struct {
	MinDelay      time.Duration // 最小延迟（默认 500ms）
	MaxDelay      time.Duration // 最大延迟（默认 3000ms）
	Strategy      Strategy      // 退避策略
	PerOperation  bool          // 是否按操作名分别累计尝试次数
	SkipOnReadOps bool          // 是否对只读操作跳过 jitter
}

// DefaultOptions 返回默认的 Jitter 选项。
func DefaultOptions() Options {
	return Options{
		MinDelay:      500 * time.Millisecond,
		MaxDelay:      3000 * time.Millisecond,
		Strategy:      StrategyFixed,
		PerOperation:  false,
		SkipOnReadOps: true,
	}
}

type jitterState struct {
	opAttempts map[string]int // 每个操作名的尝试次数
	global     int            // 全局尝试次数
	mu         sync.Mutex
}

// Middleware 返回一个在每次请求前插入随机延迟的 core.Middleware。
func Middleware(opts Options) core.Middleware {
	if opts.MinDelay <= 0 {
		opts.MinDelay = 500 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 3000 * time.Millisecond
	}
	if opts.MinDelay > opts.MaxDelay {
		opts.MinDelay, opts.MaxDelay = opts.MaxDelay, opts.MinDelay
	}

	state := &jitterState{
		opAttempts: make(map[string]int),
	}

	return func(next core.Handler) core.Handler {
		return func(ctx context.Context, req *core.Request) (*core.Response, error) {
			if req == nil {
				return next(ctx, req)
			}

			skip := opts.SkipOnReadOps && req.Meta["risk_level"] == "read_only"
			if !skip {
				delay := computeDelay(&opts, state, req.Operation)
				time.Sleep(delay)
			}

			return next(ctx, req)
		}
	}
}

func computeDelay(opts *Options, state *jitterState, operation string) time.Duration {
	var attempt int

	if opts.PerOperation {
		state.mu.Lock()
		state.opAttempts[operation]++
		attempt = state.opAttempts[operation]
		state.mu.Unlock()
	} else {
		state.mu.Lock()
		state.global++
		attempt = state.global
		state.mu.Unlock()
	}

	baseRange := int64(opts.MaxDelay - opts.MinDelay)
	if baseRange <= 0 {
		baseRange = 1
	}

	switch opts.Strategy {
	case StrategyLinear:
		linearAdd := int64(opts.MinDelay) * int64(attempt) / 10
		base := int64(opts.MinDelay) + linearAdd
		if base > int64(opts.MaxDelay) {
			base = int64(opts.MaxDelay)
		}
		jitter := randJitter(base / 4)
		return time.Duration(base) + jitter

	case StrategyExponential:
		multiplier := int64(1) << uint(attempt)
		if multiplier > 64 {
			multiplier = 64
		}
		base := int64(opts.MinDelay) * multiplier
		if base > int64(opts.MaxDelay) {
			base = int64(opts.MaxDelay)
		}
		jitter := randJitter(base / 2)
		return time.Duration(base) + jitter

	default: // StrategyFixed
		jitter := randJitter(int64(opts.MaxDelay - opts.MinDelay))
		return opts.MinDelay + time.Duration(jitter)
	}
}

// randJitter 返回 [0, maxNanos) 的随机 time.Duration。
func randJitter(maxNanos int64) time.Duration {
	if maxNanos <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(maxNanos))
	if err != nil {
		return 0
	}
	return time.Duration(n.Int64())
}

// ResetAttempts 重置全局和按操作的尝试计数。
func ResetAttempts(state *jitterState) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.global = 0
	state.opAttempts = make(map[string]int)
}
