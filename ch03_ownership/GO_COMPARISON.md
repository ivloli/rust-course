# 第 03 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`，**两边输出完全一致**。

```bash
# Rust
cd ~/rust-course && cargo run -p ch03_ownership

# Go（需要 Go 1.22+）
cd ~/rust-course/ch03_ownership/go && go run .
```

## 先记住三个总差异

1. **所有权**：Rust 每个值只有一个所有者，离开作用域自动 drop；Go 没有所有权，靠 GC 回收，变量赋值就是值复制。
2. **借用**：Rust 用 `&T` / `&mut T` 在编译期保证「共享不可变、可变不共享」；Go 用指针 `*T`，别名和并发安全全靠程序员与 `-race`。
3. **生命周期**：Rust 用 `'a` 描述引用能活多久；Go 没有生命周期，返回局部变量的指针也合法（逃逸分析会把它放到堆上）。

## 例 1：移动（move）
Rust：
```rust
let s1 = String::from("hello");
let s2 = s1;
// println!("{s1}"); // error[E0382]: borrow of moved value
println!("s2 = {s2}");
```
Go：
```go
s1 := "hello"
s2 := s1
fmt.Println(s1, s2) // 两者都可用：只是复制 string header
```
- Go 没有移动语义：赋值复制 header（指针 + 长度），底层字节共享；Rust 移动后原变量在编译期失效，避免 double free。

## 例 2：clone
Rust：
```rust
let s3 = s2.clone();
println!("s3 = {s3}, s2 仍然可用 = {s2}");
```
Go：
```go
s3 := s2
fmt.Printf("s3 = %s, s2 仍然可用 = %s\n", s3, s2)
```
- Go 里 `s3 := s2` 本来就合法，没有 `clone` 概念；需要真正复制切片/ map 时用 `slices.Clone`、`maps.Clone` 或 `append([]T(nil), s...)`。

## 例 3：Copy 类型
Rust：
```rust
let n1 = 5;
let n2 = n1;
println!("n1 = {n1}, n2 = {n2}");
```
Go：
```go
n1 := 5
n2 := n1
fmt.Printf("n1 = %d, n2 = %d\n", n1, n2)
```
- Go 的 `int`、数组、结构体都是值语义，赋值即复制，和 Rust 的 `Copy` 类型一致；切片、map、指针、接口则共享/别名底层数据。

## 例 4：借用
Rust：
```rust
fn calc_len(s: &str) -> usize { s.len() }
let len = calc_len(&s3);
println!("s3 = {s3}, 长度 = {len}");
```
Go：
```go
func calcLen(s string) int { return len(s) }
length := calcLen(s3)
fmt.Printf("s3 = %s, 长度 = %d\n", s3, length)
```
- 传给 Go 函数的 `string` 只复制 header，不复制内容，且字符串不可变，所以不需要借用标记；Rust 的 `&str` 还额外表达了「不拥有」。

## 例 5：可变借用
Rust：
```rust
let mut s4 = String::from("hello");
append_world(&mut s4);
// let r1 = &mut s4;
// let r2 = &mut s4; // error[E0499]
```
Go：
```go
s4 := "hello"
appendWorld(&s4)
// r1 := &s4
// r2 := &s4 // 合法：两个指针别名同一变量
// *r1 += "!" // r2 也能看到修改
```
- Rust 的可变借用是排他的，编译器禁止同时存在两个 `&mut`；Go 没有借用检查，两个指针指向同一变量完全合法，竞争要靠 `-race` 和同步原语发现。

## 例 6：切片
Rust：
```rust
fn first_word(s: &str) -> &str {
    s.split_whitespace().next().unwrap_or("")
}
let first = first_word(&s4);
```
Go：
```go
func firstWord(s string) string {
    fields := strings.Fields(s)
    if len(fields) == 0 { return "" }
    return fields[0]
}
first := firstWord(s4)
```
- Rust 返回的 `&str` 借用原串，原串在借用期间不能改；Go 的子串 header 也指向原串字节，但没有任何编译期约束阻止你同时修改/替换原变量。

## 例 7：生命周期
Rust：
```rust
fn longest<'a>(a: &'a str, b: &'a str) -> &'a str {
    if a.len() >= b.len() { a } else { b }
}
```
Go：
```go
func longest(a, b string) string {
    if len(a) >= len(b) { return a }
    return b
}
func dangle() *string { s := "x"; return &s } // 合法，逃逸到堆上
```
- Go 没有生命周期标注，返回 string 只是复制 header；返回局部变量指针也安全。Rust 必须用 `'a` 承诺返回值不比入参活得久，悬垂引用直接编译失败（E0106 / E0597）。

## 例 8：所有权拿走进再还回来
Rust：
```rust
fn takes_and_gives_back(s: String) -> String { s }
let s5 = takes_and_gives_back(s4);
println!("s5 = {s5}");
```
Go：
```go
func takesAndGivesBack(s string) string { return s }
s5 := takesAndGivesBack(s4)
fmt.Printf("s5 = %s\n", s5)
```
- Rust 的传参是所有权转移（move in），返回值再转移回来；Go 传 string 只是复制 header，`s4` 始终可用，函数签名也无法表达「我会消费这个值」。

## 速查表

| Rust | Go 1.22+ |
| --- | --- |
| `let s2 = s1;`（移动） | `s2 := s1`（复制 header） |
| `s2.clone()` | 切片 `slices.Clone`；string 直接赋值 |
| `i32` 赋值复制 | int/数组/结构体值复制 |
| `&T` 不可变借用 | `T` 传值或 `*T` |
| `&mut T` 排他可变借用 | `*T`，无排他检查 |
| `&str` / `&[T]` 切片 | `string` / `[]T`（共享底层） |
| `'a` 生命周期标注 | 无，GC 保证存活 |
| 返回局部变量引用（E0106） | 返回 `*T` 合法（逃逸分析） |
| `drop` 自动释放 | GC 回收 |

## 给 Go 主栈的学习建议

- 把 Rust 的「移动」理解成编译期的所有权转移，不是内存拷贝；Go 的赋值几乎总是复制 header 或值。
- 看到 `&mut` 就想到「这一段时间内只有我能改」，这是 Rust 无数据竞争的根基；Go 需要自己用 mutex/channel 保证。
- 切片和 map 在 Go 里是引用语义，函数内修改会影响调用方；Rust 用 `&mut` 把这件事写进类型，读代码时别忽略。
- 不要被生命周期吓到：它只是把「返回值借用谁」写成显式约束，Go 用 GC 换掉了这套检查。
