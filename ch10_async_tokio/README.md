# 10 异步编程与 Tokio

```bash
cargo run -p ch10_async_tokio
```

## 学习目标

- 理解 `async fn` 返回 Future，Future 是惰性的
- `join!` 并发、`spawn` 并行调度、`timeout` 取消
- 异步 channel 与 `tokio::sync::Mutex`
- 知道「不要在异步任务里阻塞」以及 `spawn_blocking` 的用途

## 代码导读（src/main.rs）

1. `let future = say_hello();` 不会打印任何东西；`.await` 才驱动执行。
2. `join!` 在同一个任务里并发等待，适合「同时请求多个下游」；顺序 await 则是串行。
3. `spawn` 相当于提交给运行时线程池，闭包必须 `Send + 'static`。
4. `timeout` 返回 `Result<T, Elapsed>`；超时后内部 Future 被 drop，所以清理逻辑要放在 Drop 里。
5. `tokio::sync::mpsc` 的 `send().await` 在缓冲区满时会等待，天然具备背压（backpressure）。
6. 第 7 节故意在持锁期间 `sleep`：`tokio::sync::Mutex` 允许跨 `await` 持有，但吞吐会下降，生产代码应缩小临界区。

## Rust async 的心智模型

- Rust 的 `async` 是「零抽象」：编译成状态机，不需要为每个 Future 分配栈。
- 运行时（executor）负责轮询；`#[tokio::main]` 会创建多线程运行时。
- `std::sync::Mutex` 可以跨线程但不能跨 `await` 长期持有；`tokio::sync::Mutex` 可以，但更慢。
- CPU 密集任务用 `tokio::task::spawn_blocking`，否则会阻塞整个 worker 线程。

## 动手练习

1. 用 `tokio::try_join!` 改写 `join!`，让任一 Future 失败时立即返回。
2. 用 `tokio::select!` 实现「两个 channel 谁先来消息就处理谁」。
3. 用 `spawn_blocking` 包装一个 `std::thread::sleep`，观察不再阻塞其他任务。
4. 把 `mpsc::channel(8)` 改成 `channel(1)`，观察发送端的等待行为。
5. 思考：为什么 Web 框架的 handler 必须是 `Send + 'static`（或借用受限于请求生命周期）？

## 与第 09 章对照

| 维度 | 线程 | 异步任务 |
| --- | --- | --- |
| 调度 | OS 抢占式 | 运行时协作式 |
| 栈 | 每线程独立栈（MB 级） | 状态机（KB 级） |
| 适合 | CPU 密集、阻塞 I/O | 高并发网络 I/O |
| 共享状态 | `Arc<Mutex>` | `Arc<tokio::Mutex>` / actor/channel |
