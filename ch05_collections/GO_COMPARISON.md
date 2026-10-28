# 第 05 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`，**两边输出完全一致**。

```bash
# Rust
cd ~/rust-course && cargo run -p ch05_collections

# Go（需要 Go 1.22+，用到 slices / cmp / range int）
cd ~/rust-course/ch05_collections/go && go run .
```

## 先记住三个总差异

1. **数据建模**：Rust 用 `Vec<T>` / `HashMap<K, V>` / `String` 拥有数据；Go 用 `[]T` / `map[K]V` / `string`，slice 和 map 是引用语义。
2. **迭代方式**：Rust 的 `Iterator` 链是惰性的、编译期单态化；Go 通常直接写 `for`，Go 1.23+ 也可以用 `iter.Seq` 写惰性管道。
3. **缺失值与错误**：Rust 用 `Option<T>` / `Result<T, E>`；Go 用 `(T, bool)`、`(T, error)` 和 `nil`。

---

## 例 1：Vec ↔ slice

Rust：

```rust
let mut numbers = vec![3, 1, 2];
numbers.sort();
numbers.push(5);
println!("numbers = {numbers:?}");

println!("get(10) = {:?}", numbers.get(10)); // Option<&i32>
// numbers[10]; // panic
```

Go：

```go
numbers := []int{3, 1, 2}
slices.Sort(numbers)
numbers = append(numbers, 5)
fmt.Println("numbers =", formatInts(numbers))

if value, ok := get(numbers, 10); ok {
    fmt.Printf("get(10) = Some(%d)\n", value)
} else {
    fmt.Println("get(10) = None")
}
```

Go 视角要点：

- **`append` 必须接返回值**。Rust 的 `push(&mut self)` 原地修改；Go 的 `append` 在容量不足时会分配新底层数组并返回新 slice。`append(s, x)` 不赋值是 Go 高频 bug。
- **越界都 panic**。Rust 额外提供 `Vec::get` 返回 `Option`；Go 用 `value, ok := s[i]` 或手动判断边界。`get` 辅助函数就是模拟这个模式。
- **排序**：Rust 用 `slice.sort()`（稳定）或 `sort_unstable()`；Go 1.21+ 用 `slices.Sort`，它**不保证稳定**，需要稳定排序用 `slices.SortStableFunc`。

---

## 例 2：String 与 UTF-8

Rust：

```rust
let mut text = String::from("hello");
text.push_str(" rust");
text.push('!');
println!("{text} | 大写: {}", text.to_uppercase());

println!("字符数: {}", "中文abc".chars().count()); // 5
println!("字节数: {}", "中文abc".len());            // 9
```

Go：

```go
text := "hello"
text += " rust"
text += "!"
fmt.Printf("%s | 大写: %s\n", text, strings.ToUpper(text))

fmt.Println("字符数:", utf8.RuneCountInString("中文abc"))
fmt.Println("字节数:", len("中文abc"))
```

Go 视角要点：

- Rust 的 `String` 可增长、可原地 `push_str`；Go 的 `string` **不可变**，`+=` 每次都会产生新字符串。频繁拼接用 `strings.Builder`。
- 两边 `len` 都是**字节数**。Rust 用 `.chars().count()` 数字符，Go 用 `utf8.RuneCountInString` 数 rune；直接 `range` 字符串时 Go 也按 rune 迭代（Rust 对应 `s.chars()`）。
- Rust 的 `char` 是 4 字节 Unicode 标量；Go 的 `rune` 是 `int32` 别名，概念相同。

---

## 例 3：HashMap 词频与排序

Rust：

```rust
let mut counts: HashMap<&str, u32> = HashMap::new();
for word in sentence.split_whitespace() {
    *counts.entry(word).or_insert(0) += 1;
}

let mut ranked: Vec<(&str, u32)> = counts.into_iter().collect();
ranked.sort_by(|a, b| b.1.cmp(&a.1).then(a.0.cmp(b.0)));
```

Go：

```go
counts := make(map[string]int)
for _, word := range strings.Fields(sentence) {
    counts[word]++
}

ranked := make([]wordCount, 0, len(counts))
for word, count := range counts {
    ranked = append(ranked, wordCount{word, count})
}
slices.SortFunc(ranked, func(a, b wordCount) int {
    return cmp.Or(
        cmp.Compare(b.count, a.count),
        cmp.Compare(a.word, b.word),
    )
})
```

