// 第 02 章的 Go 平行版本。
// 每个例子都对应 ch02_control_flow/src/main.rs 中的 Rust 写法，
// 注释从 Go 视角解释两者的差异。
package main

import (
	"fmt"
	"strconv"
)

func main() {
	// Rust 的 if 是表达式，可以直接给变量赋值；Go 的 if 是语句，
	// 只能先声明变量，再在分支里赋值。
	fmt.Println("sign(-3) =", sign(-3))
	fmt.Println("gcd(48, 18) =", gcd(48, 18))
	fmt.Println("fib(10) =", fib(10))

	age := 20
	category := "成年"
	if age < 18 {
		category = "未成年"
	}
	fmt.Printf("%d 岁：%s\n", age, category)

	// Rust 的 loop { ... break value } 在 Go 里没有对应语法：
	// 先声明结果变量，在 for 里 break，循环结束后再使用它。
	counter := 0
	doubled := 0
	for {
		counter++
		if counter == 5 {
			doubled = counter * 2
			break
		}
	}
	fmt.Printf("counter = %d, doubled = %d\n", counter, doubled)

	// Rust 的 while cond 对应 Go 的 for cond（Go 只有 for 一种循环关键字）。
	countdown := 3
	for countdown > 0 {
		fmt.Printf("%d ", countdown)
		countdown--
	}
	fmt.Println("发射！")

	// for 遍历范围：Go 用经典三段式 for，闭区间写成 i <= 3。
	for i := 1; i <= 3; i++ {
		fmt.Printf("for: %d ", i)
	}
	fmt.Println()

	// Rust 的 iter().enumerate() 就是 Go 的 for i, v := range。
	for index, score := range []int{88, 92, 60} {
		fmt.Printf("第 %d 位同学: %d\n", index, score)
	}

	// 带标签的循环：Go 的 label + break label 对应 Rust 的 'outer + break 'outer，
	// 两者都能一次跳出多层嵌套。
outer:
	for i := 0; i < 5; i++ {
		for j := 0; j < 5; j++ {
			if i*j == 6 {
				fmt.Printf("找到 i=%d, j=%d\n", i, j)
				break outer
			}
		}
	}

	// Rust 的 match 是表达式，有穷尽性检查，还支持守卫和区间模式；
	// Go 的 switch 是语句，没有穷尽性要求，守卫要写成 case 条件或 if 链。
	temp := -5
	advice := "有点热"
	switch {
	case temp < 0:
		advice = "穿厚点"
	case temp == 0:
		advice = "刚刚好"
	case temp >= 1 && temp <= 20:
		advice = "舒适"
	}
	fmt.Printf("温度 %d：%s\n", temp, advice)

	// Rust 用 match 元组模式 (n%3, n%5)；Go 的 switch 不能匹配元组，
	// 用 switch true + 条件组合来替代。
	for n := 1; n <= 15; n++ {
		fmt.Printf("%d -> %s\n", n, fizzbuzz(n))
	}

	// Rust 可以 match 元组并绑定变量；Go 用结构体或两个变量 + switch。
	command := struct {
		action string
		step   int
	}{"move", 10}
	switch command.action {
	case "move":
		fmt.Printf("向 step=%d 移动\n", command.step)
	case "jump":
		fmt.Printf("跳 %d 米\n", command.step)
	default:
		fmt.Printf("未知命令 %s\n", command.action)
	}
}

// sign 返回 string。Rust 的 &'static str 是借用的静态字符串，
// Go 的 string 本身就是不可变的只读 header，没有额外生命周期概念。
func sign(x int) string {
	if x < 0 {
		return "负数"
	} else if x == 0 {
		return "零"
	}
	return "正数"
}

// gcd：Go 的参数天然可重新赋值，不需要 Rust 的 mut a, mut b。
func gcd(a, b uint64) uint64 {
	for b != 0 {
		remainder := a % b
		a = b
		b = remainder
	}
	return a
}

// fib：Rust 的 for _ in 0..n 对应 Go 的 for i := 0; i < n; i++。
func fib(n uint32) uint64 {
	a, b := uint64(0), uint64(1)
	for i := uint32(0); i < n; i++ {
		next := a + b
		a = b
		b = next
	}
	return a
}

// fizzbuzz：Rust 的 match (n%3, n%5) 是元组模式匹配；
// Go 没有元组，用 switch true 把两个条件写在一起。
func fizzbuzz(n int) string {
	switch {
	case n%3 == 0 && n%5 == 0:
		return "FizzBuzz"
	case n%3 == 0:
		return "Fizz"
	case n%5 == 0:
		return "Buzz"
	default:
		return strconv.Itoa(n)
	}
}
