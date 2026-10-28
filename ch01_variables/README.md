# 01 变量、类型与可变性

```bash
cargo run -p ch01_variables
```

## 学习目标

- 理解 Rust 默认不可变，以及 `mut` 的确切含义
- 分清 shadowing 与 `mut`：前者是「新变量」，后者是「改旧值」
- 熟悉标量类型、元组、数组，以及 `String` 与 `&str` 的区别
- 知道整数溢出在 debug / release 下的不同行为

## 代码导读（src/main.rs）

1. `let x = 5` 后直接赋值会编译失败，这是 Rust 与多数语言最直观的差异。
2. shadowing 可以复用变量名并改变类型，常用于「解析 → 使用」的流水线。
3. `person` 元组演示了解构绑定与 `.0` 索引。
4. 数组长度是类型的一部分：`[i32; 5]` 与 `[i32; 3]` 是不同类型。
5. `as` 是可能截断的显式转换；生产代码中更安全的是 `try_into()`。
6. `const` 与 `let` 不同：它在编译期求值，可出现在常量上下文中。

## 动手练习

1. 给 `x` 加上 `mut`，观察编译器错误消失；再想想为什么 Rust 默认不可变。
2. 用 `u8` 保存 300 会编译失败吗？试试 `300u16 as u8`，结果是多少？
3. 用 `checked_add`、`saturating_add` 改写第 6 节，比较三种语义。
4. 写一个函数 `fn stats(scores: &[i32; 5]) -> (i32, i32)` 返回最大值与最小值。
5. 思考：为什么 `String` 赋值会「移动」，而 `i32` 赋值是复制？（提示：`Copy` trait，第 07 章详解）

## 延伸阅读

- 《The Rust Programming Language》第 3 章
- `std::primitive` 文档中的各整数类型溢出方法
