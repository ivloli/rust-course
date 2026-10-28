# 第 09 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`。
除最后两个 scoped 线程的输出行可能互换外，其余输出逐行一致；验证时请排序后 diff。

```bash
# Rust
cd ~/rust-course && cargo run -p ch09_threads

# Go（标准库即可，无需额外依赖）
cd ~/rust-course/ch09_threads/go && go run .

# 用运行时检测数据竞争；本目录代码应当零报告
go run -race .
```

## 先记住三个总差异

1. **线程句柄**：Rust 的 `thread::spawn` 返回 `JoinHandle<T>`，`join()` 既等待结束又拿到 `T`；Go 的 `go` 语句没有返回值，需要 channel 或共享变量把结果带回来。
2. **共享状态**：Rust 用 `Arc<Mutex<T>>` 表达「共享所有权 + 互斥」；Go 的 goroutine 直接捕获指针/变量，只靠 `sync.Mutex` 保证互斥，所有权和生命周期不在编译期检查。
3. **作用域线程**：Rust 有 `thread::scope`，可以安全借用栈变量并由编译器保证 join；Go 没有 scoped goroutine，只能用 `sync.WaitGroup` 手动等待，正确性靠程序员保证。

---

## 例 1：spawn + join ↔ goroutine + channel

Rust：

```rust
let handle = thread::spawn(|| {
    for i in 1..=3 { println!("子线程: {i}"); thread::sleep(Duration::from_millis(10)); }
    42
});
let result = handle.join().expect("子线程 panic 了");
println!("子线程返回值 = {result}");
```

Go：

```go
done := make(chan int)
go func() {
    for i := 1; i <= 3; i++ { fmt.Printf("子线程: %d\n", i); time.Sleep(10 * time.Millisecond) }
    done <- 42
}()
result := <-done // 阻塞等待，相当于 join + 取返回值
fmt.Printf("子线程返回值 = %d\n", result)
```

Go 视角要点：

- **Go 没有 `JoinHandle`**。`go f()` 立刻返回，既不等待也不携带结果；最常用的替代是「结果 channel」或 `sync.WaitGroup`。
- **不要用 WaitGroup 传值**。WaitGroup 只负责计数同步，值要靠 channel、闭包写共享变量或原子操作带回。
- **panic 语义不同**：Rust 子线程 panic 后 `join()` 返回 `Err`，主线程还能继续；Go 里未 recover 的 goroutine panic 会让整个进程崩溃。`sleep(10ms)` 是 Rust 例子本来就有的，用于制造间隔，不是用来「制造顺序」。

---

## 例 2：move 闭包 ↔ 按引用捕获

Rust：

```rust
let data: Vec<i32> = (1..=4).collect();
let sum_handle = thread::spawn(move || data.iter().sum::<i32>());
println!("data 求和 = {}", sum_handle.join().unwrap());
// println!("{data:?}"); // data 已被移动到线程里
```

Go：

```go
data := []int{1, 2, 3, 4}
sumCh := make(chan int)
go func() {
    sum := 0
    for _, value := range data { sum += value }
    sumCh <- sum
}()
fmt.Printf("data 求和 = %d\n", <-sumCh)
// data 仍然可访问，编译器不会阻止
```

Go 视角要点：

- Rust 的 `move` 把 `data` 的所有权整体交给闭包，主线程之后访问会编译失败；这是 Rust 消除数据竞争的基础。Go 闭包**总是按引用捕获**，没有 `move`、没有 `'static`，主 goroutine 若继续写 `data` 就会形成数据竞争；`go run -race .` 是必要补充。
- 传大对象时 Go 只传指针，Rust 的 `move` 语义上确实换了所有者，且编译期禁止原变量再被使用。

---

## 例 3：Arc<Mutex<T>> ↔ 共享指针 + sync.Mutex

Rust：

```rust
let counter = Arc::new(Mutex::new(0));
for _ in 0..10 {
    let counter = Arc::clone(&counter);
    thread::spawn(move || {
        for _ in 0..1_000 { *counter.lock().unwrap() += 1; }
    });
}
println!("counter = {}", *counter.lock().unwrap());
```

Go：

```go
counter := 0
var mu sync.Mutex
var wg sync.WaitGroup
for range 10 {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for range 1_000 { mu.Lock(); counter++; mu.Unlock() }
    }()
}
wg.Wait()
fmt.Printf("counter = %d\n", counter)
```

Go 视角要点：

- **`Arc` 在 Go 里不存在也不需要**：所有 goroutine 捕获同一个变量（必要时是指针），共享所有权是默认行为。**临界区要手动配对 `Lock` / `Unlock`**；循环里不要用 `defer mu.Unlock()`，否则 defer 会堆积到函数返回，而 Rust 的 `MutexGuard` 离开作用域自动解锁。
- **锁中毒**：Rust 的 `Mutex` 在持锁线程 panic 后会 poisoned，下次 `lock()` 返回 `Err`；Go 的 `sync.Mutex` 没有中毒概念，通常整个进程已经崩了。10×1000 必须得到 `10000`；去掉锁再跑 `-race` 会看到 `WARNING: DATA RACE`。

