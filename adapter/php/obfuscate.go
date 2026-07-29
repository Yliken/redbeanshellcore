// Package php — obfuscate 提供函数名混淆辅助函数。
//
// 用于在生成的 PHP 代码中将敏感函数名（base64_decode、system 等）
// 转成不可直接识别的形式，避免传输中出现函数名字符串。
package php

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// phpVar6 生成一个以字母开头的随机 PHP 变量名称。
func phpVar6() string {
	return "x" + randomVar(5)
}

type obfuscationType int

const (
	obfStrConcat  obfuscationType = iota // 字符串拼接分裂
	obfStrRev                             // strrev() 反转
	obfSubstr                             // substr() 从长字符串中截取（嵌套 strrev）
	obfChrSeq                             // chr() 序列拼接
	obfMultiConcat                        // 分裂为 3+ 段拼接
)

const numStrategies = 5

// randomStrategy 随机选择一种混淆策略。
func randomStrategy() obfuscationType {
	n, err := rand.Int(rand.Reader, big.NewInt(numStrategies))
	if err != nil {
		return obfStrConcat
	}
	return obfuscationType(n.Int64())
}

// obfuscatedFunc 生成 PHP 函数名的混淆赋值代码。
// 使用多种策略随机选择，避免在传输中出现可识别的函数名特征。
// 返回 PHP 赋值语句和变量引用表达式。
func obfuscatedFunc(name string) (setup string, ref string) {
	v := phpVar6()
	switch randomStrategy() {
	case obfStrConcat:
		setup, ref = obfuscateStrConcat(name, v)
	case obfStrRev:
		setup, ref = obfuscateStrRev(name, v)
	case obfSubstr:
		setup, ref = obfuscateSubstr(name, v)
	case obfChrSeq:
		setup, ref = obfuscateChrSeq(name, v)
	case obfMultiConcat:
		setup, ref = obfuscateMultiConcat(name, v)
	default:
		setup, ref = obfuscateStrConcat(name, v)
	}
	return setup, ref
}

// obfuscateStrConcat 将函数名拆为两段拼接： $x='base6' . '4_decode'
func obfuscateStrConcat(name, varName string) (string, string) {
	n := len(name)
	split := 1
	if n > 3 {
		buf := make([]byte, 1)
		_, _ = rand.Read(buf)
		split = 1 + int(buf[0])%(n-1)
		if split >= n {
			split = n - 1
		}
	}
	part1 := name[:split]
	part2 := name[split:]
	return fmt.Sprintf("$%s='%s' . '%s'", varName, part1, part2), fmt.Sprintf("$%s", varName)
}

// obfuscateStrRev 使用 strrev() 反转名字： $x=strrev('edoced_46esab')
func obfuscateStrRev(name, varName string) (string, string) {
	reversed := reverseString(name)
	return fmt.Sprintf("$%s=strrev('%s')", varName, reversed), fmt.Sprintf("$%s", varName)
}

// obfuscateSubstr 嵌套使用 strrev() 和 substr()：
//   $x=strrev(substr('a3f8edoced_46esabb1e9',4,13))
func obfuscateSubstr(name, varName string) (string, string) {
	reversed := reverseString(name)
	prefix := randomHex(4)
	suffix := randomHex(4)
	fullStr := prefix + reversed + suffix
	start := len(prefix)
	length := len(name)
	return fmt.Sprintf("$%s=strrev(substr('%s',%d,%d))", varName, fullStr, start, length), fmt.Sprintf("$%s", varName)
}

// obfuscateChrSeq 使用 chr() 序列拼接： $x=chr(98).chr(97).chr(115)...
func obfuscateChrSeq(name, varName string) (string, string) {
	var parts []string
	for _, c := range []byte(name) {
		parts = append(parts, fmt.Sprintf("chr(%d)", c))
	}
	return fmt.Sprintf("$%s=%s", varName, strings.Join(parts, ".")), fmt.Sprintf("$%s", varName)
}

// obfuscateMultiConcat 将函数名拆为 3~5 段拼接。
func obfuscateMultiConcat(name, varName string) (string, string) {
	n := len(name)
	if n < 4 {
		return obfuscateStrConcat(name, varName)
	}
	segCount := 3
	if n > 5 {
		buf := make([]byte, 1)
		_, _ = rand.Read(buf)
		extra := int(buf[0]) % 3
		segCount = 3 + extra
		if segCount > n {
			segCount = n
		}
	}
	cutPoints := make(map[int]bool)
	for len(cutPoints) < segCount-1 {
		buf := make([]byte, 1)
		_, _ = rand.Read(buf)
		pt := 1 + int(buf[0])%(n-1)
		cutPoints[pt] = true
	}
	sorted := make([]int, 0, segCount)
	for p := range cutPoints {
		sorted = append(sorted, p)
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	var segments []string
	prev := 0
	for _, pt := range sorted {
		segments = append(segments, name[prev:pt])
		prev = pt
	}
	segments = append(segments, name[prev:])
	expr := "'" + strings.Join(segments, "' . '") + "'"
	return fmt.Sprintf("$%s=%s", varName, expr), fmt.Sprintf("$%s", varName)
}

// reverseString 反转字符串。
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// obfuscatedFuncRef 生成只返回函数引用（不包含 setup 代码）的便捷函数。
// 用于那些只需要在 PHP global 声明中引用的函数名。
func obfuscatedFuncRef(name string) string {
	_, ref := obfuscatedFunc(name)
	return ref
}

// obfuscatedFuncSubstr 始终使用 strrev+substr 方式的函数名混淆。
func obfuscatedFuncSubstr(name string) (setup string, ref string) {
	v := phpVar6()
	return obfuscateSubstr(name, v)
}

// obfuscatedFuncRefSubstr 只返回 strrev+substr 混淆方式的函数引用。
func obfuscatedFuncRefSubstr(name string) string {
	_, ref := obfuscatedFuncSubstr(name)
	return ref
}

