# 第 11 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`。
两边都是「监听随机端口 → 3 个并发客户端各发两行 → 回显 → 可选 HTTP 请求」，
但**端口随机、客户端与服务端的输出行序不确定**，验证时必须归一化端口并排序。

```bash
# Rust
cd ~/rust-course && cargo run -p ch11_networking
cargo run -p ch11_networking -- --http   # 可选，需要外网

# Go（只用标准库）
cd ~/rust-course/ch11_networking/go && go run .
go run . --http   # 可选，需要外网
go run -race .    # 数据竞争检测
```

## 先记住三个总差异

1. **并发模型**：Rust 用 Tokio 的 `TcpListener` + `tokio::spawn`，每个连接是一个异步任务；Go 用 `net.Listen` + 「每连接一个 goroutine」，代码是同步阻塞风格，但连接之间互不阻塞。
2. **I/O 抽象**：Rust 需要 `AsyncRead` / `AsyncWrite`、`BufReader`、`into_split()` 和 `.await`；Go 的 `net.Conn` 同时实现 `io.Reader` / `io.Writer`，包一层 `bufio` 就能按行读写。
3. **错误与清理**：Rust 用 `Result` + `?`，连接关闭靠 `Drop`；Go 用 `(value, error)` + `if err != nil`，连接关闭靠 `defer conn.Close()`。

## 例 1：绑定随机端口

Rust：

```rust
let listener = TcpListener::bind("127.0.0.1:0").await?;
let addr = listener.local_addr()?;
println!("echo 服务器监听 {addr}");
```

Go：

```go
listener, err := net.Listen("tcp", "127.0.0.1:0")
if err != nil { log.Fatal(err) }
addr := listener.Addr().String()
fmt.Printf("echo 服务器监听 %s\n", addr)
```

Go 视角要点：

- `:0` 在两边都表示「让内核分配端口」；Rust 用 `local_addr()`，Go 用 `listener.Addr()` 取回实际地址。
- 打印出的端口每次运行都不同，所以对比输出前要把 `127.0.0.1:\d+` 归一化。
- Go 的 `net.Listen` 第一个参数是网络类型 `"tcp"`；Rust 由 `TcpListener` 类型隐含。需要 IPv6 时 Go 写 `"tcp6"` 或直接 `"[::1]:0"`。
- `log.Fatal` 会直接退出；演示程序可以接受，库代码应把 error 返回给调用方。

## 例 2：accept 循环与每连接 goroutine

Rust：

```rust
tokio::spawn(async move {
    loop {
        let (socket, peer) = listener.accept().await?;
        println!("新连接来自 {peer}");
        tokio::spawn(async move {
            if let Err(error) = handle_connection(socket).await {
                eprintln!("连接 {peer} 出错: {error}");
            }
        });
    }
});
```

Go：

```go
go func() {
    for {
        conn, err := listener.Accept()
        if err != nil { return }
        peer := conn.RemoteAddr().String()
        fmt.Printf("新连接来自 %s\n", peer)
        go handleConnection(conn, peer)
    }
}()
```

Go 视角要点：

- 两者都是「accept 一个连接就派生一个执行单元」：Rust 是 `tokio::spawn`，Go 是 `go`。慢客户端不会阻塞后续 accept。
- **Go 的代码看起来是阻塞的**，但 goroutine 阻塞在 `Read` 时，运行时会调度其他 goroutine；Tokio 则必须在 `.await` 点让出。
- 监听循环本身放在 goroutine 里，`main` 不必等它；客户端结束后 `main` 返回，Go 进程会终止所有 goroutine。演示里另用 `handlerDone` 等 3 个连接处理完，保证「客户端断开」打印完整。
- Rust 例子在 `accept` 出错时用 `?` 退出任务；Go 这里直接 `return`。真实服务通常要区分临时错误并重试。

## 例 3：按行读取与回显

Rust：

```rust
let (reader, mut writer) = socket.into_split();
let mut reader = BufReader::new(reader);
let mut line = String::new();
while reader.read_line(&mut line).await? > 0 {
    print!("服务器收到: {line}");
    writer.write_all(line.as_bytes()).await?;
    writer.flush().await?;
    line.clear();
}
```

Go：

```go
scanner := bufio.NewScanner(conn)
for scanner.Scan() {
    line := scanner.Text()
    fmt.Printf("服务器收到: %s\n", line)
    fmt.Fprintf(conn, "%s\n", line)
}
if err := scanner.Err(); err != nil { /* 记录错误 */ }
```

Go 视角要点：

- **TCP 是字节流**，两边都用「换行」做消息边界（line framing）。Rust 的 `read_line` 保留 `\n`，Go 的 `Scanner.Text()` 去掉换行，所以回写时要补 `\n`。
- Rust 需要 `into_split()` 取得读、写两半；Go 的 `net.Conn` 本身既能读也能写，不需要拆分。
- **flush**：Go 的 `net.Conn` 是无缓冲的，`fmt.Fprintf` 直接写内核；Rust 这里虽然也是直接写，但显式 `flush().await` 更稳妥，换成 `BufWriter` 时必须要。
- `bufio.Scanner` 默认单行上限 64KB；超长行要 `scanner.Buffer` 调大，或改用 `bufio.Reader.ReadString('\n')`。

