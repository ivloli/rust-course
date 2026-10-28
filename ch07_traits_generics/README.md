# 07 Trait、泛型与多态

```bash
cargo run -p ch07_traits_generics
```

## 学习目标

- 定义 trait、编写默认方法、为自定义类型实现 trait
- 用泛型 + trait bound（`T: Summary`）编写可复用函数
- 理解静态分发（`impl Trait` / 泛型）与动态分发（`dyn Trait`）的取舍
- 熟悉标准库最常用的 `From` / `Into` / `Display`

## 代码导读（src/main.rs）

1. `Summary` 的 `summarize` 是默认方法，`Tweet` 和 `Article` 可以按需覆盖（这里都覆盖了，便于对比）。
2. `notify` 使用 `&impl Summary`，编译器为每个具体类型生成一份代码（单态化），运行时零开销。
3. `notify_with_display<T: Summary + Display>` 展示多约束；`where` 子句在约束多时更易读。
4. `make_tweet() -> impl Summary` 隐藏具体类型，后续可以替换实现而不破坏调用方。
5. `Vec<Box<dyn Summary>>` 是异构集合：每个元素通过虚表调用，灵活但有间接调用开销。
6. `From<u32> for UserId` 自动获得 `Into`；这是 Rust 转换 trait 的对称设计。
7. `largest` 对 `T` 的要求（`PartialOrd + Copy`）由编译器强制检查，调用时无需额外标注。

## 动手练习

1. 给 `Summary` 增加方法 `fn tag(&self) -> String { "general".into() }`，让 `Tweet` 覆盖它。
2. 为 `Circle` 实现 `Summary`，并在 `Vec<Box<dyn Summary>>` 中混入三种类型。
3. 用泛型写 `fn max_by_key<T, K: Ord>(items: &[T], key: impl Fn(&T) -> K) -> Option<&T>`。
4. 解释：为什么 `dyn Summary` 不能有返回 `Self` 的方法？（对象安全 / object safety）
5. 把 `UserId` 的 `From<u32>` 改成 `TryFrom<i64>`，体会「可能失败」的转换。

## 工程提醒

- 优先用泛型/`impl Trait`（静态分发），需要异构集合或插件式扩展时再用 `dyn Trait`。
- trait 的孤儿规则：只能为自己的类型或本 crate 的 trait 实现；不能给 `Vec<T>` 实现 `std` 的 trait。
- 异步 trait 方法不能直接用于 `dyn`（截至 Rust 2024 仍需 `async-trait` 或 `trait-variant`），第 14 章会用到。
