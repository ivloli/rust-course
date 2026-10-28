# 第 10 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`。输出行结构一致，但**耗时数值必然不同**；验证时只看行序、结果和量级。

```bash
# Rust
cd ~/rust-course && cargo run -p ch10_async_tokio

# Go（只用标准库）
cd ~/rust-course/ch10_async_tokio/go && go run .
go run -race .   # 数据竞争检测，应当零报告
```

## 先记住三个总差异

1. **惰性 vs 立即**：Rust 的 `async fn` 返回 Future，创建时不执行，`.await` 才驱动；Go 的 `go f()` 一执行就立刻调度，没有「未启动的任务」。
2. **运行时**：Tokio 是库级运行时，`#[tokio::main]` 建多线程 executor；Go 的 GMP 调度器内建在语言运行时里。
3. **取消与锁**：Rust 超时会 drop Future，清理放在 `Drop`；Go 的 `context` 取消是协作式的。Rust 需要 `tokio::sync::Mutex` 才能跨 `await` 持锁；Go 只有阻塞式 `sync.Mutex`，因为阻塞 goroutine 很便宜。

## 例 1：惰性 Future ↔ 函数值 / goroutine

Rust：

```rust
let future = say_hello();
println!("Future 已创建，但尚未执行");
future.await;
```

Go：

```go
future := sayHello // 函数值：只是「还没调用」
fmt.Println("Future 已创建，但尚未执行")
future()
```

Go 视角要点：

- **Go 没有 Future**。`future := sayHello` 只是保存函数，`future()` 才同步执行；用函数值模拟「已创建、未执行」才能保持相同行序。Rust Future 是状态机，通常不单独分配栈；goroutine 有可增长的小栈（初始约 2KB）。
- 真正的 Go 写法是 `go sayHello()`：goroutine **立刻开始**，与 `println` 的先后不确定。惰性的价值在于超时/取消/组合可以在启动前完成；goroutine 一旦启动只能靠 channel 或 context 协调。

## 例 2：tokio::join! ↔ goroutine + WaitGroup

Rust：

```rust
let (a, b) = tokio::join!(work("A", 200), work("B", 300));
println!("join! 完成: {a} / {b}，耗时 {:?}", started.elapsed());
```

Go：

```go
results := make([]string, 2)
var wg sync.WaitGroup; wg.Add(2)
go func() { defer wg.Done(); results[0] = work(context.Background(), "A", 200) }()
go func() { defer wg.Done(); results[1] = work(context.Background(), "B", 300) }()
wg.Wait()
fmt.Printf("join! 完成: %s / %s，耗时 %v\n", results[0], results[1], time.Since(started).Round(time.Millisecond))
```

Go 视角要点：

- `join!` 是**结构化并发**：两个 Future 在同一个任务里被并发轮询，不创建新任务；Go 直接开两个 goroutine，用 WaitGroup 汇合。两个 goroutine 写 `results` 的不同下标是安全的；若写同一变量就需要锁或 channel。耗时约 300ms（取最大值），不是 500ms。
- `join!` 可返回不同类型的元组；Go 的 slice 元素类型必须一致，类型不同就用各自的 channel 或 struct。时间数字不要逐字比较。

## 例 3：顺序 await ↔ 顺序调用

Rust：

```rust
work("C", 100).await;
work("D", 100).await;
println!("顺序执行耗时 {:?}", started.elapsed());
```

Go：

```go
work(context.Background(), "C", 100); work(context.Background(), "D", 100)
fmt.Printf("顺序执行耗时 %v\n", time.Since(started).Round(time.Millisecond))
```

Go 视角要点：

- 两边都是**串行**，总耗时约 200ms。Rust 的 `.await` 会让出执行权；Go 的 `work` 阻塞当前 goroutine，但运行时会把这个 M 上的其他 G 调度走。**Go 的「阻塞」不等于卡死整个程序**，只有 CPU 密集且不让出时才会占满 P。
- Tokio 里阻塞 worker 线程会影响同一线程上的其他任务；CPU 密集计算时 Go 可用 `runtime.Gosched` 或限制并行度，Tokio 用 `spawn_blocking`。本例两行没有并发，输出顺序稳定。

## 例 4：tokio::spawn ↔ go + channel

Rust：

```rust
let handle = tokio::spawn(async move { 6 * 7 });
println!("spawn 结果 = {}", handle.await.unwrap());
```

Go：

```go
spawned := make(chan int, 1)
go func() { spawned <- 6 * 7 }()
fmt.Printf("spawn 结果 = %d\n", <-spawned)
```

Go 视角要点：

