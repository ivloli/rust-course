# 第 08 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 在 `src/`，Go 在 `go/`，**程序输出与 7 个单元测试完全一致**。Rust 用 crate + `mod`，Go 用 module + package，都把「一个目录一个职责」作为工程边界。

```bash
cd ~/rust-course && cargo run -p ch08_modules
cd ~/rust-course && cargo test -p ch08_modules
cd ~/rust-course/ch08_modules/go && go run .
cd ~/rust-course/ch08_modules/go && go test ./... -v
```

## 先记住三个总差异

1. **组织单位**：Rust crate 是编译单元，`mod` 可多层嵌套且不要求和目录一一对应；Go 的 package 就是目录，嵌套模块必须变成子目录。
2. **可见性**：Rust 用 `pub` / `pub(crate)` / 私有逐项标注；Go 只看**首字母**，大写导出、小写包内私有，没有中间粒度。
3. **测试位置**：Rust 单元测试在 `#[cfg(test)] mod tests`，集成测试在 `tests/`；Go 统一用同目录 `_test.go`，`package foo_test` 则只能访问导出项。

---

## 例 1：crate / mod ↔ module / package

Rust：
```rust
// src/lib.rs
pub mod math;
pub mod text;
// src/main.rs: use ch08_modules::{gcd, mean, word_count};
```
Go：
```go
// go.mod: module ch08_go
import (
    "ch08_go/mathx"
    "ch08_go/textx"
)
```
Go 视角要点：
- **`go.mod` 的 module 相当于 crate 名 + 导入前缀**；`ch08_go/mathx` 是 module 路径 + 目录。Rust 用 `::`，Go 用 `/`。
- Rust 的 `mod math;` 去同目录找 `math.rs`；Go 包由目录自动确定，同目录所有 `.go` 文件共享包名。Rust 的 bin + lib 可同名共存，Go 的 main 包必须单独目录。

---

## 例 2：pub ↔ 首字母大写

Rust：
```rust
pub fn gcd(a: u64, b: u64) -> u64 { /* ... */ }
mod stats { pub fn mean(values: &[f64]) -> Option<f64> { /* ... */ } }
```
Go：
```go
func Gcd(a, b uint64) uint64 { /* ... */ }  // 导出
func Mean(values []float64) (float64, bool) { /* ... */ }
func helper() {} // 小写：包内私有
```
Go 视角要点：
- **`Gcd` 导出，`gcd` 不导出**；Go 没有 `pub(crate)` / `pub(super)`，包内私有就是小写，跨包可见就是大写。
- 结构体字段同样靠首字母：`Value` 导出，`value` 包内私有；Rust 的 `pub` 只影响路径可达性，字段默认私有。

---

## 例 3：嵌套 math::stats ↔ stats.go / 子包

Rust：
```rust
pub mod math {
    pub fn gcd(a: u64, b: u64) -> u64 { /* ... */ }
    pub mod stats { pub fn mean(values: &[f64]) -> Option<f64> { /* ... */ } }
}
```
Go：
```go
// mathx/math.go 与 mathx/stats.go 都是 package mathx
func Gcd(a, b uint64) uint64 { /* ... */ }
func Mean(values []float64) (float64, bool) { /* ... */ } // 原 math::stats::mean
```
Go 视角要点：
- **两种落地方式**：摊平成同包 `stats.go`（本例，调用 `mathx.Mean`），或建 `mathx/stats/` 子目录形成 `ch08_go/mathx/stats` 包（`stats.Mean`）。
- Rust 模块布局可与目录不一致；Go 的包路径**严格等于目录结构**。摊平会丢失 `stats` 命名空间，函数多了应拆子包。

---

## 例 4：pub use ↔ 没有重导出

Rust：
```rust
// src/lib.rs
pub use math::{gcd, stats::mean};
pub use text::word_count;
// 调用方：use ch08_modules::{gcd, mean, word_count};
```
Go：
```go
// 没有 re-export，调用方直接导入具体包：
mathx.Gcd(48, 18)
mathx.Mean([]float64{2.0, 4.0})
textx.WordCount("a b c")
```
Go 视角要点：
- **Go 没有 `pub use`**：导出名只能属于定义它的包；想暴露短名字只能写包装函数，维护成本更高。
- Rust 的 `pub use` 是 API 门面工具，内部层级可很深；Go 的对应手段是**扁平包设计**或门面包（facade）。

---

## 例 5：单元测试 ↔ _test.go

