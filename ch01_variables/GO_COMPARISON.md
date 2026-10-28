# 第 01 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`，**两边输出完全一致**。

```bash
# Rust
cd ~/rust-course && cargo run -p ch01_variables

# Go（需要 Go 1.22+）
cd ~/rust-course/ch01_variables/go && go run .
```

## 先记住三个总差异

1. **可变性**：Rust 变量默认不可变，改值要 `mut`；Go 变量默认可变，只有 `const` 不可变。
2. **类型映射**：Rust 有元组、`[T; N]`、`&str`/`String`、`char`；Go 用结构体/多变量、`[N]T`/`[]T`、单一不可变 `string`、`rune`。
3. **转换与溢出**：两边都要求显式转换，但 Rust debug 构建溢出会 panic，Go 在任何构建下都静默回绕。

## 例 1：变量默认不可变
Rust：
```rust
let x = 5;
println!("x = {x}");
// x = 6; // error[E0384]: cannot assign twice
```
Go：
```go
x := 5
fmt.Println("x =", x)
// x = 6 完全合法：Go 没有 mut
```
- Go 变量声明后就能反复赋值；Rust 的不可变是默认安全，Go 只能靠约定和 `const`。

## 例 2：需要修改时加 mut
Rust：
```rust
let mut y = 10;
y += 1;
println!("y = {y}");
```
Go：
```go
y := 10
y++
fmt.Println("y =", y)
```
- `y++` 在 Go 里是语句而非表达式；Rust 的 `mut` 还是类型系统的一部分，`&mut` 会检查独占借用。

## 例 3：隐藏（shadowing）
Rust：
```rust
let x = x + 1;
let x = format!("现在 x 变成了字符串: {x}");
println!("{x}");
```
Go：
```go
{
    x := x + 1 // 内层作用域的新变量，遮蔽外层 x
    message := fmt.Sprintf("现在 x 变成了字符串: %d", x)
    fmt.Println(message)
}
```
- Rust 可在同一作用域连续 `let x` 新建变量并换类型；Go 同作用域重复 `:=` 会编译错误，必须开内层花括号。

## 例 4：类型标注与解析
Rust：
```rust
let guess: u32 = "42".parse().expect("输入不是数字");
println!("guess = {guess}");
```
Go：
```go
guess, err := strconv.Atoi("42")
if err != nil {
    panic(err) // 对应 .expect(...)
}
fmt.Println("guess =", guess)
```
- Rust 返回 `Result`；Go 返回 `(int, error)`。`Atoi` 得到平台相关 `int`，要定宽用 `ParseUint`；生产代码别 panic。

## 例 5：标量类型
Rust：
```rust
let integer: i64 = 1_000_000;
let float: f64 = 2.5;
let is_rust_fun = true;
let letter: char = '中';
println!("{integer} {float} {is_rust_fun} {letter}");
```
Go：
```go
var integer int64 = 1_000_000
var float float64 = 2.5
isRustFun := true
var letter rune = '中'
fmt.Printf("%d %g %t %c\n", integer, float, isRustFun, letter)
```
- `char` 与 `rune`（`int32` 别名）语义相同；`byte` 是 `uint8` 不是字符。`%c` 打印 rune，`%v` 打印码点。

## 例 6：整数溢出
Rust：
```rust
let max_u8 = u8::MAX;
println!("u8::MAX = {max_u8}");
// let overflow = max_u8 + 1; // debug 下 panic，release 回绕
```
Go：
```go
maxU8 := uint8(math.MaxUint8)
fmt.Println("u8::MAX =", maxU8)
wrapped := maxU8 + 1 // 255 + 1 静默变成 0
_ = wrapped
```
- **Go 的整数溢出运行时永远静默回绕，不会 panic**；常量运算溢出才是编译错误，所以 `uint8(300)` 编译失败，必须先赋给变量。

## 例 7：元组与数组
Rust：
```rust
let person: (&str, u32, bool) = ("小明", 28, true);
let (name, age, active) = person;
println!("{name} {age} 岁，在职: {active}");
let scores: [i32; 5] = [90, 85, 77, 60, 99];
```
Go：
```go
person := struct {
    name   string
    age    uint32
    active bool
}{"小明", 28, true}
name, age, active := person.name, person.age, person.active
scores := [5]int{90, 85, 77, 60, 99}
```
- Go 没有元组：数据组合用结构体，多返回值只用于函数。`[5]int` 是值类型；日常多用切片，`{:?}` 的逗号格式要自己拼。

## 例 8：字符串
Rust：
```rust
let literal: &str = "字面量";
let owned: String = String::from("堆上的 String");
let combined = format!("{literal} + {owned}");
println!("{combined}");
```
Go：
```go
literal := "字面量"
owned := "堆上的 String"
combined := fmt.Sprintf("%s + %s", literal, owned)
fmt.Println(combined)
```
- Go 只有一种**不可变** `string`，没有 `&str`/`String` 的借用区分；赋值只复制 header。可增长串用 `strings.Builder`。

## 例 9：常量
Rust：
```rust
const MAX_POINTS: u32 = 100_000;
println!("MAX_POINTS = {MAX_POINTS}");
```
Go：
```go
const maxPoints uint32 = 100_000
fmt.Println("MAX_POINTS =", maxPoints)
```
- 两者都在编译期求值；Go 常量可以无类型（`const n = 100`），Rust 的 `const` 必须写类型。命名习惯也不同。

## 例 10：显式转换
Rust：
```rust
let big: i64 = 300;
let small: i32 = big as i32;
let truncated: u8 = 300u16 as u8; // 300 % 256 = 44
```
Go：
```go
big := int64(300)
small := int32(big)
var medium uint16 = 300
truncated := uint8(medium) // 44
```
- Rust 用 `as`，Go 用 `T(v)`，都不会隐式转换；常量 `uint8(300)` 编译失败，先赋变量才会截断。

## 速查表

| Rust | Go 1.22+ |
| --- | --- |
| `let x = 5`（不可变） | `x := 5`（默认可变） |
| 同名 `let` 遮蔽 | 内层作用域 `:=` |
| `"42".parse::<u32>()` | `strconv.Atoi` / `ParseUint` |
| `char` | `rune`（`%c`） |
| `[i32; 5]` | `[5]int`（值类型） |
| `&str` / `String` | `string`（不可变） |
| `format!` | `fmt.Sprintf` |
| `x as u8` | `uint8(x)` |
| 溢出：debug panic / release 回绕 | 永远静默回绕 |

## 给 Go 主栈的学习建议

- 读 Rust 先找 `mut`：没有 `mut` 的值在作用域内不会变，能大幅减少状态推演。
- 把 shadowing 理解成「重新声明」；`parse`/`Atoi` 都强制面对失败，别用 `_` 吞掉 error。
