package mathx

import "testing"

// Go 的测试放在同目录的 _test.go 文件中，函数名必须是 TestXxx。
// 这对应 Rust 的 #[cfg(test)] mod tests：
//   - 测试代码不会进入正式二进制（go build 会忽略 _test.go）；
//   - 同包测试可以访问未导出（小写）的标识符。

func TestGcdWorks(t *testing.T) {
	// 对应 Rust 的 assert_eq!(gcd(48, 18), 6);
	if got := Gcd(48, 18); got != 6 {
		t.Fatalf("Gcd(48, 18) = %d, want 6", got)
	}
	if got := Gcd(17, 13); got != 1 {
		t.Fatalf("Gcd(17, 13) = %d, want 1", got)
	}
}

func TestGcdWithZero(t *testing.T) {
	if got := Gcd(0, 5); got != 5 {
		t.Fatalf("Gcd(0, 5) = %d, want 5", got)
	}
}

func TestMeanWorks(t *testing.T) {
	got, ok := Mean([]float64{2.0, 4.0})
	if !ok || got != 3.0 {
		t.Fatalf("Mean([2, 4]) = (%v, %v), want (3, true)", got, ok)
	}
}

func TestMeanOfEmptyIsNone(t *testing.T) {
	// Rust: assert_eq!(stats::mean(&[]), None);
	if got, ok := Mean(nil); ok {
		t.Fatalf("Mean(nil) = (%v, true), want (0, false)", got)
	}
}
