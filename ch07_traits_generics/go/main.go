// 第 07 章的 Go 平行版本。
// 每个例子都对应 ch07_traits_generics/src/main.rs。
// 运行：cd ch07_traits_generics/go && go run .
package main

import (
	"cmp"
	"fmt"
	"strconv"
	"strings"
)

// Rust 的 trait Summary ↔ Go 的 interface Summary。
// 接口只声明方法签名，不包含字段，也不能有默认实现。
type Summary interface {
	SummarizeAuthor() string
	Summarize() string
}

// Rust 的 trait 默认方法在 Go 里没有直接对应物。
// 常见替代方案是辅助函数：实现者显式调用它，效果等同于「不覆盖默认方法」。
func defaultSummary(author string) string {
	return fmt.Sprintf("（阅读更多来自 %s 的内容…）", author)
}

type Article struct {
	Title  string
	Author string
}

// 值接收者让 Article 和 *Article 都满足接口。
// Rust 里是 impl Summary for Article。
func (a Article) SummarizeAuthor() string {
	return a.Author
}

func (a Article) Summarize() string {
	// Rust: format!("{} — {}", self.title, self.summarize_author())
	return a.Title + " — " + a.SummarizeAuthor()
}

type Tweet struct {
	Username string
	Body     string
}

func (t Tweet) SummarizeAuthor() string {
	return "@" + t.Username
}

func (t Tweet) Summarize() string {
	return t.SummarizeAuthor() + ": " + t.Body
}

// 对应 Rust 的 impl fmt::Display for Tweet。
// Go 的 fmt.Stringer 接口就是 String() string，相当于 Display 的简化版。
func (t Tweet) String() string {
	return t.Summarize()
}

// 1. impl Trait 参数 ↔ 接口参数。
// Rust 的 &impl Summary 是静态分发；Go 传接口是动态分发（itab + 间接调用）。
func notify(item Summary) string {
	return "Breaking! " + item.Summarize()
}

// displaySummary 模拟 Rust 的多个 trait bound：Summary + Display。
// Go 用接口嵌入组合方法集，对应 fmt.Stringer。
type displaySummary interface {
	Summary
	fmt.Stringer
}

// 2. 多约束。
// Rust: fn notify_with_display<T: Summary + Display>(item: &T)
// Go 写成组合接口；%s 会调用 String()，等价于 Rust 的 Display。
func notifyWithDisplay(item displaySummary) string {
	return fmt.Sprintf("Breaking! %s | %s", item, item.Summarize())
}

// 3. 泛型 + trait bound ↔ 类型参数 + 约束。
// Rust: fn summarize_all<T: Summary>(items: &[T]) -> Vec<String>
// Go 的类型参数约束直接写接口名。
func summarizeAll[T Summary](items []T) []string {
	summaries := make([]string, len(items))
	for i, item := range items {
		summaries[i] = item.Summarize()
	}
	return summaries
}

// 4. 返回 impl Trait ↔ 返回接口。
// Rust 的 -> impl Summary 对调用方隐藏具体类型，同时保留静态分发；
// Go 只能返回接口值，具体类型在运行时装箱，没有「隐藏但零开销」的保证。
func makeTweet() Summary {
	return Tweet{
		Username: "rustlang",
		Body:     "Rust 1.99 已发布",
	}
}

// 5. 泛型函数。
// Rust: fn largest<T: PartialOrd + Copy>(list: &[T]) -> T
// Go: cmp.Ordered 已经覆盖所有可比较大小的内置类型（int/float/string 等），
// 而且 Go 赋值天然复制，不需要额外的 Copy 约束。
func largest[T cmp.Ordered](list []T) T {
	largest := list[0]
	for _, item := range list {
		if item > largest {
			largest = item
		}
	}
	return largest
}

// 6. 自定义 trait 做动态分发 ↔ 接口 + 接口切片。
type Area interface {
	Area() float64
}

type Square struct {
	Side float64
}

func (s Square) Area() float64 {
	return s.Side * s.Side
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	// Rust: std::f64::consts::PI * self.0 * self.0
	return 3.141592653589793 * c.Radius * c.Radius
}

// 7. From / Into / Display ↔ 构造函数 + 类型转换 + String()。
// Rust 的 newtype 在 Go 里就是 defined type。
type UserID uint64

// 对应 impl From<u32> for UserId。
// Go 没有 From/Into trait，通常写 NewXxx 构造函数。
func NewUserID(value uint32) UserID {
	return UserID(value)
}

// 对应 impl fmt::Display for UserId。
func (id UserID) String() string {
	return fmt.Sprintf("user#%d", uint64(id))
}

// formatStrings 模拟 Rust 的 {:?} 字符串切片输出：["a", "b"]。
func formatStrings(values []string) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Quote(value)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func main() {
	// Go 的结构体字面量必须写字段名或按位置写全部字段；
	// 没有 Rust 那种 `Article { title: .., ..Default::default() }` 的简写。
	article := Article{Title: "Rust 所有权入门", Author: "小明"}
	tweet := Tweet{Username: "chuan", Body: "今天在学 trait"}

	fmt.Println(article.Summarize())
	fmt.Println(tweet.Summarize())
	fmt.Println(notify(tweet))
	fmt.Println(notifyWithDisplay(tweet))

	// Rust: let summaries = summarize_all(&[make_tweet()]);
	// Go 这里用 []Summary 装接口值。
	summaries := summarizeAll([]Summary{makeTweet()})
	fmt.Println(formatStrings(summaries))

	// 动态分发：Rust 的 Vec<Box<dyn Summary>> ↔ Go 的 []Summary。
	// Go 没有 trait object 的结构体字面量简写，必须显式创建值再放进切片。
	items := []Summary{
		Article{Title: "动态分发", Author: "A"},
		Tweet{Username: "B", Body: "在运行时决定调用"},
	}
	for _, item := range items {
		fmt.Println("dyn ->", item.Summarize())
	}

	// Rust 的 Vec<Box<dyn Area>> ↔ Go 的 []Area。
	shapes := []Area{Square{Side: 2.0}, Circle{Radius: 1.0}}
	total := 0.0
	for _, shape := range shapes {
		total += shape.Area()
	}
	// Rust 的 {total:.2} 对应 Go 的 %.2f：两边都保留两位小数并四舍五入。
	fmt.Printf("总面积 = %.2f\n", total)

	fmt.Printf("largest(&[3, 9, 2]) = %d\n", largest([]int{3, 9, 2}))
	// Rust 的 char 是 Unicode 标量；Go 的 rune 是 int32 别名，输出用 %c。
	fmt.Printf("largest(&['a', 'z', 'm']) = %c\n", largest([]rune{'a', 'z', 'm'}))

	// Rust: let id = UserId::from(42u32); let other: UserId = 7u32.into();
	// Go: 构造函数 + 显式类型转换，没有自动的 Into。
	id := NewUserID(42)
	other := UserID(7)
	fmt.Printf("id = %s, other = %s\n", id, other)
}
