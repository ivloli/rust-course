# 第 02 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`，**两边输出完全一致**。

```bash
# Rust
cd ~/rust-course && cargo run -p ch02_control_flow

# Go（需要 Go 1.22+）
cd ~/rust-course/ch02_control_flow/go && go run .
```

## 先记住三个总差异

1. **表达式 vs 语句**：Rust 的 `if`、`match`、`loop` 都是表达式，能直接产生值；Go 的控制流全是语句，结果要先声明变量再赋值。
2. **循环只有 for**：Rust 有 `loop` / `while` / `for`；Go 只有 `for`，靠 `for {}`、`for cond`、三段式表达三种形态。
3. **模式匹配能力**：Rust 的 `match` 支持区间、守卫、解构且强制穷尽；Go 的 `switch` 只匹配值或条件，没有解构和穷尽性检查。

## 例 1：函数与返回值
Rust：
```rust
fn gcd(mut a: u64, mut b: u64) -> u64 {
    while b != 0 { let r = a % b; a = b; b = r; }
    a
}
```
Go：
```go
func gcd(a, b uint64) uint64 {
    for b != 0 {
        a, b = b, a%b
    }
    return a
}
```
- Rust 最后一行表达式就是返回值；Go 必须 `return`，参数天然可重新赋值（不需要 `mut`），`a, b = b, a%b` 这种多重赋值 Rust 也没有。

## 例 2：if 是表达式
Rust：
```rust
let age = 20;
let category = if age < 18 { "未成年" } else { "成年" };
```
Go：
```go
age := 20
category := "成年"
if age < 18 {
    category = "未成年"
}
```
- Go 没有三元运算符，`if` 不能出现在赋值右侧；先给默认值、再在分支里覆盖是最常见写法。Rust 的分支还必须是同类型表达式。

## 例 3：loop 与 break 返回值
Rust：
```rust
let mut counter = 0;
let doubled = loop {
    counter += 1;
    if counter == 5 { break counter * 2; }
};
```
Go：
```go
counter, doubled := 0, 0
for {
    counter++
    if counter == 5 { doubled = counter * 2; break }
}
```
- Go 的 `break` 不能带值：结果变量在循环外声明，`break` 之后再用。`for {}` 就是无限循环，Rust 的 `loop` 还能推断 `break value` 的类型。

## 例 4：while
Rust：
```rust
let mut countdown = 3;
while countdown > 0 {
    print!("{countdown} ");
    countdown -= 1;
}
println!("发射！");
```
Go：
```go
countdown := 3
for countdown > 0 {
    fmt.Printf("%d ", countdown)
    countdown--
}
fmt.Println("发射！")
```
- Go 的 `for cond` 直接对应 `while`，没有单独的 `while` 关键字；`print!` / `println!` 分别对应 `fmt.Printf` / `fmt.Println`。

## 例 5：for 与 range
Rust：
```rust
for i in 1..=3 { print!("for: {i} "); }
for (index, score) in [88, 92, 60].iter().enumerate() {
    println!("第 {index} 位同学: {score}");
}
```
Go：
```go
for i := 1; i <= 3; i++ {
    fmt.Printf("for: %d ", i)
}
for index, score := range []int{88, 92, 60} {
    fmt.Printf("第 %d 位同学: %d\n", index, score)
}
```
- Rust 的 `1..=3` 是闭区间迭代器；Go 用三段式 `for`，闭区间写 `i <= 3`（Go 1.22 也支持 `for range 3`，得到 0、1、2）。`enumerate()` 就是 `for i, v := range`。

## 例 6：带标签的循环
Rust：
```rust
'outer: for i in 0..5 {
    for j in 0..5 {
        if i * j == 6 {
            println!("找到 i={i}, j={j}");
            break 'outer;
        }
    }
}
```
Go：
```go
outer:
for i := 0; i < 5; i++ {
    for j := 0; j < 5; j++ {
        if i*j == 6 {
            fmt.Printf("找到 i=%d, j=%d\n", i, j)
            break outer
        }
    }
}
```
- 两边都能一次跳出多层循环：Rust 用 `'label`，Go 用 `label:` + `break label`。Rust 的标签可加在任意块上，Go 的标签只能标在 `for`/`switch`/`select` 前。

## 例 7：match 的守卫与区间
Rust：
```rust
let advice = match temp {
    n if n < 0 => "穿厚点",
    0 => "刚刚好",
    1..=20 => "舒适",
    _ => "有点热",
};
```
Go：
```go
advice := "有点热"
switch {
case temp < 0:
    advice = "穿厚点"
case temp == 0:
    advice = "刚刚好"
case temp >= 1 && temp <= 20:
    advice = "舒适"
}
```
- Go 的 `switch` 默认不穿透（无需 `break`），且**不会检查穷尽性**；Rust 的 `match` 少一个分支就编译失败。守卫在 Go 里直接写进 `case` 条件。

## 例 8：元组模式与 fizzbuzz
Rust：
```rust
match (n % 3, n % 5) {
    (0, 0) => "FizzBuzz".to_string(),
    (0, _) => "Fizz".to_string(),
    (_, 0) => "Buzz".to_string(),
    _ => n.to_string(),
}
```
Go：
```go
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
```
- Go 没有元组模式，`(n%3, n%5)` 要展开成条件表达式；`strconv.Itoa` 对应 `n.to_string()`。命令元组也一样：Rust 用 `match ("move", 10)` 解构，Go 用结构体字段 + `switch command.action`。

## 速查表

| Rust | Go 1.22+ |
| --- | --- |
| `fn f(x: i32) -> i32` | `func f(x int) int` |
| 尾表达式返回 | 显式 `return` |
| `if cond { a } else { b }` 作为值 | 先声明变量再分支赋值 |
| `while cond` | `for cond` |
| `loop { break v }` | `for { ... }` + 外部变量 |
| `for i in 1..=3` | `for i := 1; i <= 3; i++` |
| `iter().enumerate()` | `for i, v := range` |
| `'outer: ... break 'outer` | `outer: ... break outer` |
| `match n { 1..=20 => ... }` | `switch { case n >= 1 && n <= 20: }` |
| `match (a, b)` | 条件组合或结构体 |

## 给 Go 主栈的学习建议

- 看到 Rust 的 `if`/`match` 出现在赋值右侧，先意识到它是表达式；移植 Go 时改成「默认值 + 分支覆盖」。
- Rust 的 `loop` 常用来「重试直到成功」；Go 用 `for {}`，结果变量声明在循环外。
- `match` 的穷尽性是免费保险，Go 的 `switch` 需要自己写 `default` 兜底。
