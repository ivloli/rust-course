# 02 函数、控制流与模式匹配

```bash
cargo run -p ch02_control_flow
```

## 学习目标

- 函数定义、返回值与「最后一行表达式即返回值」的写法
- `if` / `loop` / `while` / `for` 的适用场景
- `match` 的穷尽性检查、区间模式、守卫与解构
- 标签循环（`'outer`）跳出多层嵌套

## 代码导读（src/main.rs）

1. `sign` 返回 `&'static str`：字符串字面量有静态生命周期，不涉及分配。
2. `gcd` 用 `while` 演示参数加 `mut` 的写法。
3. `loop { ... break value }` 是 Rust 中带返回值的循环，适合「重试直到成功」。
4. `match temp { n if n < 0 => ... }` 展示了 match guard；区间模式 `1..=20` 必须是连续且不重叠的。
5. `fizzbuzz` 用 `match (n % 3, n % 5)` 把多条件压缩成元组模式，比多重 `if` 更清晰。
6. 注意：`for n in 1..=15` 中 `15` 是 `u32`，因此 `fizzbuzz` 的参数类型必须匹配。

## 动手练习

1. 把 `fizzbuzz` 改成返回 `impl Display` 或 `Cow<'static, str>`，避免部分分支分配。
2. 用 `match` 实现一个四则运算解析器：输入 `("add", 1, 2)` 输出结果。
3. 验证：把 `match` 中的 `_` 分支删掉，编译器会报什么？为什么这是优点？
4. 实现 `fib` 的递归版本，并思考为什么生产代码更偏好迭代版本。
5. 用带标签的 `continue 'outer` 改写第 7 节，跳过特定组合。

## 工程提醒

- Rust 没有三元运算符，`if` 就是表达式。
- `loop` 不会因为「可能不执行」而要求初始化返回值；`while` / `for` 则不行。
