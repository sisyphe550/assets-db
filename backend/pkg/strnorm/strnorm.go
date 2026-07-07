// Package strnorm 字符串规范化工具
// 用于盘点比对时的位置名称标准化
package strnorm

import (
	"strings"
	"unicode"
)

// Location 规范化位置字符串
// trim 首尾空格 + 全角空格 → 半角 + 连续空格合并为单空格
func Location(s string) string {
	s = strings.TrimSpace(s)
	// 全角空格 (U+3000) → 半角空格
	s = strings.ReplaceAll(s, "\u3000", " ")
	// 全角数字/字母 → 半角
	s = toHalfWidth(s)
	// 合并连续空格
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' {
			if !inSpace {
				b.WriteRune(' ')
				inSpace = true
			}
		} else {
			b.WriteRune(r)
			inSpace = false
		}
	}
	return b.String()
}

func toHalfWidth(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0xFF01 && r <= 0xFF5E {
			b.WriteRune(r - 0xFEE0)
		} else if r == 0x3000 {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Equal 比较两个位置是否等价（规范化后相等）
func Equal(a, b string) bool {
	return Location(a) == Location(b)
}

// Normalize 通用规范化：trim + 全角转半角
func Normalize(s string) string {
	return strings.TrimSpace(toHalfWidth(s))
}

// IsWhitespaceOnly 判断是否仅包含空白字符
func IsWhitespaceOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