Go 视角要点：

- **这是 Go 更简洁的地方**：map 的零值是 0，`m[k]++` 直接完成「不存在则插入 0，再 +1」。Rust 需要 `entry(k).or_insert(0)` 或 `get_mut`。
- **遍历顺序都随机**：Rust `HashMap` 和 Go `map` 都不保证顺序，输出前必须排序。
- **按 value 排序**：Go 的 map 不能直接排序，必须先转成 slice（这里是 `[]wordCount`）；Rust 也是先 `collect` 成 `Vec`。
- **多级排序**：Rust 的 `Ordering::then` 对应 Go 的 `cmp.Or`，都是「第一项非 0 就直接用，否则看下一项」。
- 只需要按 key 排序时，Go 1.23+ 可以写 `slices.Sorted(maps.Keys(counts))`。

---

## 例 4：迭代器链 ↔ for 循环

Rust：

```rust
let evens_squared: Vec<i32> = (1..=10)
    .filter(|n| n % 2 == 0)
    .map(|n| n * n)
    .collect();

let sum: i32 = (1..=100).sum();
let product: u64 = (1..=10).product();
let max = vec![3, 9, 2].into_iter().max(); // Option<i32>
```

Go：

```go
evensSquared := make([]int, 0, 5)
for n := 1; n <= 10; n++ {
    if n%2 == 0 {
        evensSquared = append(evensSquared, n*n)
    }
}

sum := 0
for n := 1; n <= 100; n++ {
    sum += n
}
product := 1
for n := 1; n <= 10; n++ {
    product *= n
}
max := slices.Max([]int{3, 9, 2}) // 空 slice 会 panic
```

Go 视角要点：

- Rust 链式调用是**惰性**的：`filter`/`map` 只构建适配器，`collect`/`sum` 才真正遍历。Go 的 for 循环是立即执行的。
- **Go 没有 `Iterator::map` 这样的链式标准库 API**。需要转换就写 `for + append`；这通常比硬凑链式调用更符合 Go 风格。
- `slices.Max` 在空 slice 上 panic；Rust 的 `max()` 返回 `Option`。这是两边错误模型差异的一个缩影。
- 如果确实想要惰性管道，Go 1.23+ 可以用 `iter.Seq`：

```go
func doubled(seq iter.Seq[int]) iter.Seq[int] {
    return func(yield func(int) bool) {
        for n := range seq {
            if !yield(n * 2) {
                return
            }
        }
    }
}

result := slices.Collect(doubled(slices.Values(numbers)))
```

但多数 Go 项目仍然偏好显式循环，可读性和调试体验更好。

---

## 例 5：zip / enumerate / join / fold

Rust：

```rust
for (index, (name, score)) in names.iter().zip(scores.iter()).enumerate() {
    println!("#{index}: {name} -> {score}");
}

let joined = names
    .iter()
    .map(|name| name.to_uppercase())
    .collect::<Vec<_>>()
    .join(", ");

let csv = scores.iter().fold(String::new(), |mut acc, score| {
    if !acc.is_empty() {
        acc.push(',');
    }
    acc.push_str(&score.to_string());
    acc
});
```

Go：

```go
for i := range names {
    fmt.Printf("#%d: %s -> %d\n", i, names[i], scores[i])
}

upperNames := make([]string, len(names))
for i, name := range names {
    upperNames[i] = strings.ToUpper(name)
}
fmt.Println("joined =", strings.Join(upperNames, ", "))

csvParts := make([]string, len(scores))
for i, score := range scores {
    csvParts[i] = strconv.Itoa(score)
}
fmt.Println("csv =", strings.Join(csvParts, ","))
```

Go 视角要点：

- **没有内置 zip**：Go 用下标同时访问两个 slice。Rust 的 `zip` 会在较短的一方结束；Go 手写时要自己处理长度不一致。
- **enumerate 就是 `for i, v := range`**，比 Rust 的 `.enumerate()` 更直接。
- **join**：Rust 先 `map` 成 `Vec<String>` 再 `join`；Go 先构造 `[]string`，再 `strings.Join`。
- **fold**：Go 没有泛型 fold，直接声明累加变量 + 循环。Rust 的 `fold` 优势是能把中间类型也泛化（比如 `String` 累加器）。

