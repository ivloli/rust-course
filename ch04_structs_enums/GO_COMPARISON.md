# 第 04 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`，**两边输出完全一致**。

```bash
# Rust
cd ~/rust-course && cargo run -p ch04_structs_enums

# Go（需要 Go 1.22+）
cd ~/rust-course/ch04_structs_enums/go && go run .
```

## 先记住三个总差异

1. **数据与方法**：Rust 用 `struct` + `impl`，方法显式区分 `self` / `&self` / `&mut self`；Go 用 `struct` + 接收者，值接收者拿副本，指针接收者才能改原值。
2. **枚举**：Rust 的 `enum` 是可携带不同数据的和类型（tagged union）；Go 没有 enum 关键字，用 `const` + `iota` 表达简单枚举，用「接口 + 具体类型 + type switch」表达带数据的变体。
3. **缺失值**：Rust 用 `Option<T>` 强制处理「没有」；Go 用 `(T, bool)`、`*T` + `nil`，编译期不强制检查。

## 例 1：结构体与方法
Rust：
```rust
#[derive(Debug, Clone)]
struct User { name: String, age: u32, active: bool }

impl User {
    fn new(name: &str, age: u32) -> Self { /* ... */ }
    fn birthday(&mut self) { self.age += 1; }
    fn is_adult(&self) -> bool { self.age >= 18 }
}
let mut user = User::new("小明", 17);
user.birthday();
println!("生日后: {:?}", user);
```
Go：
```go
type User struct {
    Name   string
    Age    uint32
    Active bool
}
func newUser(name string, age uint32) User { /* ... */ }
func (u *User) birthday() { u.Age++ } // 指针接收者才能改原值
func (u User) isAdult() bool { return u.Age >= 18 }
user := newUser("小明", 17)
user.birthday() // 编译器自动取地址：(&user).birthday()
fmt.Printf("生日后: %v\n", user) // User 实现了 String()，格式与 {:?} 一致
```
- `birthday` 必须用指针接收者；值接收者改的是副本（类似 Rust 里只拿 `&self` 却想修改）。Go 调用时会自动取地址，所以 `user.birthday()` 不用写 `(&user)`。
- Go 没有构造函数和 `#[derive(Debug)]`：用 `newUser` 工厂函数，用 `String()` 方法定制 `%v` 输出。
- Rust 的 `new` 是关联函数（无 `self`），Go 直接写包级函数；Rust 字段默认私有、Go 字段首字母大写才导出。

## 例 2：枚举携带不同数据
Rust：
```rust
enum Shape {
    Circle { radius: f64 },
    Rect { width: f64, height: f64 },
    Triangle { a: f64, b: f64, c: f64 },
}
impl Shape {
    fn area(&self) -> f64 {
        match self {
            Shape::Circle { radius } => PI * radius * radius,
            Shape::Rect { width, height } => width * height,
            Shape::Triangle { a, b, c } => { /* 海伦公式 */ }
        }
    }
}
println!("{:>6} 面积 = {:.2}", shape.name(), shape.area());
```
Go：
```go
type Shape interface {
    Area() float64
    Name() string
}
type Circle struct{ Radius float64 }
func (c Circle) Area() float64 { return math.Pi * c.Radius * c.Radius }
type Rect struct{ Width, Height float64 }
func (r Rect) Area() float64 { return r.Width * r.Height }

for _, shape := range []Shape{Circle{1}, Rect{2, 3}, Triangle{3, 4, 5}} {
    fmt.Printf("%6s 面积 = %.2f\n", shape.Name(), shape.Area())
}
```
- Rust 的 `match` 对枚举做穷尽性检查，新增变体必须补分支；Go 的接口是开放的，加一种形状只要实现方法，但 `type switch` 没有穷尽性检查。
- 输出对齐：Rust 的 `{:>6}` 按字符右对齐，Go 的 `%6s` 按 rune 数右对齐，两者对中文结果一致；`{:.2}` 对应 `%.2f`。
- Rust 枚举变体可以直接带字段；Go 要为每个变体建一个 struct，接口只负责统一方法集。

