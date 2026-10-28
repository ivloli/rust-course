# 09 线程与共享状态（std）

```bash
cargo run -p ch09_threads
```

> Go 主栈读者：每个例子的 Go 平行写法见 [GO_COMPARISON.md](GO_COMPARISON.md)，可运行代码在 `go/main.go`。（两个 scoped 线程的输出顺序不确定，Rust 与 Go 都可能互换；对比时排序即可。）

## 学习目标

- `thread::spawn` / `join`：创建与等待线程，拿到返回值
- `move` 闭包把所有权交给线程，理解 `'static` 约束
- `Arc<Mutex<T>>`：跨线程共享可变状态
- `mpsc` channel：消息传递优于共享内存的场景
- `thread::scope`：借用栈上数据的线程

## 代码导读（src/main.rs）

1. `spawn` 返回 `JoinHandle<T>`，`join()` 阻塞直到线程结束。
2. `move` 闭包会拿走 `data`；如果线程活得比 `main` 长，编译器不允许闭包借用栈变量。
3. 10 个线程各加 1000 次：`Arc` 提供共享所有权，`Mutex` 提供互斥访问。
4. `lock()` 返回 `MutexGuard`，离开作用域自动释放——不会忘记 unlock。
5. `drop(tx)` 很关键：所有发送端都被丢弃后，`for message in rx` 才会结束。
6. 两个 scoped 线程都可以不可变借用 `shared`；scope 结束前编译器保证线程已 join。

## 并发安全从哪来？

- `Send`：类型可以安全地跨线程转移所有权（`String`、`Vec` 是，`Rc` 不是）。
- `Sync`：类型可以安全地被多个线程同时引用（`Mutex<T>` 是，`Cell` 不是）。
- Rust 在编译期检查这两个 marker trait，数据竞争在编译阶段就被排除。

## 动手练习

1. 把 `Mutex` 换成 `AtomicUsize`，用 `fetch_add` 重写计数器，比较性能与代码。
2. 用两个 channel 实现 ping-pong：主线程发数字，工作线程平方后发回。
3. 让一个线程 panic，观察 `join()` 返回 `Err` 与 `Mutex` 中毒（poisoned）现象。
4. 构造一个死锁：两个线程以相反顺序锁两个 Mutex，解释为什么编译器不会帮你发现它。
5. 思考：为什么 Tokio 的异步任务里不能直接使用 `std::sync::Mutex` 跨 `await` 持有锁？

## 工程提醒

- 锁的临界区尽量短；不要在持锁时做 I/O 或调用外部服务。
- 共享内存 + 锁在服务端很容易成为瓶颈；优先考虑消息传递或分片。
- 线程池 / 异步运行时（第 10 章）才是高并发网络服务的默认选择。
