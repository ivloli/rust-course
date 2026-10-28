# 第 13 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`。**响应状态、CORS 头、`x-response-time-ms` 与优雅关闭行为一致**，差异主要在中间件的写法。

```bash
# Rust
cd ~/rust-course && RUST_LOG=debug cargo run -p ch13_axum_advanced

# Go（需要 Go 1.22+）
cd ~/rust-course/ch13_axum_advanced/go && go run .
```

两边都读 `PORT`，默认 3000，绑定 `127.0.0.1`。

## 先记住三个总差异

1. **中间件**：tower 的 `.layer()` 是洋葱模型，后加的层在更外层；Go 用 `func(http.Handler) http.Handler` 一层层手工包，最外层最后包。
2. **超时与取消**：Rust 用 `TimeoutLayer` 直接丢弃 future；Go 用 `context.WithTimeout` + goroutine，必须自己缓冲响应，否则超时后 handler 再写会冲突。
3. **错误与日志**：`IntoResponse for AppError` 对应 `AppError` + `writeAppError` 辅助函数；`tracing` 对应标准库 `log`。

## 例 1：/ 与 /health —— 路由与 layer 顺序

Rust：
```rust
let app = Router::new()
    .route("/", get(root))
    .route("/health", get(health))
    .layer(TimeoutLayer::with_status_code(StatusCode::REQUEST_TIMEOUT, Duration::from_millis(500)))
    .layer(middleware::from_fn(log_elapsed))
    .layer(CorsLayer::new() /* ... */)
    .layer(TraceLayer::new_for_http());
```
Go：
```go
var handler http.Handler = timeoutMiddleware(500 * time.Millisecond)(mux)
handler = logElapsed(handler)
handler = cors(handler)
handler = trace(handler) // trace 最外层，timeout 最内层
```
- Rust 的 layer 从下往上包：Trace 最外、Timeout 最内。Go 的赋值顺序完全对应，**最后赋值的 trace 最外层**。
- 层序很关键：CORS 在 `log_elapsed` 外，所以预检响应没有 `x-response-time-ms`；Timeout 在内，所以 408 仍会经过耗时统计。

## 例 2：/slow —— 2 秒 handler 与可取消等待

Rust：
```rust
async fn slow() -> &'static str {
    tokio::time::sleep(Duration::from_secs(2)).await;
    "终于完成了"
}
```
Go：
```go
select {
case <-time.After(2 * time.Second):
    writeText(w, http.StatusOK, "终于完成了")
case <-r.Context().Done():
    return // 超时后 context 被取消，不再写响应
}
```
- Rust 的 future 被 drop 时 `sleep` 自动取消；Go 的 `time.After` 不会自动取消，必须 `select` 监听 `ctx.Done()`，否则 goroutine 会白等 2 秒。
- 500ms 内不会走到成功分支；两边客户端都在约 500ms 收到 408 空 body。

## 例 3：500ms 超时 —— 408 而不是 503

Rust：
```rust
.layer(TimeoutLayer::with_status_code(
    StatusCode::REQUEST_TIMEOUT, Duration::from_millis(500)))
```
Go：
```go
buf := newBufferedResponse()
done := make(chan struct{})
go func() { defer close(done); next.ServeHTTP(buf, r.WithContext(ctx)) }()
select {
case <-done:
    buf.flushTo(w)
case <-ctx.Done():
    if errors.Is(ctx.Err(), context.DeadlineExceeded) {
        w.WriteHeader(http.StatusRequestTimeout) // 408 空 body
    }
}
```
- **不要用 `http.TimeoutHandler`**：它默认返回 503，且响应头不可控。自定义中间件才能精确复刻 `TimeoutLayer` 的 408。
- **缓冲 ResponseWriter 是核心**：handler 超时后可能继续写，直接写真实 `w` 会触发 “superfluous WriteHeader”；写进 buffer 则安全丢弃。
- `done` 用无缓冲 channel 即可；handler 完成后关闭，主 goroutine 只在 `done` 关闭后读取 buffer，不存在数据竞争。
- 客户端断开时 `r.Context()` 也会取消，Go 版本只在 `DeadlineExceeded` 时写 408。

## 例 4：x-response-time-ms —— log_elapsed

Rust：
```rust
async fn log_elapsed(request: Request, next: Next) -> Response {
    let started = Instant::now();
    let mut response = next.run(request).await;
    response.headers_mut().insert("x-response-time-ms",
        HeaderValue::from_str(&started.elapsed().as_millis().to_string()).unwrap());
    response
}
```
Go：
```go
started := time.Now()
buffer := newBufferedResponse()
next.ServeHTTP(buffer, r)
buffer.header.Set("x-response-time-ms",
    strconv.FormatInt(time.Since(started).Milliseconds(), 10))
buffer.flushTo(w)
```
- 这个中间件也要**先缓冲再提交**：408 是在内层写出的，若直接透传，等它返回时响应头已经发出，再加 `x-response-time-ms` 就无效了。
- Rust 的 `Instant` 对应 `time.Now()`，`as_millis()` 对应 `Milliseconds()`；`/slow` 两边都显示约 500。
- 日志内容对应 `GET /slow -> 408 in 501ms`，Go 用 `log.Printf` 打到 stderr。

## 例 5：/fail —— AppError 与统一 JSON