---

## 例 6：闭包与捕获

Rust：

```rust
let threshold = 5;
let above: Vec<i32> = (1..=10).filter(|n| *n > threshold).collect();

let mut calls = 0;
(1..=3).for_each(|_| calls += 1); // 需要 mut，闭包是 FnMut
```

Go：

```go
threshold := 5
above := make([]int, 0)
for n := 1; n <= 10; n++ {
    if n > threshold {
        above = append(above, n)
    }
}

calls := 0
increment := func() { calls++ }
for range 3 {
    increment()
}
```

Go 视角要点：

- Rust 闭包按需要实现 `Fn` / `FnMut` / `FnOnce`：只读捕获是 `Fn`，修改捕获变量需要变量本身是 `mut`，会移动捕获值的是 `FnOnce`。
- Go 的闭包**总是按引用捕获**外层变量，没有 `mut` 标记，也没有 `move` 关键字。
- Go 1.22 起，`for` 循环变量改为每轮独立；在此之前，goroutine 里直接捕获循环变量是经典坑。Rust 的 `move` 闭包则常用来把变量所有权交给线程/任务。

---

## 例 7：Option / Result ↔ Go 的缺失值与错误

| 场景 | Rust | Go |
| --- | --- | --- |
| 值可能不存在 | `Option<T>` | `(T, bool)`、`*T`、哨兵值 |
| 操作可能失败 | `Result<T, E>` | `(T, error)` |
| 提前返回 | `?` 运算符 | `if err != nil { return ..., err }` |
| 错误包装 | `thiserror` / `anyhow` / `source()` | `fmt.Errorf("...: %w", err)` |
| 空值 | 没有 null | `nil`（指针、接口、map、slice） |
| 空值检查 | `match` / `if let` / `unwrap_or` | 判 `nil` / comma-ok |

第 05 章的 `numbers.get(10)` 是最小例子：

```rust
// Rust：返回 Option，调用方必须处理 None
if let Some(value) = numbers.get(10) {
    println!("Some({value})");
} else {
    println!("None");
}
```

```go
// Go：value, ok 模式
if value, ok := get(numbers, 10); ok {
    fmt.Printf("Some(%d)\n", value)
} else {
    fmt.Println("None")
}
```

---

## 速查表

| Rust | Go 1.21+ |
| --- | --- |
| `Vec<T>` / `vec![]` | `[]T` / slice 字面量 |
| `v.push(x)` | `v = append(v, x)` |
| `v.get(i)` → `Option` | `v, ok := ...` / 边界检查 |
| `v.sort()` | `slices.Sort(v)` |
| `HashMap<K,V>` | `map[K]V` |
| `entry(k).or_insert(0)` | `m[k]++` |
| `iter().map(f).collect()` | `for` + `append` |
| `iter().filter(f)` | `if` + `append` |
| `iter().sum()` / `product()` | 累加循环 |
| `iter().max()` | `slices.Max`（空切片 panic） |
| `iter().zip(other)` | 下标循环（自行处理长度） |
| `iter().enumerate()` | `for i, v := range` |
| `iter().fold(init, f)` | 累加变量 + 循环 |
| `Vec<String>.join(", ")` | `strings.Join` |
| `String::push_str` | `strings.Builder` / `+=` |
| `Option<T>` | `(T, bool)` / `*T` |
| `Result<T,E>` + `?` | `(T, error)` + `if err != nil` |
| `FnMut` 闭包 + `mut` 变量 | 普通 `func` 闭包（引用捕获） |

## 给 Go 主栈的学习建议

- 先用 `for` 循环把逻辑写出来，再考虑 Rust 的 iterator 链；不要为了「像 Rust」而硬写长链。
- 三个高频差异：`append` 要接返回值、`String` 的所有权/可变性、`Option` 必须显式处理。
- 读 Rust 代码时，把 `iter().map().filter().collect()` 在脑子里翻译成「for + if + append」即可。
- 反过来写 Rust 时，优先用 `for x in &v` 而不是下标循环：越界检查和优化都更好。
- Go 的 `slices` / `maps` / `cmp` 包已经覆盖了很多以前要手写的逻辑，值得系统看一遍。
