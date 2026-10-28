# 第 07 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 在 `src/main.rs`，Go 在 `go/main.go`，**两边输出完全一致**；trait ↔ interface 是主线，但默认方法、泛型约束、`impl Trait` 并不对称。

```bash
cd ~/rust-course && cargo run -p ch07_traits_generics
cd ~/rust-course/ch07_traits_generics/go && go run .   # Go 1.21+，用到 cmp.Ordered
```

## 先记住三个总差异

1. **抽象机制**：Rust trait 可给已有类型补实现（受孤儿规则限制）；Go interface 只描述方法集，类型**隐式**满足，方法必须和类型同包。
2. **分发方式**：Rust 泛型 / `impl Trait` 是编译期单态化（静态）；Go 的接口参数和返回值**默认动态分发**。
3. **代码复用**：Rust trait 有默认方法、关联类型；Go 接口只有方法签名，复用靠辅助函数、嵌入或泛型。

---

## 例 1：trait ↔ interface
Rust：
```rust
trait Summary {
    fn summarize_author(&self) -> String;
    fn summarize(&self) -> String;
}
impl Summary for Tweet { /* 方法体省略 */ }
```
Go：
```go
type Summary interface {
    SummarizeAuthor() string
    Summarize() string
}
func (t Tweet) Summarize() string { return t.SummarizeAuthor() + ": " + t.Body }
```
Go 视角要点：
- **隐式实现**：方法集齐全就满足接口，不写 `implements`；坏处是「实现了哪些接口」要靠工具查。
- 接口方法必须大写（`Summarize`），否则包外类型无法实现；值接收者会复制，大结构体用指针接收者。

---

## 例 2：默认方法 ↔ 辅助函数 / 嵌入
Rust：
```rust
trait Summary {
    fn summarize_author(&self) -> String;
    fn summarize(&self) -> String {
        format!("（阅读更多来自 {} 的内容…）", self.summarize_author())
    }
}
```
Go：
```go
func defaultSummary(author string) string {
    return fmt.Sprintf("（阅读更多来自 %s 的内容…）", author)
}
// 嵌入 base 类型也能共享实现，但嵌入是组合，无法回调外层的动态方法。
```
Go 视角要点：
- **Go 接口不能带方法体**，没有「不覆盖就用默认实现」；要么每类型都写，要么抽辅助函数。
- 嵌入可模拟「继承默认实现」，但被嵌入类型不知道外层类型，无法分派到实现者；本例两边都没走到默认分支。

---

## 例 3：impl Trait 参数 / 多约束 ↔ 接口参数 / 组合接口
Rust：
```rust
fn notify(item: &impl Summary) -> String { format!("Breaking! {}", item.summarize()) }
fn notify_with_display<T: Summary + fmt::Display>(item: &T) -> String {
    format!("Breaking! {item} | {}", item.summarize())
}
```
Go：
```go
func notify(item Summary) string { return "Breaking! " + item.Summarize() }
type displaySummary interface {
    Summary
    fmt.Stringer // 对应 Display
}
func notifyWithDisplay(item displaySummary) string { return fmt.Sprintf("Breaking! %s | %s", item, item.Summarize()) }
```
Go 视角要点：
- `&impl Summary` 是**泛型语法糖**，每类型生成一份 `notify`；Go 接口参数走 itab 间接调用，并发生一次装箱/复制。
- Rust 多约束用 `+`；Go 用**接口嵌入**合并方法集，`fmt.Stringer` 对应 `Display`，`%s` 自动调用 `String()`。

---

## 例 4：返回 impl Trait ↔ 返回接口
Rust：
```rust
fn make_tweet() -> impl Summary {
    Tweet { username: "rustlang".into(), body: "Rust 1.99 已发布".into() }
}
```
Go：
```go
func makeTweet() Summary {
    return Tweet{Username: "rustlang", Body: "Rust 1.99 已发布"}
}
```
Go 视角要点：
- **语义不同**：`impl Summary` 是不透明具体类型，编译器知道真实类型并静态分发；`Summary` 是接口值，真实类型被装箱，动态分发。
- **Go 没有静态分发保证**：编译器偶尔去虚拟化，但语言不承诺；返回接口后调用方只能类型断言，Rust 的 `impl Trait` 连断言都做不到。

---

