// Package php — obfuscate 提供函数名混淆辅助函数。
//
// 用于在生成的 PHP 代码中将敏感函数名（base64_decode、system 等）
// 拆成字符串拼接形式，避免传输中出现完整函数名字符串。
package php

import (
	"crypto/rand"
	"fmt"
)

// phpVar6 生成一个以字母开头的随机 PHP 变量名称。
func phpVar6() string {
	return "x" + randomVar(5)
}

// obfuscatedFunc 生成 PHP 函数名的混淆赋值代码。
// 将函数名拆为两段字符串拼接赋值给随机变量，避免在传输中出现完整函数名字符串。
// 返回 PHP 赋值语句（如 $a='base6'.'4_decode';）和变量表达式（如 $a）。
func obfuscatedFunc(name string) (setup string, ref string) {
	v := phpVar6()
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
	return fmt.Sprintf("$%s='%s'.'%s'", v, part1, part2), fmt.Sprintf("$%s", v)
}
