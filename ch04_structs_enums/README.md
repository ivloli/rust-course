# 04 结构体、枚举与 Option

```bash
cargo run -p ch04_structs_enums
```

## 学习目标

- 定义结构体、关联函数与方法，理解 `self` / `&self` / `&mut self`
- 用枚举表达「多种可能的值」，并携带不同数据
- 用 `match` 解构枚举，理解穷尽性检查
- 用 `Option<T>` 替代 null，掌握 `if let` / `while let`

## 代码导读（src/main.rs）

1. `impl User` 中的 `new` 没有 `self`，属于关联函数（类似 Java 静态方法）；`birthday` 需要 `&mut self`。
2. `Shape` 的每个变体携带的字段不同，`area()` 用 `match` 分支处理，编译器保证不漏分支。
3. `shapes.iter().map(Shape::area)` 演示方法可以直接当函数指针使用。
4. `Option<&User>` 来自 `find`，用 `match`、`if let` 或组合子处理，编译器强制你面对「没有」的情况。
5. `while let Some(top) = stack.pop()` 是消费型循环的惯用写法。
6. `Command` 是典型的「消息枚举」，Web 服务里的事件/命令通常这样建模。

## 动手练习

1. 给 `User` 加一个 `email: Option<String>` 字段，并在 `greeting` 中处理「未填写」。
2. 给 `Shape` 增加 `Perimeter` 方法，思考如何用 `match` 统一处理三种形状。
3. 用枚举实现 `enum JsonValue { Null, Bool(bool), Number(f64), Str(String), Array(Vec<JsonValue>) }`，并写一个 `to_string`。
4. 把 `Command::Move` 改为元组变体 `Move(i32, i32)`，观察解构写法变化。
5. 尝试给 `Shape` 加第四个变体但不修改 `area()`，读一下编译器的提示。

## 工程提醒

- 结构体 + 枚举 + match 是 Rust 的主要「数据建模三件套」。
- 不要在结构体里滥用 `pub` 字段；对外暴露方法与不变式更安全。
- `Option<&T>` 是 `Copy`，因为引用是 `Copy`，但 `Option<String>` 不是。
