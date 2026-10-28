// 第 08 章的 Go 平行版本。
// 运行：cd ch08_modules/go && go run .
// 测试：cd ch08_modules/go && go test ./...
package main

import (
	"fmt"
	"strconv"
	"strings"

	"ch08_go/mathx"
	"ch08_go/textx"
)

// 导入路径 ch08_go/mathx 对应 go.mod 里的 module ch08_go 加上目录名。
// Rust 的 use ch08_modules::{gcd, mean, word_count} 依赖 lib.rs 的 pub use；
// Go 没有重导出（re-export），通常直接导入具体包，或在包内写包装函数。

func main() {
	fmt.Printf("gcd(48, 18) = %d\n", mathx.Gcd(48, 18))

	// Rust: mean(&[2.0, 4.0]) 返回 Option<f64>；
	// Go: (float64, bool)，用 helper 打印成 Some(3.0) / None。
	inputs := []float64{2.0, 4.0}
	mean, ok := mathx.Mean(inputs)
	fmt.Printf("mean(%s) = %s\n", formatFloats(inputs), formatOptionFloat(mean, ok))

	// Rust 的 word_count 是 usize；Go 的 int 在 64 位平台是 64 位。
	fmt.Printf("word_count(%q) = %d\n", "a b c", textx.WordCount("a b c"))

	// Rust 可以走完整路径 ch08_modules::math::stats::mean；
	// Go 的完整路径是 包导入路径 + 导出名：mathx.Mean。
	// 因为 stats 被摊平到 mathx 包内，这里不再有 stats 这一层。
	nested, nestedOK := mathx.Mean([]float64{1.0, 2.0, 3.0})
	fmt.Printf("嵌套路径也能访问: %s\n", formatOptionFloat(nested, nestedOK))
}

// formatOptionFloat 模拟 Rust 的 {:?} 对 Option<f64> 的输出：
// Some(3.0) / None。Go 的 %v 会打印成 3，所以要自己补 .0。
func formatOptionFloat(value float64, ok bool) string {
	if !ok {
		return "None"
	}
	return "Some(" + formatFloat(value) + ")"
}

// formatFloat 让 3.0 输出 "3.0"，同时保留 2.5 这类小数。
func formatFloat(value float64) string {
	text := strconv.FormatFloat(value, 'g', -1, 64)
	if !strings.ContainsAny(text, ".eE") {
		text += ".0"
	}
	return text
}

// formatFloats 模拟 Rust 的 {:?} 对 &[f64] 的输出：[2.0, 4.0]。
func formatFloats(values []float64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = formatFloat(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
