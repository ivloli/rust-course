# 第 06 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 在 `src/main.rs`，Go 在 `go/main.go`；**输出结构一致**，唯一差异是 `abc` 的底层解析文案（见文末）。

```bash
cd ~/rust-course && cargo run -p ch06_errors   # Rust：读根目录 Cargo.toml
cd ~/rust-course/ch06_errors/go && go run .    # Go：读 ../../Cargo.toml
```

## 先记住三个总差异

1. **错误表示**：Rust 用 `Result<T, E>` 把成功值和错误放进枚举；Go 用 `(T, error)`，成功时 `err == nil`。
2. **传播**：Rust 的 `?` 自动 `return Err(e.into())`，`From` 负责升级类型；Go 每步手写 `if err != nil`。
3. **缺失值**：Rust 用 `Option<T>` 强制处理；Go 用 `(T, bool)`、`nil` 或哨兵值，检查与否靠自觉。

---

## 例 1：Option ↔ comma-ok

Rust：

```rust
println!("get(1) = {:?}", numbers.get(1)); // Some(2)
println!("get(9) = {:?}", numbers.get(9)); // None
let scaled = numbers.iter().find(|&&n| n % 2 == 0).map(|n| n * 10).unwrap_or(0);
```
Go：

```go
if value, ok := get(numbers, 1); ok { fmt.Printf("get(1) = Some(%d)\n", value) } else { fmt.Println("get(1) = None") }
firstEven, found := 0, false
for _, n := range numbers { if n%2 == 0 { firstEven, found = n, true; break } }
scaled := 0
if found { scaled = firstEven * 10 }
```
Go 视角要点：

- **`(值, ok)` 就是 Go 的 Option**，map 取值、类型断言、channel 关闭检测都用它。
- Rust 有 `map` / `unwrap_or` 组合子；Go 只能写 `if`，组合性弱但直白。

---

## 例 2：Result + ? ↔ (T, error)

Rust：

```rust
enum AppError { EmptyInput, NotANumber(ParseIntError), OutOfRange(i64) }

fn parse_score(input: &str) -> Result<u32, AppError> {
    if input.trim().is_empty() { return Err(AppError::EmptyInput); }
    let value: i64 = input.trim().parse()?; // ? 触发 From<ParseIntError>
    if !(1..=100).contains(&value) { return Err(AppError::OutOfRange(value)); }
    Ok(value as u32)
}
// Display 输出 "不是合法整数: {e}" 等；source() 暴露 NotANumber 的底层错误
```
Go：

```go
var ErrEmptyInput = errors.New("输入不能为空") // 哨兵错误
type OutOfRangeError struct{ Value int }
func (e *OutOfRangeError) Error() string { return fmt.Sprintf("数值 %d 超出 1..=100 范围", e.Value) }
type NotANumberError struct{ Err error }
func (e *NotANumberError) Error() string { return fmt.Sprintf("不是合法整数: %v", e.Err) }
func (e *NotANumberError) Unwrap() error { return e.Err }

func parseScore(input string) (uint32, error) {
    trimmed := strings.TrimSpace(input)
    if trimmed == "" { return 0, ErrEmptyInput }
    value, err := strconv.Atoi(trimmed)
    if err != nil { return 0, &NotANumberError{Err: err} } // 手动代替 From
    if value < 1 || value > 100 { return 0, &OutOfRangeError{Value: value} }
    return uint32(value), nil
}
```
Go 视角要点：

- **`?` 的展开就是「失败提前 return」**；Rust 的 `?` 自动调用 `From::from`，Go 必须显式构造包装错误。
- **没有枚举变体**：固定错误用哨兵，带数据用结构体实现 `error`；`Error()` 对应 `Display`，`Unwrap()` 对应 `source()`。
- 调用方用 `errors.Is` 判哨兵、`errors.As` 取具体类型，对应 Rust 的 `match`；**丢了 `Unwrap` 就会断链**。
- Rust 的 `as u32` 不检查溢出；Go 的 `uint32(value)` 同样是显式转换。

---

## 例 3：unwrap 系列 ↔ 忽略错误 / 默认值

Rust：

```rust
let parsed: i32 = "10".parse().unwrap();
let fallback: i32 = "x".parse().unwrap_or(-1);
let computed: i32 = "8".parse::<i32>().map(|n| n * 2).unwrap_or_default();
```
Go：