Rust：
```rust
// src/math.rs
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn gcd_works() { assert_eq!(gcd(48, 18), 6); }
}
```
Go：
```go
// mathx/math_test.go
package mathx
import "testing"
func TestGcdWorks(t *testing.T) {
    if got := Gcd(48, 18); got != 6 { t.Fatalf("Gcd = %d, want 6", got) }
}
```
Go 视角要点：
- **命名规则**：文件以 `_test.go` 结尾，函数必须是 `TestXxx(t *testing.T)`；`go build` / `go run` 忽略它们，等价于 `#[cfg(test)]`。
- 同包测试可访问未导出标识符，对应 Rust 同文件 `mod tests`；断言要手写 `if + t.Fatalf`，标准库没有 `assert_eq!`。本例 7 个测试对应 Rust 的 4 个 math + 3 个 text。

---

## 例 6：集成测试 ↔ 外部测试包

Rust：
```rust
// tests/integration.rs
use ch08_modules::{mean, word_count};
#[test]
fn end_to_end_metrics() {
    assert_eq!(word_count("rust makes systems programming fun"), 5);
    assert_eq!(mean(&[1.0, 2.0, 3.0]), Some(2.0));
}
```
Go：
```go
// 放 mathx/integration_test.go 即可：
package mathx_test
import ("testing"; "ch08_go/mathx"; "ch08_go/textx")
func TestEndToEndMetrics(t *testing.T) {
    if got := textx.WordCount("rust makes systems programming fun"); got != 5 { t.Fatal(got) }
}
```
Go 视角要点：
- Rust 的 `tests/` 每个文件编译成独立 crate；Go 用 **`package foo_test`** 表示外部测试，只能访问导出 API。本例为保持 7 个测试与 Rust 一一对应，没有额外加入它。
- Rust 集成测试走 crate 根的重导出；Go 必须导入具体包路径，再次体现没有 `pub use`。`go test ./...` 跑当前 module 的全部包。

---

## 例 7：输出格式对齐

Rust：
```rust
println!("mean([2.0, 4.0]) = {:?}", mean(&[2.0, 4.0]));
println!("嵌套路径也能访问: {:?}", ch08_modules::math::stats::mean(&[1.0, 2.0, 3.0]));
```
Go：
```go
fmt.Printf("mean(%s) = %s\n", formatFloats(inputs), formatOptionFloat(mean, ok))
fmt.Printf("嵌套路径也能访问: %s\n", formatOptionFloat(nested, nestedOK))
// formatFloats → [2.0, 4.0]；formatOptionFloat → Some(3.0) / None
```
Go 视角要点：
- **`Option<f64>` 没有直接对应物**：`(float64, bool)` 要用 helper 打印成 `Some(3.0)` / `None`；`%v` 会把 `3.0` 打成 `3`，需补 `.0`。
- **slice 的 Debug 格式**：Rust `{:?}` 输出 `[2.0, 4.0]`，Go `%v` 输出 `[2 4]`，必须自己拼。浮点最短表示用 `strconv.FormatFloat(v, 'g', -1, 64)`。

---

## 速查表

| Rust | Go 1.21+ |
| --- | --- |
| crate | module（`go.mod`） |
| `mod math;` | `mathx/` 目录 + `package mathx` |
| `math::stats::mean` | `stats.Mean` 或摊平成 `mathx.Mean` |
| `pub fn` / 私有 `fn` | 首字母大写 / 小写 |
| `pub use` | 无 re-export，包装函数或完整路径 |
| `use ch08_modules::gcd;` | `import "ch08_go/mathx"` |
| `#[cfg(test)] mod tests` | 同包 `_test.go` |
| `tests/integration.rs` | `package foo_test` 外部测试 |
| `cargo test -p ch08_modules` | `go test ./...` |
| `Option<f64>` + `{:?}` | `(float64, bool)` + 手写格式化 |
| `[2.0, 4.0]`（Debug） | 手写 `[2.0, 4.0]`（`%v` 是 `[2 4]`） |

## 给 Go 主栈的学习建议

- 把 Rust crate 想成 Go module，把 `mod` 想成 package；区别是 Rust 模块可嵌套在单文件里，Go 必须落到目录。
- 设计 Rust 模块时先想好公开路径，`pub use` 是门面工具；设计 Go 包时尽量扁平、短路径，因为导出名不能搬家。
- 测试迁移很直接：同文件 `mod tests` → 同包 `_test.go`，`tests/` → `package foo_test`；断言库要自己补或引第三方。
- 不要指望 Go 自动打印 Rust 的 Debug 格式；`Option`、slice、浮点数都要显式格式化，这是 diff 对齐的主要工作量。
- 提交前跑 `gofmt -l .`（应无输出）、`go vet ./...`、`go test ./... -v`，对应 Rust 的 fmt / clippy / test 三件套。
