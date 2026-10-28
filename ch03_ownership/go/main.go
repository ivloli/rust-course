// 第 03 章的 Go 平行版本。
// 每个例子都对应 ch03_ownership/src/main.rs 中的 Rust 写法，
// 注释从 Go 视角解释两者（所有权 / 借用 / 生命周期）的差异。
package main

import (
	"fmt"
	"strings"
)

func main() {
	// 1. Go 没有所有权和移动语义：赋值只是复制 string header（指针 + 长度），
	// s1 和 s2 都可用，底层字节共享。Rust 里 let s2 = s1 之后 s1 已失效（E0382）。
	s1 := "hello"
	s2 := s1
	fmt.Println("s2 =", s2)

	// 2. Rust 需要显式 clone() 才能保留两份可用值；Go 里 s2 一直可用，
	// 不存在“移动后失效”，也就没有 clone 的概念。
	s3 := s2
	fmt.Printf("s3 = %s, s2 仍然可用 = %s\n", s3, s2)

	// 3. Rust 的 Copy 类型（i32）赋值是按位复制；Go 的 int 同样按值复制。
	n1 := 5
	n2 := n1
	fmt.Printf("n1 = %d, n2 = %d\n", n1, n2)

	// 4. Rust 传 &str 借用以避免复制；Go 传 string 只复制 header，不复制内容，
	// 而且 string 不可变，函数内部无法修改它。
	length := calcLen(s3)
	fmt.Printf("s3 = %s, 长度 = %d\n", s3, length)

	// 5. Rust 的 &mut String 是排他借用；Go 用指针 *string 表达“可修改”，
	// 但没有任何编译期检查阻止你同时持有多个指针。
	s4 := "hello"
	appendWorld(&s4)
	fmt.Println("s4 =", s4)

	// 下面这段在 Go 里完全合法，两个指针别名同一块数据：
	//   r1 := &s4
	//   r2 := &s4 // Rust 会报 E0499：cannot borrow as mutable more than once
	//   *r1 += "!"
	//   fmt.Println(*r1, *r2) // 两个指针都能看到修改
	// Rust 靠借用检查器在编译期禁止这种别名，Go 把责任交给程序员。

	// 6. Rust 的 &str 切片借用一段连续内存；Go 的 s[a:b] 同样共享底层数组。
	// firstWord 返回的是 string header，仍指向原串的只读字节。
	first := firstWord(s4)
	fmt.Println("第一个单词:", first)

	// 7. Rust 的 longest<'a> 需要生命周期标注来约束返回引用；
	// Go 没有生命周期，返回 string 只是复制 header，编译器不追踪它引用谁。
	longer := longest("短", "一个更长的字符串")
	fmt.Println("更长的字符串:", longer)

	// 8. Rust 把 String 传进函数会移动所有权，再返回“还回来”；
	// Go 没有所有权，传 string 就是按值复制 header，原变量照样可用。
	s5 := takesAndGivesBack(s4)
	fmt.Printf("s5 = %s\n", s5)

	// 返回局部变量的指针在 Go 里是安全的：逃逸分析会把它分配到堆上。
	// Rust 会拒绝这种悬垂引用（E0106 / E0597），这是所有权模型的核心收益。
	_ = *dangle()
}

func calcLen(s string) int {
	return len(s) // 字节数，和 Rust 的 str::len() 一致
}

// appendWorld 用 *string 模拟 Rust 的 &mut String：调用方必须显式取地址。
func appendWorld(s *string) {
	*s += ", world"
}

// firstWord 对应 Rust 的 s.split_whitespace().next().unwrap_or("")。
func firstWord(s string) string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// longest 在 Rust 里必须写成 longest<'a>(a: &'a str, b: &'a str) -> &'a str，
// 向编译器承诺返回值的生命周期不超过两个入参。Go 不需要也无法表达这个约束。
func longest(a, b string) string {
	if len(a) >= len(b) {
		return a
	}
	return b
}

func takesAndGivesBack(s string) string {
	return s
}

// dangle 在 Go 里合法：s 会逃逸到堆上，返回的指针始终有效。
// 对应的 Rust 代码 fn dangle() -> &String { &String::from("x") } 无法编译。
func dangle() *string {
	s := "x"
	return &s
}
