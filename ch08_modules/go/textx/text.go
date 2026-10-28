// Package textx 对应 Rust 的 text 模块（src/text.rs）。
package textx

import "strings"

// WordCount 对应 Rust 的 pub fn word_count。
// strings.Fields 与 Rust 的 split_whitespace 一样按 Unicode 空白切分，
// 并自动忽略首尾和连续空白，所以不需要额外过滤空串。
func WordCount(text string) int {
	return len(strings.Fields(text))
}

// Shout 对应 Rust 的 pub fn shout。
// Rust 的 to_uppercase 返回新 String；Go 的 string 不可变，ToUpper 也返回新串。
func Shout(text string) string {
	return strings.ToUpper(text)
}
