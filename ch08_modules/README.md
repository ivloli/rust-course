# 08 模块、测试与工程结构

```bash
cargo run -p ch08_modules     # 运行示例
cargo test -p ch08_modules    # 运行 6 个单元测试 + 1 个集成测试
```

## 学习目标

- 用 `mod` / `pub` / `use` 组织多文件 crate
- 理解 `src/lib.rs` 与 `src/main.rs` 的区别，以及 `pub use` 的重导出
- 编写单元测试（`#[cfg(test)]`）与集成测试（`tests/` 目录）
- 认识 workspace 中「一个 crate 一个职责」的工程结构

## 代码导读

1. `src/lib.rs` 声明 `pub mod math; pub mod text;`，对应 `src/math.rs`、`src/text.rs`。
2. `math.rs` 内部还有嵌套模块 `stats`，访问路径是 `crate::math::stats::mean`。
3. `lib.rs` 中的 `pub use` 把 `gcd`、`mean`、`word_count` 提升到 crate 根，`main.rs` 因此可以短路径引用。
4. 单元测试与被测代码在同一文件，能访问私有项；集成测试只能访问公开 API。
5. `#[cfg(test)]` 保证测试代码不会进入正式构建。
6. 本章 crate 同时包含 lib 和 bin：`main.rs` 通过 crate 名 `ch08_modules` 使用 lib。

## 动手练习

1. 新增 `src/json.rs`，实现一个极简的键值解析函数，并在 `lib.rs` 中导出。
2. 给 `word_count` 加测试：中英文混排、多个连续空格、换行。
3. 把 `stats::mean` 改成 `pub(crate)`，观察 `main.rs` 与集成测试分别会发生什么。
4. 运行 `cargo test -p ch08_modules -- --nocapture`，让测试中的 `println!` 显示出来。
5. 思考：为什么集成测试不能测试私有函数？这如何影响 API 设计？

## 工程提醒

- 真实项目通常拆成 `core`（纯逻辑）+ `api`（HTTP）+ `infra`（DB/缓存）多个 crate，用 workspace 统一依赖版本。
- 测试不是「额外工作」：Rust 的测试编译进同一个二进制，写起来成本很低。
- `cargo fmt` / `cargo clippy` 应在提交前执行；后续章节的代码都保持可 clippy 通过。
