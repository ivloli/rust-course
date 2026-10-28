# 05 集合、迭代器与闭包

```bash
cargo run -p ch05_collections
```

## 学习目标

- 掌握 `Vec`、`String`、`HashMap` 的常用操作
- 理解迭代器是「惰性」的，`collect` / `sum` 等消费者才触发计算
- 使用 `entry` API 优雅地更新 HashMap
- 区分 `Fn` / `FnMut` / `FnOnce` 三种闭包捕获方式

## 代码导读（src/main.rs）

1. `numbers.get(10)` 返回 `Option<&i32>`；`numbers[10]` 会 panic，服务端代码尽量避免索引。
2. `"中文abc".len()` 是字节数，`chars().count()` 是字符数，处理多语言时不要混用。
3. 词频统计用 `*counts.entry(word).or_insert(0) += 1`，这是 Rust 最常用的 HashMap 模式。
4. `(1..=10).filter(..).map(..).collect()` 是零成本抽象：编译器会把它优化成接近手写循环。
5. `sort_by(|a, b| b.1.cmp(&a.1).then(a.0.cmp(b.0)))` 演示了主键降序、次键升序的稳定排序。
6. 闭包会按需捕获 `&T`、`&mut T` 或 `T`，分别对应 `Fn`、`FnMut`、`FnOnce`。

## 动手练习

1. 统计一段文本的字符频率（`HashMap<char, usize>`），输出出现最多的 3 个字符。
2. 用 `iter().filter(..).take(5)` 找出 1..=100 中所有能被 7 整除的数，不分配中间 Vec。
3. 用 `flat_map` 把 `["hello world", "rust"]` 展开成所有单词。
4. 把 `csv` 的 `fold` 改成 `scores.iter().map(|s| s.to_string()).collect::<Vec<_>>().join(",")`，比较两种写法。
5. 解释：为什么 `let mut calls = 0; (1..=3).for_each(|_| calls += 1);` 需要 `mut`？

## 工程提醒

- 迭代器链通常比手写下标循环更快，也更安全（无越界）。
- HashMap 默认使用随机哈希种子，遍历顺序不可预测；需要有序输出时先排序或换 `BTreeMap`。
- `String::len()` 不是字符数；HTTP 的 `Content-Length` 用的是字节数。