- `tokio::spawn` 立刻把任务交给运行时并返回 `JoinHandle`；Go 的 `go` 同样立刻调度，但没有句柄，结果要靠 channel 带回。Rust 要求 Future 满足 `Send + 'static`，编译器阻止借用栈变量；Go 的 goroutine 可以捕获任何东西，生命周期和竞争由程序员负责。
- Rust 的 `JoinHandle.await` 会以 `Err` 传播 panic；Go 未 recover 的 goroutine panic 会终止整个进程。

## 例 5：timeout ↔ context.WithTimeout

Rust：

```rust
match tokio::time::timeout(Duration::from_millis(100), work("慢任务", 500)).await {
    Ok(value) => println!("慢任务完成: {value}"),
    Err(_) => println!("慢任务 100ms 超时，已取消"),
}
```

Go：

```go
ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond); defer cancel()
if _, err := workCtx(ctx, "慢任务", 500); err != nil { fmt.Println("慢任务 100ms 超时，已取消") }
```

Go 视角要点：

- **取消模型完全不同**：Rust 的 `timeout` 在 100ms 时 drop 内部 Future，`Drop` 里的清理立即执行；Go 只是关闭 `ctx.Done()`，`workCtx` 必须 `select` 到它才会返回。若被调用函数忽略 context，Go 会跑完 500ms，**没有强制取消**。
- `defer cancel()` 必须调用，否则 timer 和 context 资源要等到超时才释放。输出固定是「慢任务 100ms 超时，已取消」。

## 例 6：tokio::sync::mpsc ↔ 带缓冲 channel

Rust：

```rust
let (tx, mut rx) = mpsc::channel::<u32>(8);
tokio::spawn(async move {
    for i in 1..=5 {
        tx.send(i).await.unwrap(); tokio::time::sleep(Duration::from_millis(20)).await;
    }
});
while let Some(value) = rx.recv().await { print!("{value} "); }
```

Go：

```go
values := make(chan int, 8)
go func() {
    defer close(values)
    for i := 1; i <= 5; i++ {
        values <- i; time.Sleep(20 * time.Millisecond)
    }
}()
for value := range values { fmt.Printf("%d ", value) }
fmt.Println()
```

Go 视角要点：

- **两者都有背压**：缓冲满时，Rust 的 `send().await` 挂起任务，Go 的 `values <- i` 阻塞 goroutine。容量 8 大于 5，本例不会触发背压；改成 1 就能观察。Go 用 `close(values)` 表示结束，Rust 靠所有 `Sender` 被 drop 后 `recv()` 返回 `None`。
- 输出是 `1 2 3 4 5 `（末尾有空格再换行），和 Rust 的 `print!` 写法一致。

## 例 7：tokio::sync::Mutex ↔ sync.Mutex

Rust：

```rust
let shared = Arc::new(Mutex::new(0));
for _ in 0..5 {
    let shared = Arc::clone(&shared);
    tokio::spawn(async move {
        let mut guard = shared.lock().await;
        tokio::time::sleep(Duration::from_millis(10)).await; *guard += 1;
    });
}
```

Go：

```go
shared := 0
var mu sync.Mutex
var handles sync.WaitGroup
handles.Add(5)
for range 5 {
    go func() {
        mu.Lock(); time.Sleep(10 * time.Millisecond)
        shared++; mu.Unlock(); handles.Done()
    }()
}
handles.Wait(); fmt.Printf("shared = %d\n", shared)
```

Go 视角要点：

- **Go 没有 async mutex，也不需要**。goroutine 阻塞在 `mu.Lock()` 时，运行时会把 M 上的其他 G 调度走；`tokio::sync::Mutex` 存在是因为不能阻塞 executor 的 worker 线程。**没有「取消等待者」语义**：Rust 的任务在等待锁时被 drop 会退出队列；Go 的 `Lock` 会一直等。
- 临界区里的 `sleep(10ms)` 是故意的：5 个 goroutine 串行持锁，总耗时约 50ms。结果必须是 `shared = 5`；去掉锁跑 `go run -race .` 会报告数据竞争。

## 速查表

| Rust | Go |
| --- | --- |
| `async fn` → `Future`（惰性） | `func` / `go func()`（立即） |
| `.await` | 直接调用 / `<-ch` / `wg.Wait()` |
| `tokio::join!` | 多个 goroutine + WaitGroup |
| `tokio::time::timeout` | `context.WithTimeout` + `select` |
| `tokio::sync::Mutex` | `sync.Mutex` |

## 给 Go 主栈的学习建议

- 先把 goroutine 版本写出来，再读 Rust 的 async：把每个 `.await` 想成「这里可能让出」，把 `join!` 想成「WaitGroup 汇合」。
- 最重要的心智差异是**惰性**：Future 创建后不运行，goroutine 创建后立即运行；这决定了取消、超时和错误处理的位置。
- 验证时不要比较耗时数字，只比较输出行、A/B 约 300ms、顺序执行约 200ms、超时行为和 `shared = 5`，并保持 `-race` 干净。