## 例 3：Option 与 if let / while let
Rust：
```rust
let found: Option<&User> = users.iter().find(|u| u.name == "Bob");
match found {
    Some(u) => println!("找到 {}，{} 岁", u.name, u.age),
    None => println!("没找到"),
}
let bob_age = found.map(|u| u.age).unwrap_or(0);
if let Some(u) = users.iter().find(|u| u.name == "Carol") { /* ... */ }
while let Some(top) = stack.pop() { print!("{top} "); }
```
Go：
```go
func findUser(users []User, name string) (User, bool) { /* ... */ }

bob, found := findUser(users, "Bob")
if found {
    fmt.Printf("找到 %s，%d 岁\n", bob.Name, bob.Age)
} else {
    fmt.Println("没找到")
}
bobAge := 0
if found { bobAge = int(bob.Age) } // 对应 map(...).unwrap_or(0)
if carol, ok := findUser(users, "Carol"); ok { /* ... */ }
for len(stack) > 0 {
    top := stack[len(stack)-1]
    stack = stack[:len(stack)-1]
    fmt.Printf("%d ", top)
}
```
- `Option<&User>` 在 Go 里通常写成 `(User, bool)` 或 `*User`（找不到返回 `nil`）；`if let` / `while let` 对应 comma-ok 的 `if` 和 `for len(stack) > 0`。
- Go 的 `(T, bool)` 忘了检查 `ok` 也能编译，Rust 的 `Option` 不处理就编译不过，这是两种错误模型的根本差异。

## 例 4：枚举解构与 match
Rust：
```rust
enum Command {
    Quit,
    Move { x: i32, y: i32 },
    Write(String),
}
for command in commands {
    match command {
        Command::Quit => println!("退出"),
        Command::Move { x, y } => println!("移动到 ({x}, {y})"),
        Command::Write(text) => println!("写入: {text}"),
    }
}
```
Go：
```go
type Command interface{ isCommand() }
type Quit struct{}
func (Quit) isCommand() {}
type Move struct{ X, Y int }
func (Move) isCommand() {}
type Write struct{ Text string }
func (Write) isCommand() {}

for _, command := range []Command{Move{10, 20}, Write{"保存文件"}, Quit{}} {
    switch c := command.(type) {
    case Quit:
        fmt.Println("退出")
    case Move:
        fmt.Printf("移动到 (%d, %d)\n", c.X, c.Y)
    case Write:
        fmt.Printf("写入: %s\n", c.Text)
    }
}
```
- Rust 的枚举变体自带数据，`match` 一次完成判别与解构；Go 用标记接口 + type switch，判别后还要断言出具体类型（`c` 已是具体类型）。
- type switch 不会强制覆盖所有实现，新增变体时旧 `switch` 会静默走 `default`（或什么都不做），需要自己补分支。
- 空变体 `Quit` 用 `struct{}` 表示，零值就是唯一值；Rust 的空变体 `Command::Quit` 同理。

## 速查表

| Rust | Go 1.22+ |
| --- | --- |
| `struct User { ... }` | `type User struct { ... }` |
| `impl User { fn new() }` | `func newUser() User` |
| `fn birthday(&mut self)` | `func (u *User) birthday()` |
| `fn is_adult(&self)` | `func (u User) isAdult()` |
| `#[derive(Debug)]` + `{:?}` | 实现 `String()` + `%v` |
| `enum Shape { Circle { .. } }` | 接口 + 各变体 struct |
| `match self { ... }` | type switch / 方法分派 |
| `Option<T>` | `(T, bool)` / `*T` / `nil` |
| `if let Some(x)` | `if x, ok := ...; ok` |
| `while let Some(x)` | `for len(stack) > 0` |
| `{:>6}` / `{:.2}` | `%6s` / `%.2f` |

## 给 Go 主栈的学习建议

- 方法集先想清楚「要不要改原值」：要改就用指针接收者，和 Rust 的 `&mut self` 对应；只读用值接收者即可。
- Rust 的枚举 + match 是「编译器帮你查漏」，Go 的接口 + type switch 更灵活但没有穷尽性；新增变体时全局搜一遍 type switch。
- 用 `(T, bool)` 模拟 Option 时，别忽略 `ok`；接口里的 `nil` 和「接口本身为 nil」还有一层坑，需要单独留意。
- 输出对照时注意 `%v` 和 `{:?}` 的差异，必要时实现 `String()` 让两边逐字节一致。