```go
parsed, _ := strconv.Atoi("10") // 确定不会失败：忽略 error
fallback, err := strconv.Atoi("x")
if err != nil { fallback = -1 } // unwrap_or(-1)
computed, err := strconv.Atoi("8")
if err != nil { computed = 0 } else { computed *= 2 } // unwrap_or_default + map
```
Go 视角要点：

- Rust 的 `unwrap()` 会 panic；Go 的 `_` 是**静默丢弃**，失败时得到零值，生产代码里更危险。
- Go 没有 `unwrap_or` 组合子，默认值逻辑就是 `if err != nil`；要崩溃则 `panic(err)`。

---

## 例 4：Box<dyn Error> ↔ error 接口

Rust：

```rust
fn read_text_len(path: &str) -> Result<usize, Box<dyn std::error::Error>> {
    let content = std::fs::read_to_string(path)?; // main 中传 "Cargo.toml"
    Ok(content.len())
}
```
Go：

```go
func readTextLen(path string) (int, error) {
    content, err := os.ReadFile(path)
    if err != nil { return 0, err }
    return len(content), nil
}
length, err := readTextLen("../../Cargo.toml") // Go 从 ch06_errors/go 运行
```
Go 视角要点：

- **`error` 本身就是接口**，不需要 `Box<dyn Error>` 装箱；任何错误都能直接向上返回。
- **路径差异**：Rust 从 workspace 根目录读 `Cargo.toml`；Go 从 `go/` 运行，必须读 `../../Cargo.toml`，字节数（863）才对得上。

---

## 例 5：main 返回 Result ↔ 打印后退出

Rust：

```rust
fn run() -> Result<(), AppError> {
    let score = parse_score("100")?;
    println!("run() 解析结果 = {score}");
    Ok(())
} // main 里 match run()，Err 时 eprintln!
```
Go：

```go
func run() error {
    score, err := parseScore("100")
    if err != nil { return err }
    fmt.Printf("run() 解析结果 = %d\n", score)
    return nil
}
// main: if err := run(); err != nil { fmt.Fprintf(os.Stderr, "run() 失败: %v\n", err); os.Exit(1) }
```
Go 视角要点：

- Rust 的 `fn main() -> Result` 由标准库打印错误并返回非零码；Go 的 `main` 不能返回 error，要自己打印 + `os.Exit(1)`。
- `os.Exit` 会跳过 `defer`；Rust 从 `main` 返回 `Err` 则正常展开栈。

---

## 输出差异说明

11 行输出中 10 行完全一致，唯一差异：

```text
Rust: "abc" -> 错误: 不是合法整数: invalid digit found in string
Go:   "abc" -> 错误: 不是合法整数: strconv.Atoi: parsing "abc": invalid syntax
```

外层前缀已对齐；底层原文来自各自标准库（Rust `ParseIntError` ↔ Go `*strconv.NumError`），**没有为了模仿而硬编码翻译**。其余失败分支与所有成功分支逐字一致。

## 速查表

| Rust | Go 1.21+ |
| --- | --- |
| `Result<T, E>` | `(T, error)` |
| `Ok(v)` / `Err(e)` | `return v, nil` / `return zero, err` |
| `?` | `if err != nil { return ..., err }` |
| `From<E>` 自动转换 | 手动包装 `&MyError{Err: err}` |
| `enum AppError` | 哨兵 `var ErrX` + 实现 `error` 的结构体 |
| `impl Display` / `source` | `Error() string` / `Unwrap() error` |
| `match err` | `errors.Is` / `errors.As` |
| `Option<T>` / `unwrap_or` | `(T, bool)` + `if` 默认值 |
| `Box<dyn Error>` | `error`（本身就是接口） |
| `fn main() -> Result` | 打印 + `os.Exit(1)` |

## 给 Go 主栈的学习建议

- 把 `?` 读作「失败就 return」，把 `From` 读作「错误类型自动升级」；看到 `?` 先想目标错误类型是谁。
- 设计 Rust 错误枚举时先列全失败方式；这相当于 Go 的「哨兵 + 结构体」清单，但编译器强制调用方处理完整。
- 写 Go 保留 `Unwrap`，写 Rust 保留 `source()`，否则错误链断裂、日志看不到根因。
- 不要为了输出对齐翻译标准库文案；理解应用层文案与底层原文的分层更重要。