Rust：
```rust
enum AppError { NotFound(String), Internal(String) }

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        (StatusCode::NOT_FOUND, Json(json!({ "error": "资源不存在" }))).into_response()
    }
}
```
Go：
```go
type AppError struct{ Status int; Message string }
func (e *AppError) Error() string { return e.Message }

func writeAppError(w http.ResponseWriter, err *AppError) {
    writeJSON(w, err.Status, map[string]string{"error": err.Message})
}
```
- Rust 靠 trait 让 handler 返回 `Result<T, AppError>`；Go 没有 `IntoResponse`，用显式类型 + 辅助函数达到同样效果。
- `/fail` 两边都是 404 + `{"error":"资源不存在"}`，Content-Type 都是 `application/json`。
- Rust 还能 `impl From<io::Error>` 做错误转换；Go 用 `fmt.Errorf("...: %w", err)` 包装，再在边界映射成 `AppError`。

## 例 6：CORS 与 OPTIONS 预检

Rust：
```rust
CorsLayer::new()
    .allow_origin(Any)
    .allow_methods([Method::GET, Method::POST])
    .allow_headers([header::CONTENT_TYPE])
```
Go：
```go
header.Set("Vary", "origin, access-control-request-method, access-control-request-headers")
header.Set("Access-Control-Allow-Origin", "*")
if r.Method == http.MethodOptions {
    header.Set("Access-Control-Allow-Methods", "GET,POST")
    header.Set("Access-Control-Allow-Headers", "content-type")
    w.WriteHeader(http.StatusOK)
    return
}
```
- 两边普通响应都带 `access-control-allow-origin: *` 和 `vary`；Go 的 header 名在线上会规范化成 `Access-Control-Allow-Origin`，大小写不敏感。
- 预检：Rust 的 CorsLayer 对所有 OPTIONS 直接返回 200，Go 的 cors 中间件也直接短路，**不会进入 mux，也不会经过 log_elapsed**，所以预检没有 `x-response-time-ms`。
- 生产环境不要无条件 `*`；需要 cookie 时不能用通配符，应配置明确来源和 credentials。

## 例 7：Trace 日志

Rust：
```rust
.layer(TraceLayer::new_for_http())
// RUST_LOG=debug 时打印 on_request / on_response
```
Go：
```go
log.Printf("started %s %s", r.Method, r.URL.RequestURI())
recorder := &statusRecorder{ResponseWriter: w}
next.ServeHTTP(recorder, r)
log.Printf("finished %s %s -> %d", r.Method, r.URL.RequestURI(), recorder.statusCode())
```
- `TraceLayer` 是结构化 span；Go 标准库 `log` 只有文本，生产项目通常换 `log/slog` 或 zap。
- `statusRecorder` 只记录状态码并透传，不缓冲；`WriteHeader` 只记第一次，避免重复写。

## 例 8：优雅关闭与 PORT

Rust：
```rust
axum::serve(listener, app)
    .with_graceful_shutdown(shutdown_signal())
    .await?;
// shutdown_signal() 等待 tokio::signal::ctrl_c()
```
Go：
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
go func() { serveErr <- server.Serve(listener) }()
<-ctx.Done()
server.Shutdown(context.Background()) // 停止接收新连接，等待在途请求
```
- Rust 只监听 Ctrl-C；Go 版本同时监听 SIGINT 和 SIGTERM，容器/K8s 部署更实用。
- 两边都先停止接收新连接，再等在途请求完成；Go 验证 `SIGINT`/`SIGTERM` 退出码都是 0。
- `http.ErrServerClosed` 是 `Shutdown` 后的正常返回值，不能当错误；Rust 的 `axum::serve` 则直接返回 `Ok`。
## 速查表

| Rust / Axum / tower | Go 标准库 |
| --- | --- |
| `Router::route` | `ServeMux.HandleFunc("GET /x")` |
| `.layer(TimeoutLayer)` | 自定义 `timeoutMiddleware` + `context` |
| `.layer(CorsLayer)` | `cors` 中间件 + OPTIONS 短路 |
| `.layer(TraceLayer)` | `trace` 中间件 + `log` |
| `middleware::from_fn` | `func(http.Handler) http.Handler` |
| `IntoResponse` | `AppError` + `writeAppError` |
| 408 + 空 body | `WriteHeader(408)`，自定义缓冲 |
| `with_graceful_shutdown` | `signal.NotifyContext` + `Server.Shutdown` |
| `tokio::spawn` / future drop | goroutine + `context` 取消 |
## 给 Go 主栈的学习建议

- 先画清楚洋葱顺序再写代码：Trace → CORS → log_elapsed → Timeout → handler，任何一层位置变了，响应头就会不同。
- 写超时中间件必须缓冲响应，并让 handler 监听 `ctx.Done()`；只 `select` 不取消，会留下白等的 goroutine。
- 408 的 `x-response-time-ms` 是检验层序的最好用例：能稳定看到这个头，说明 log 层确实在 timeout 外层且先缓冲后提交。
- CORS 预检在中间件层短路，不要让它落到路由；生产环境收紧 origin，不要照搬 `*`。
- 优雅关闭要处理 `http.ErrServerClosed`，并给 `Shutdown` 设置超时，避免在途请求无限拖住退出。
