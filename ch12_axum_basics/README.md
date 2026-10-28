# 12 Axum 基础：路由、提取器、状态

```bash
cargo run -p ch12_axum_basics
# 另开终端：
curl http://127.0.0.1:3000/hello/Rust
curl http://127.0.0.1:3000/users/1
curl http://127.0.0.1:3000/users
curl -X POST http://127.0.0.1:3000/users \
  -H 'content-type: application/json' \
  -d '{"name":"Bob","email":"bob@example.com"}'
curl 'http://127.0.0.1:3000/search?q=rust&page=2'
curl -X POST http://127.0.0.1:3000/echo \
  -H 'content-type: application/json' -d '{"msg":"hi"}'
```

端口冲突时用 `PORT=8080 cargo run -p ch12_axum_basics`。

## 学习目标

- 用 `Router` 注册路径与 HTTP 方法
- 掌握 Path / Query / Json / HeaderMap / State 提取器
- 用共享状态（`Arc<RwLock<HashMap>>`）实现一个内存版用户服务
- 用 `Result<T, (StatusCode, Json<...>)>` 返回结构化错误

## 代码导读（src/main.rs）

1. 路由参数使用 axum 0.8 的新语法 `/hello/{name}`（旧版 `:name` 已移除）。
2. `get(list_users).post(create_user)` 把同一路径的不同方法挂到同一个 `MethodRouter`。
3. 提取器顺序有约束：`Json` 会消费请求体，必须放在参数列表最后。
4. `with_state(state)` 后，handler 可以用 `State<AppState>` 获取 `Clone` 的状态；状态内部用 `Arc` 共享。
5. 返回 `impl IntoResponse` 的类型都可以作为 handler 返回值：`&str`、`String`、`Json<T>`、`(StatusCode, Json<T>)` 等。
6. `get_user` 的 404 与 `create_user` 的 400 演示了「错误也是响应」：状态码 + JSON body。
7. 这里用 `std::sync::RwLock` 只在极短时间内持锁、不跨 `await`；生产环境更常见的是数据库或 `tokio::sync` 原语。

## 动手练习

1. 增加 `DELETE /users/{id}`，返回 `204 No Content` 或 404。
2. 增加 `PUT /users/{id}`，实现完整更新；思考「部分更新」该用 `PATCH` 还是 `Option` 字段。
3. 给 `CreateUser` 加校验：name 长度 1..=20，email 必须包含 `@`，返回统一错误结构。
4. 把 `AppState` 改成 `Arc<HashMap<...>>`（只读），观察为什么不再需要 `RwLock`。
5. 用 `curl -i` 查看响应头，确认 `content-type` 是 `application/json`。
6. 思考：多个 handler 同时写 `users` 时，`RwLock` 会不会成为瓶颈？如何分片或换成数据库？

## 常见坑

- `Json` 提取失败会返回 415/422，除非你自定义 rejection。
- handler 必须是 `Send`，并且不能返回借用局部变量的引用。
- 在 handler 里做阻塞操作（同步文件、CPU 重计算）会拖慢整个运行时。

## 下一步

第 13 章加入日志、CORS、超时、统一错误和优雅关闭——这是「能上线」与「demo」的分水岭。