---

## 例 4：mpsc channel ↔ Go channel

Rust：

```rust
let (tx, rx) = mpsc::channel();
for worker in 0..3 {
    let tx = tx.clone();
    thread::spawn(move || {
        thread::sleep(Duration::from_millis(worker * 10)); tx.send(format!("worker {worker} 完成")).unwrap();
    });
}
drop(tx); // 所有发送端关闭后，rx 循环才结束
for message in rx { println!("收到: {message}"); }
```

Go：

```go
messages := make(chan string)
var workers sync.WaitGroup
workers.Add(3)
send := func(worker int) {
    defer workers.Done()
    time.Sleep(time.Duration(worker) * 10 * time.Millisecond)
    messages <- fmt.Sprintf("worker %d 完成", worker)
}
for worker := 0; worker < 3; worker++ { go send(worker) }
go func() { workers.Wait(); close(messages) }()
for message := range messages { fmt.Printf("收到: %s\n", message) }
```

Go 视角要点：

- **channel 是语言内建类型**，不需要 `clone` 发送端；多个 goroutine 可以同时向同一个 channel 发送。**关闭规则**：只能由发送方 `close`，不能重复关闭；接收方用 `for range ch` 或 `v, ok := <-ch` 判断，Rust 则靠「所有 `Sender` 被 drop」自动结束。
- **容量语义**：`mpsc::channel()` 是无界队列；`make(chan string)` 是无缓冲 channel，发送会一直阻塞到有人接收。需要缓冲写 `make(chan string, 8)`。
- 输出顺序 `worker 0/1/2` 来自例子里的 `worker * 10ms` 延迟，两种语言都保留了这个 sleep；不要额外加 sleep 去「修」顺序。`worker` 是循环变量，Go 1.22 起每轮独立。

---

## 例 5：thread::scope ↔ goroutine + WaitGroup

Rust：

```rust
let mut shared = vec![];
thread::scope(|scope| {
    scope.spawn(|| println!("scoped 线程读取 shared.len() = {}", shared.len()));
    scope.spawn(|| println!("第二个 scoped 线程"));
});
shared.push(1);
println!("scope 结束后可以修改 shared = {shared:?}");
```

Go：

```go
shared := []int{}
var scoped sync.WaitGroup
scoped.Add(2)
go func() { defer scoped.Done(); fmt.Printf("scoped 线程读取 shared.len() = %d\n", len(shared)) }()
go func() { defer scoped.Done(); fmt.Println("第二个 scoped 线程") }()
scoped.Wait(); shared = append(shared, 1)
fmt.Printf("scope 结束后可以修改 shared = %v\n", shared)
```

Go 视角要点：

- **Go 没有借用检查版的 scoped thread**。goroutine 可以捕获 `shared`，编译器不验证它是否比变量活得久；如果忘了 `Wait`，`main` 返回后 goroutine 直接被终止。
- Rust 的 `scope` 在闭包返回前隐式 join 所有线程，因此 `shared` 的不可变借用安全结束，之后才能 `push`；这是编译期保证的。
- **两个 scoped 线程的打印顺序本身就不确定**，Rust 和 Go 都可能先打印任意一行。本次实测 Rust 有时先打印「第二个 scoped 线程」，Go 也一样。
- 验证方法：两边输出分别 `sort` 再 `diff`；除这两行外，前四个例子的输出顺序确定，应逐行一致。共享 slice 时注意 `append` 可能换底层数组，这里写入在 `Wait` 之后，没有竞争。

---

## 速查表

| Rust | Go |
| --- | --- |
| `thread::spawn(f)` → `JoinHandle<T>` | `go f()`（无返回值） |
| `handle.join()` | `<-ch` / `wg.Wait()` |
| `move` 闭包 | 闭包按引用捕获（无所有权） |
| `Arc<Mutex<T>>` | 共享变量 + `sync.Mutex` |
| `mpsc::channel()` + `drop(tx)` | `make(chan T)` + `close(ch)` |
| `thread::scope` | `sync.WaitGroup` + 共享闭包 |

## 给 Go 主栈的学习建议

- 把 `go f()` 理解为「启动后就不管」，需要结果时先想好 channel 或 WaitGroup，不要指望句柄。
- Rust 的 `Arc<Mutex<T>>` 在 Go 里就是「共享变量 + Mutex」；读 Rust 代码时重点看 `Arc::clone` 出现的次数，它对应有几个所有者。
- 写 Go 并发代码时，优先用 channel 传递数据，共享内存时务必让 `-race` 进 CI。
- Rust 的 `thread::scope` 很实用，短生命周期任务不必 `Arc`；Go 对应 WaitGroup + 约定不要越界。面试常问「Go 为什么没有 scoped goroutine」：GC 保证不悬空，但不保证逻辑并发安全。
