# 13 Axum 进阶：中间件、错误、优雅关闭与性能

```bash
RUST_LOG=debug cargo run -p ch13_axum_advanced
# 另开终端：
curl http://127.0.0.1:3000/health
curl -i http://127.0.0.1:3000/slow      # 约 500ms 后返回 408
curl -i http://127.0.0.1:3000/fail      # 404 + JSON 错误
curl -i http://127.0.0.1:3000/ -H 'Origin: http://example.com'
```

在运行终端按 `Ctrl-C`，观察 graceful shutdown 日志。

## 学习目标

- 用 tower / tower-http 中间件处理追踪、CORS、超时
- 用 `middleware::from_fn` 编写自定义中间件
- 用 `IntoResponse for AppError` 统一错误响应
- 实现优雅关闭，理解 layer 的包裹顺序
- 掌握上线前的性能与可靠性检查清单

## 代码导读（src/main.rs）

1. `tracing_subscriber` + `RUST_LOG`：结构化日志是排查线上问题的第一手材料。
2. `TraceLayer` 记录请求/响应；`CorsLayer` 允许浏览器跨域；`TimeoutLayer` 防止慢请求占满连接。
3. `.layer()` 是「洋葱模型」：后加的层在更外层。`Trace` 包住 `CORS`，`CORS` 包住自定义日志，`Timeout` 最贴近 handler。
4. `log_elapsed` 读取 `x-response-time-ms` 写入响应，浏览器 / curl `-i` 可直接看到。
5. `AppError` 实现 `IntoResponse` 后，handler 只需要返回 `Result<T, AppError>`，状态码与 JSON 结构统一。
6. `with_graceful_shutdown` 在收到 Ctrl-C 后停止接受新连接，并等待已接受的请求完成。

## 性能与上线清单（Rust Web 服务）

- 用 `--release` 构建；开发用 debug，压测必须 release。
- 不要在 async handler 里做阻塞 I/O 或重计算：用 `spawn_blocking` 或独立线程池。
- 复用 `reqwest::Client` / 数据库连接池，不要在请求里新建客户端。
- 为下游调用设置超时、重试与熔断；超时应该在「最接近下游」的层设置。
- 限制请求体大小（`DefaultBodyLimit`）与并发数，防止内存被打爆。
- 用 `tower-http` 的 `CompressionLayer` 压缩大响应，用 `TimeoutLayer` 兜底慢请求。
- 日志采样 + `tracing` span；高 QPS 下避免每个请求打印大 body。
- 压测工具：`oha`、`wrk`、`vegeta`；观察 P99、错误率、内存与 CPU。
- 优雅关闭配合负载均衡摘流；Kubernetes 下注意 readiness/liveness 探针。
- 关注 `tokio-console`、`cargo flamegraph` 定位异步任务堆积与热点。

## 动手练习

1. 把超时改为 1.5s，验证 `/slow` 行为；再把 `/slow` 里 sleep 改成 1s，观察成功返回。
2. 给 `AppError` 增加 `BadRequest` 变体，并让一个 `POST /echo` handler 在字段缺失时返回 400。
3. 用 `tower_http::limit::RequestBodyLimitLayer` 限制请求体为 1MB（需要开启对应 feature）。
4. 把 CORS 从 `Any` 改为只允许 `http://localhost:5173`，思考生产环境为什么不能全放开。
5. 用 `oha -z 10s -c 500 http://127.0.0.1:3000/health` 压测（先安装 `oha`），观察 QPS 与延迟分布。
6. 思考：如果某个 handler 调用的下游没有超时，会发生什么？服务端超时能否替代下游超时？

## 与第 14 章的衔接

第 14 章会把「调用 LLM」当成下游服务：同样需要超时、错误映射、并发控制与会话状态；那是一个真实的 AI Agent 服务骨架。