## 例 5：dyn Trait ↔ []Summary / []Area
Rust：
```rust
let items: Vec<Box<dyn Summary>> = vec![
    Box::new(Article { title: "动态分发".into(), author: "A".into() }),
    Box::new(Tweet { username: "B".into(), body: "在运行时决定调用".into() }),
];
let shapes: Vec<Box<dyn Area>> = vec![Box::new(Square(2.0)), Box::new(Circle(1.0))];
```
Go：
```go
items := []Summary{Article{Title: "动态分发", Author: "A"}, Tweet{Username: "B", Body: "在运行时决定调用"}}
shapes := []Area{Square{Side: 2.0}, Circle{Radius: 1.0}}
for _, item := range items { fmt.Println("dyn ->", item.Summarize()) }
```
Go 视角要点：
- **接口切片天生异构**：`[]Summary` / `[]Area` 不需要 `Box`；Rust 必须 `Box<dyn Trait>` 统一大小。接口值是类型指针 + 数据指针，`Box<dyn>` 是数据指针 + 虚表指针。
- **没有 trait object 的结构体字面量简写**：Rust 可 `Box::new(Article { .. })` 进 `vec![]`；Go 必须先构造值，字段写全名或按位置写全。面积用 `%.2f` 对齐 Rust 的 `{:.2}`。

---

## 例 6：泛型 largest ↔ cmp.Ordered
Rust：
```rust
fn largest<T: PartialOrd + Copy>(list: &[T]) -> T {
    let mut largest = list[0];
    for &item in list { if item > largest { largest = item; } }
    largest
}
```
Go：
```go
func largest[T cmp.Ordered](list []T) T {
    largest := list[0]
    for _, item := range list { if item > largest { largest = item } }
    return largest
}
```
Go 视角要点：
- `PartialOrd` 允许部分可比较（如 `f64` 的 NaN）；`cmp.Ordered` 是内置有序类型的并集，直接用 `>`。
- **不需要 `Copy` bound**：Go 赋值天然复制；Rust 从 `&[T]` 取 `T` 必须 `Copy`。空切片两边都在 `list[0]` panic。

---

## 例 7：From / Into / Display ↔ 构造函数 / 转换 / String
Rust：
```rust
struct UserId(u64);
impl From<u32> for UserId { fn from(value: u32) -> Self { UserId(value as u64) } }
impl fmt::Display for UserId {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result { write!(f, "user#{}", self.0) }
}
let id = UserId::from(42u32);
let other: UserId = 7u32.into();
```
Go：
```go
type UserID uint64
func NewUserID(value uint32) UserID { return UserID(value) } // 对应 From
func (id UserID) String() string { return fmt.Sprintf("user#%d", uint64(id)) }
id := NewUserID(42)
other := UserID(7) // 显式转换，没有 Into
```
Go 视角要点：
- **没有 `From` / `Into`**：Go 用 `NewXxx` 构造函数和 `T(v)` 显式转换，没有自动推导的 `.into()`。
- `String()` 对应 `Display`；**没有运算符重载**：Rust 可为 `UserId` 实现 `Add` / `PartialEq`，Go 不允许自定义 `+` / `==`，只能写方法或函数。

---

## 速查表

| Rust | Go 1.21+ |
| --- | --- |
| `trait Summary` | `type Summary interface` |
| `impl Summary for T` | 给 `T` 写同名方法（隐式） |
| trait 默认方法 | 辅助函数 / 结构体嵌入 |
| `&impl Summary` 参数 | `Summary` 接口参数（动态） |
| `T: Summary + Display` | 嵌入 `Summary` 与 `fmt.Stringer` |
| `-> impl Summary` | `-> Summary`（语义不同） |
| `Vec<Box<dyn Summary>>` | `[]Summary` |
| `T: PartialOrd + Copy` | `T cmp.Ordered` |
| `From<u32>` / `Into` | `NewUserID(u32)` / `UserID(7)` |
| `impl Display` | `String() string` |
| 运算符重载 | 不支持，写方法/函数 |
| `{:.2}` | `%.2f` |

## 给 Go 主栈的学习建议

- 把 Rust trait 理解成「更强大的 interface」：多了默认方法、关联类型、泛型约束和静态分发。
- 看到 `impl Trait` 参数，翻译成「泛型 + 约束」，不要翻译成 Go 接口参数——前者静态，后者动态。
- 看到 `Box<dyn Trait>`，翻译成 `[]Interface` 或接口字段；Rust 显式装箱，Go 接口值自动完成。
- 共享默认实现用 trait 默认方法（Rust）或辅助函数 + 嵌入（Go）；嵌入不是继承，没有虚方法派发。
