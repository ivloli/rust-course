// 第 04 章的 Go 平行版本。
// 每个例子都对应 ch04_structs_enums/src/main.rs 中的 Rust 写法，
// 注释从 Go 视角解释结构体、枚举与 Option 的建模差异。
package main

import (
	"fmt"
	"math"
)

// User 对应 Rust 的 struct User。
// Rust 用 #[derive(Debug)] 打印 {:?}；Go 通过实现 String() 让 %v 输出同样格式。
type User struct {
	Name   string
	Age    uint32
	Active bool
}

// newUser 对应 Rust 的关联函数 User::new（没有 self）。
func newUser(name string, age uint32) User {
	return User{Name: name, Age: age, Active: true}
}

// birthday 需要指针接收者：值接收者改的是副本。
// 这相当于 Rust 的 fn birthday(&mut self)。
func (u *User) birthday() {
	u.Age++
}

// isAdult 用值接收者，相当于 Rust 的 &self（只读）。
func (u User) isAdult() bool {
	return u.Age >= 18
}

func (u User) greeting() string {
	status := "离职"
	if u.Active {
		status = "在职"
	}
	return fmt.Sprintf("你好，我叫%s，%d 岁，%s", u.Name, u.Age, status)
}

// String 让 fmt 的 %v 输出与 Rust 的 {:?} 完全一致。
func (u User) String() string {
	return fmt.Sprintf("User { name: %q, age: %d, active: %t }", u.Name, u.Age, u.Active)
}

// Shape 对应 Rust 的 enum Shape：Go 用接口 + 具体类型模拟“带数据的枚举”。
// Rust 的 match 穷尽性检查，在 Go 里由 type switch 的 default 分支兜底（非强制）。
type Shape interface {
	Area() float64
	Name() string
}

type Circle struct{ Radius float64 }

func (c Circle) Area() float64 { return math.Pi * c.Radius * c.Radius }
func (c Circle) Name() string  { return "圆形" }

type Rect struct{ Width, Height float64 }

func (r Rect) Area() float64 { return r.Width * r.Height }
func (r Rect) Name() string  { return "矩形" }

type Triangle struct{ A, B, C float64 }

// Area 用海伦公式，对应 Rust 的 Shape::Triangle 分支。
func (t Triangle) Area() float64 {
	s := (t.A + t.B + t.C) / 2
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}

func (t Triangle) Name() string { return "三角形" }

// Command 对应 Rust 的 enum Command：
// 空变体 Quit、结构体变体 Move、元组变体 Write，在 Go 里是三个类型 + 一个标记接口。
type Command interface{ isCommand() }

type Quit struct{}

func (Quit) isCommand() {}

type Move struct{ X, Y int }

func (Move) isCommand() {}

type Write struct{ Text string }

func (Write) isCommand() {}

func main() {
	// 1. 结构体 + 方法
	user := newUser("小明", 17)
	fmt.Println(user.greeting())
	fmt.Printf("成年了吗？%t\n", user.isAdult())
	user.birthday() // Go 自动取地址：等价于 (&user).birthday()
	fmt.Printf("生日后: %v\n", user)

	// 2. 枚举可以携带不同形状的数据
	shapes := []Shape{
		Circle{Radius: 1.0},
		Rect{Width: 2.0, Height: 3.0},
		Triangle{A: 3.0, B: 4.0, C: 5.0},
	}
	for _, shape := range shapes {
		// Go 的 %6s 按 rune 数右对齐，对应 Rust 的 {:>6}；
		// %.2f 对应 {:.2}。
		fmt.Printf("%6s 面积 = %.2f\n", shape.Name(), shape.Area())
	}
	total := 0.0
	for _, shape := range shapes {
		total += shape.Area()
	}
	fmt.Printf("总面积 = %.2f\n", total)

	// 3. Option<T>：Go 没有 Option，用 (T, bool) 或 *T + nil 模拟。
	users := []User{newUser("Alice", 20), newUser("Bob", 30)}
	bob, found := findUser(users, "Bob")
	if found {
		fmt.Printf("找到 %s，%d 岁\n", bob.Name, bob.Age)
	} else {
		fmt.Println("没找到")
	}

	bobAge := 0
	if found {
		bobAge = int(bob.Age) // 对应 found.map(|u| u.age).unwrap_or(0)
	}
	fmt.Println("bob_age =", bobAge)

	// if let 只关心 Some 分支；Go 用 comma-ok 的 if。
	if carol, ok := findUser(users, "Carol"); ok {
		fmt.Println("Carol 存在:", carol.greeting())
	} else {
		fmt.Println("Carol 不存在")
	}

	// while let Some(top) = stack.pop() 在 Go 里写成 for len(stack) > 0。
	stack := []int{1, 2, 3}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		fmt.Printf("%d ", top)
	}
	fmt.Println()

	// 4. 枚举解构 + match → 接口 + type switch
	commands := []Command{
		Move{X: 10, Y: 20},
		Write{Text: "保存文件"},
		Quit{},
	}
	for _, command := range commands {
		switch c := command.(type) {
		case Quit:
			fmt.Println("退出")
		case Move:
			fmt.Printf("移动到 (%d, %d)\n", c.X, c.Y)
		case Write:
			fmt.Printf("写入: %s\n", c.Text)
		}
	}
}

// findUser 用 (User, bool) 模拟 Option<&User>。
// Rust 的 find 返回 Option，调用方必须处理 None；Go 里忘记检查 bool 也能编译。
func findUser(users []User, name string) (User, bool) {
	for _, user := range users {
		if user.Name == name {
			return user, true
		}
	}
	return User{}, false
}