## 例 4：3 个并发客户端

Rust：

```rust
let mut clients = Vec::new();
for client_id in 0..3 {
    clients.push(tokio::spawn(async move {
        echo_client(addr, client_id).await.unwrap();
    }));
}
for client in clients { client.await.unwrap(); }
```

Go：

```go
var clients sync.WaitGroup
for clientID := 0; clientID < 3; clientID++ {
    clients.Add(1)
    go func() {
        defer clients.Done()
        if err := echoClient(addr, clientID); err != nil {
            fmt.Fprintf(os.Stderr, "客户端 %d 出错: %v\n", clientID, err)
        }
    }()
}
clients.Wait()
```

Go 视角要点：

- Rust 收集 `JoinHandle` 再逐个 await；Go 用 `WaitGroup` 等待全部完成。两者都保证 3 个客户端结束后才走下一步。
- Go 1.22 起循环变量每轮独立，`clientID` 可以直接被 goroutine 捕获；更早版本要写 `clientID := clientID`。
- 每个客户端做两轮「写一行、读一行」，所以一共 6 次请求/回显；服务端日志与客户端日志交错，顺序不可预测。
- 客户端用 `net.Dial("tcp", addr)`，对应 `TcpStream::connect`；`bufio.Reader.ReadString('\n')` 保留换行，打印格式和 Rust 的 `read_line` 一致。

## 例 5：可选 HTTP 请求

Rust：

```rust
let response = reqwest::get("https://api.github.com/zen").await?;
println!("HTTP 状态: {}", response.status());
println!("响应: {}", response.text().await?.trim());
```

Go：

```go
client := &http.Client{Timeout: 10 * time.Second}
response, err := client.Get("https://api.github.com/zen")
if err != nil { /* 打印失败信息并返回 */ }
defer response.Body.Close()
body, _ := io.ReadAll(response.Body)
fmt.Printf("HTTP 状态: %s\n", response.Status)
fmt.Printf("响应: %s\n", strings.TrimSpace(string(body)))
```

Go 视角要点：

- `reqwest::get` 一行完成请求；Go 用 `http.Get` 或带超时的 `http.Client`。生产代码应始终设置超时，避免永久挂起。
- Rust 的 `StatusCode` Display 和 Go 的 `resp.Status` 都是 `200 OK` 形式；`response.text().await?.trim()` 对应 `io.ReadAll` + `strings.TrimSpace`。
- **必须 `defer response.Body.Close()`**，否则连接无法复用；Rust 的 `reqwest::Response` 在 drop 时处理。
- 没有 `--http` 时两边都打印提示：`提示：加 --http 参数可以演示 reqwest 请求外部 HTTP 服务`。Go 的 URL 是写死的，不接受用户输入。

## 验证与注意事项

```bash
# 两边都归一化端口并排序后比较
cargo run -q -p ch11_networking | sed -E 's/127\.0\.0\.1:[0-9]+/ADDR/g' | sort > /tmp/rust.txt
(cd ch11_networking/go && go run .) | sed -E 's/127\.0\.0\.1:[0-9]+/ADDR/g' | sort > /tmp/go.txt
diff /tmp/rust.txt /tmp/go.txt
```

- 应包含：1 行监听、3 行新连接、6 行「服务器收到」、6 行「客户端收到回声」、3 行「客户端断开」、1 行提示。
- 端口、连接顺序、客户端/服务端日志的交错都不确定；**不要直接 diff 原始输出**。
- Go 版本在 3 个客户端和 3 个 handler 都结束后才打印提示并返回；监听 goroutine 仍阻塞在 `Accept`，随进程退出而消失，不需要额外关闭。
- `go run -race .` 应无数据竞争；服务端每个连接只被自己的 goroutine 读写，客户端各自持有自己的连接。

## 速查表

| Rust | Go |
| --- | --- |
| `TcpListener::bind("127.0.0.1:0")` | `net.Listen("tcp", "127.0.0.1:0")` |
| `listener.accept().await` | `listener.Accept()` |
| `tokio::spawn` | `go` |
| `BufReader::read_line` | `bufio.Scanner` / `ReadString('\n')` |
| `reqwest::get` | `http.Get` / `http.Client` |
| `Result` + `?` | `(value, error)` + `if err != nil` |

## 给 Go 主栈的学习建议

- 把 Tokio 版本理解成「Go 的 goroutine-per-connection，只是每个阻塞点都变成 `.await`」；读 Rust 代码时先找 `spawn` 和 `await`。
- Go 的同步写法更直观，但要注意 `bufio.Scanner` 的行长度上限、`defer Close` 和 HTTP 超时；Rust 的 `?` 会向上传播错误，Go 要显式检查，生产代码应带上下文包装（`fmt.Errorf("...: %w", err)`）。
- 验证网络程序时永远先归一化随机端口和并发顺序，再比较「行集合」而不是「行序列」。
