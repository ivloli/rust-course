// 第 01 章的 Go 平行版本。
// 每个例子都对应 ch01_variables/src/main.rs 中的 Rust 写法，
// 注释从 Go 视角解释两者的差异。
package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func main() {
	// 1. Rust 变量默认不可变（let x = 5 之后不能再赋值，除非加 mut）。
	// Go 没有 mut：变量默认可变，只有 const 不可变。
	x := 5
	fmt.Println("x =", x)

	// 2. Rust 需要 let mut y；Go 直接改，语法上没有任何标记。
	y := 10
	y++
	fmt.Println("y =", y)

	// 3. Shadowing：Rust 用同名 let 创建新变量，甚至可以换类型。
	// Go 的 := 只有在“新作用域”里才是新变量；同一作用域重复 := 是编译错误
	// （no new variables on left side of :=），所以这里用花括号开一个内层作用域。
	{
		x := x + 1 // 内层 x 遮蔽外层；出了大括号，外层 x 仍然是 5
		message := fmt.Sprintf("现在 x 变成了字符串: %d", x)
		fmt.Println(message)
	}

	// 4. 类型标注与解析：Rust 的 "42".parse::<u32>() 返回 Result；
	// Go 的 strconv.Atoi 返回 (int, error)，错误必须显式检查。
	guess, err := strconv.Atoi("42")
	if err != nil {
		panic(err) // 对应 Rust 的 .expect("输入不是数字")
	}
	fmt.Println("guess =", guess)

	// 5. 标量类型。Rust 的 char 是 4 字节 Unicode 标量值；
	// Go 的 rune 是 int32 的别名，概念相同。
	// 注意 fmt 对 rune 用 %c 才打印字符，用 %v 会打印码点数字。
	var integer int64 = 1_000_000
	var float float64 = 2.5
	isRustFun := true
	var letter rune = '中'
	fmt.Printf("%d %g %t %c\n", integer, float, isRustFun, letter)

	// 6. 整数溢出：Rust 的 debug 构建会 panic（attempt to add with overflow），
	// release 构建回绕；Go 在任何构建下都静默回绕，不会 panic。
	maxU8 := uint8(math.MaxUint8)
	fmt.Println("u8::MAX =", maxU8)
	wrapped := maxU8 + 1 // 255 + 1 静默变成 0：无编译错误、无运行时 panic
	_ = wrapped
	// 对比被注释掉的 Rust 代码：let overflow = max_u8 + 1; // debug 下 panic

	// 7. 复合类型。Rust 的元组在 Go 里没有直接对应：
	// 用字段有名字的匿名结构体，或几个独立变量。
	// Go 也没有 .0 语法，输出里的 person.0 只是照抄 Rust 的文案。
	person := struct {
		name   string
		age    uint32
		active bool
	}{"小明", 28, true}
	name, age, active := person.name, person.age, person.active
	fmt.Printf("%s %d 岁，在职: %t，也可以写 person.0 = %s\n", name, age, active, person.name)

	// Rust 的数组 [i32; 5] 长度是类型的一部分；Go 的 [5]int 也是定长数组，
	// 但更常用的是切片 []int（长度可变的视图）。
	scores := [5]int{90, 85, 77, 60, 99}
	zeros := [3]uint8{}
	fmt.Printf("scores = %s, 长度 %d, 第一项 %d\n", formatInts(scores[:]), len(scores), scores[0])
	fmt.Printf("zeros = %s\n", formatInts(zeros[:]))

	// 8. Rust 区分 &str（借用的 UTF-8 视图）与 String（可增长的堆内存）；
	// Go 只有一个 string 类型，不可变，赋值和传参都共享同一份只读数据。
	literal := "字面量"
	owned := "堆上的 String"
	combined := fmt.Sprintf("%s + %s", literal, owned)
	fmt.Println(combined)

	// 9. 常量在编译期确定；Go 的 const 同样必须有类型（或由字面量推导）。
	// 命名习惯不同：Rust 用 SCREAMING_SNAKE_CASE，Go 用驼峰。
	const maxPoints uint32 = 100_000
	fmt.Println("MAX_POINTS =", maxPoints)

	// 10. 显式转换。Rust 用 as，Go 用 T(v)，两者都不会隐式发生。
	big := int64(300)
	small := int32(big)
	// 注意：常量转换 uint8(300) 是编译错误（常量溢出），
	// 所以先放进变量再截断，行为才和 Rust 的 300u16 as u8 一致。
	var medium uint16 = 300
	truncated := uint8(medium) // 300 % 256 = 44
	fmt.Printf("big = %d, small = %d, truncated = %d\n", big, small, truncated)
}

// formatInts 模拟 Rust 的 {:?}：输出 [1, 2, 3] 而不是 Go 默认的 [1 2 3]。
func formatInts[T ~int | ~uint8](values []T) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprint(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
