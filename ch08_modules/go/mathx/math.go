// Package mathx 对应 Rust 的 math 模块（src/math.rs）。
//
// Go 的包由目录决定，一个目录只能有一个包名；
// 因此 mathx 目录下的 math.go 和 stats.go 都属于同一个包。
package mathx

// Gcd 对应 Rust 的 pub fn gcd。
// Go 用首字母大写表示导出（相当于 Rust 的 pub），小写则是包内私有。
func Gcd(a, b uint64) uint64 {
	// Rust 用 let (mut a, mut b) = (a, b); Go 的参数本身就可以重新赋值。
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
