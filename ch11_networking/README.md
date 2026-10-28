# 11 TCP 与 HTTP 网络编程

```bash
cargo run -p ch11_networking                # 纯本地 TCP 演示，无需外网
cargo run -p ch11_networking -- --http      # 额外演示 reqwest 访问公网
```

> Go 主栈读者：每个例子的 Go 平行写法见 [GO_COMPARISON.md](GO_COMPARISON.md)，可运行代码在 `go/main.go`。（端口随机、客户端/服务端输出行序不确定，对比时需归一化端口并排序。）

## 学习目标

- 用 Tokio 写一个 TCP echo 服务器：accept → spawn → 读写
- 理解 TCP 是字节流，必须自己定义「消息边界」
- 用 `TcpStream::connect` 写客户端，并发的多个连接互不阻塞
- 用 `reqwest` 发起 HTTP 请求，知道 HTTP 是「带格式的 TCP 报文」

## 代码导读（src/main.rs）

1. `bind("127.0.0.1:0")` 让操作系统分配随机端口，避免端口冲突；`local_addr()` 取回实际端口。
2. 每个连接 `tokio::spawn` 一个任务：慢客户端不会阻塞 accept。
3. 服务端与客户端都用 `into_split()` 把读写半边拆开，再包 `BufReader` 按行读取。
4. 本章用「换行」作为消息边界（line framing）。真实协议通常用长度前缀（如 gRPC）或 HTTP 头。
5. `flush().await` 确保小报文立即发出（`BufWriter` 场景尤其重要）。
6. `reqwest::get` 只演示客户端；HTTP 服务端会在第 12 章用 Axum 实现。

## 为什么不用手写 TCP 做业务服务？

- HTTP/1.1、HTTP/2、TLS、keep-alive、超时、连接池、压缩都很复杂。
- Axum 基于 hyper/tower，把这些细节封装好，同时保留底层扩展点。
- 手写 TCP 适合自定义协议、消息队列、游戏服务器等场景。

## 动手练习

1. 把「按行」改成「4 字节大端长度前缀 + payload」，写一个二进制帧编解码。
2. 让服务器回显时加上时间戳与连接 ID（提示：`SystemTime`）。
3. 给客户端加连接超时：`tokio::time::timeout(Duration::from_secs(1), TcpStream::connect(addr))`。
4. 用 `cargo run -p ch11_networking -- --http` 发起真实 HTTP 请求，再用 `curl -v https://api.github.com/zen` 对比响应头。
5. 思考：10 万并发连接时，为什么「每连接一个线程」不可行，而 Tokio 可行？

## 延伸

- 抓包：`tcpdump -i lo0 -A 'tcp port <port>'`（macOS 需要 sudo）
- 下一步：第 12 章用 Axum 构建真正的 HTTP 服务
