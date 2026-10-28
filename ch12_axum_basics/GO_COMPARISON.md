# 第 12 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/main.rs`，Go 平行实现在 `go/main.go`。**状态码与 JSON 结构一致**，只有错误文案和未知路由的默认 body 略有差别。

```bash
# Rust
cd ~/rust-course && cargo run -p ch12_axum_basics

# Go（需要 Go 1.22+）
cd ~/rust-course/ch12_axum_basics/go && go run .
```

两边都读 `PORT`，默认 3000，绑定 `127.0.0.1`。

## 先记住三个总差异
1. **路由**：Axum 用 `Router::route` 挂到 `MethodRouter`；Go 1.22 直接在 pattern 里写方法和通配符。
2. **提取器**：`Path` / `Query` / `Json` / `HeaderMap` / `State` 类型驱动，失败自动 400/415/422；Go 用 `PathValue`、`URL.Query()`、`Decoder`、`Header.Get`，失败处理全手写。
3. **状态与响应**：`State<AppState>` + `Arc<RwLock<_>>` 对应结构体指针 + `sync.RWMutex`；`IntoResponse` 对应“状态码 + 手写编码器”。

## 例 1：GET / 与 /health —— 路由表

Rust：
```rust
Router::new()
    .route("/", get(root))
    .route("/health", get(health))
    .route("/users", get(list_users).post(create_user))
```
Go：
```go
mux.HandleFunc("GET /{$}", state.root) // {$} = 精确根路径
mux.HandleFunc("GET /health", state.health)
mux.HandleFunc("GET /users", state.listUsers)
mux.HandleFunc("POST /users", state.createUser)
```
- **`{$}` 不能省**：`"GET /"` 是子树匹配，会吞掉所有未知 GET；`{$}` 才是精确根。
- Axum 的 `get(a).post(b)` 是一个 `MethodRouter`；Go 写两行 pattern。`&str` 响应自动是 `text/plain`，`Json<T>` 对应 `writeJSON(w, status, v)`。

## 例 2：GET /hello/{name} —— Path

Rust：
```rust
async fn hello(Path(name): Path<String>) -> String {
    format!("Hello, {name}!")
}
```
Go：
```go
name := r.PathValue("name")
writeText(w, http.StatusOK, "Hello, "+name+"!")
```
- `Path<String>` 在调用 handler 前解析；`PathValue` 在 handler 内主动取值，没有“提取失败”这一层。
- 两边语法都是 `{name}`（Axum 0.8 已移除 `:name`），都会 URL 解码；Go 只匹配单个路径段。

## 例 3：GET /users 与 /users/{id} —— 状态、类型错误、404

Rust：
```rust
async fn get_user(Path(id): Path<u64>, State(state): State<AppState>)
    -> Result<Json<User>, (StatusCode, Json<Value>)> {
    match state.users.read().unwrap().get(&id) {
        Some(u) => Ok(Json(u.clone())),
        None => Err((StatusCode::NOT_FOUND, Json(json!({ "error": format!("用户 {id} 不存在") })))),
    }
}
```
Go：
```go
id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
if err != nil { writeText(w, 400, "Invalid URL: Cannot parse path as u64"); return }
s.mu.RLock()
user, ok := s.users[id]
s.mu.RUnlock()
if !ok {
    writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("用户 %d 不存在", id)})
    return
}
writeJSON(w, http.StatusOK, user)
```
- **类型化路径参数**：`Path<u64>` 提取阶段就拒绝非数字并返 400 纯文本；`PathValue` 永远是字符串，必须 `ParseUint`，否则 `"abc"` 会被当合法 id 查表。
- 错误也是响应：Rust 用 `Result<T, (StatusCode, Json<_>)>`；Go 直接 `writeJSON` 后 `return`。`RwLock` 只包住 map 访问，排序在锁外（两边 map 遍历都无序）。

## 例 4：POST /users —— Json 提取与写锁

Rust：
```rust
async fn create_user(State(state): State<AppState>, Json(p): Json<CreateUser>)
    -> Result<(StatusCode, Json<User>), (StatusCode, Json<Value>)> {
    if !p.email.contains('@') {
        return Err((StatusCode::BAD_REQUEST, Json(json!({ "error": "邮箱格式不正确" }))));
    }
    let mut users = state.users.write().unwrap();
    let id = users.keys().max().copied().unwrap_or(0) + 1;
    users.insert(id, User { id, name: p.name, email: p.email });
    Ok((StatusCode::CREATED, Json(users[&id].clone())))
}
```
Go：
```go
type CreateUser struct { // 指针区分“缺字段”和“空字符串”
    Name  *string `json:"name"`
    Email *string `json:"email"`
}
if err := decodeJSONBody(r, &payload); err != nil { /* 415 / 400 */ }
if payload.Name == nil || payload.Email == nil { /* 422 */ }
s.mu.Lock()
var maxID uint64
for id := range s.users { if id > maxID { maxID = id } }
id := maxID + 1
s.users[id] = User{ID: id, Name: *payload.Name, Email: *payload.Email}
s.mu.Unlock()
writeJSON(w, http.StatusCreated, user)
```
- **提取器顺序**：Axum 规定 `Json` 消费请求体，必须放参数列表最后；Go 没有语法限制，但逻辑顺序仍是“查 Content-Type → 解码 → 校验 → 加锁”。
- **Json rejection vs 手动解码**：Axum 自动 415（Content-Type 错）、400（语法错）、422（缺字段/类型错）；`json.Decoder` 只看 body。示例用 `decodeJSONBody` 补 415、用指针字段补缺字段 422；类型不匹配仍是 400，错误文案也不可能和 serde 完全一致。
- **没有 serde**：字段名靠 tag；未知字段默认忽略（和 serde 一致）；Go 默认允许 body 后有多余 JSON，serde 会报错。**id 生成必须持写锁**，`max+1` 和插入是一个临界区。

## 例 5：GET /search —— Query 与默认值

Rust：
```rust
#[derive(Deserialize)]
struct SearchParams { q: Option<String>, page: Option<u32> }

async fn search(Query(p): Query<SearchParams>) -> Json<Value> {
    Json(json!({ "query": p.q.unwrap_or_default(), "page": p.page.unwrap_or(1),
                 "note": "真实项目这里会查询数据库" }))
}
```
Go：
```go
query := r.URL.Query()
q := query.Get("q") // 缺失时是 ""
page := uint64(1)
if raw := query.Get("page"); raw != "" {
    parsed, err := strconv.ParseUint(raw, 10, 32)
    if err != nil { /* 400 */ }
    page = parsed
}
writeJSON(w, http.StatusOK, map[string]any{
    "query": q, "page": page, "note": "真实项目这里会查询数据库"})
```
- `unwrap_or_default()` 对应 `Get`（缺失返空串）；`unwrap_or(1)` 对应先赋默认值再判断。`Query<T>` 类型错误是提取阶段 400，Go 要自己 `ParseUint`。
- `map[string]any` 由 `encoding/json` 按 key 排序，和 serde_json 默认 `BTreeMap` 顺序一致。`?page=` 空值在 Rust 会失败，Go 这里按缺省处理，是细微差异。

## 例 6：POST /echo —— Json<Value>

Rust：
```rust
async fn echo(Json(body): Json<Value>) -> Json<Value> {
    Json(json!({ "received": body }))
}
```
Go：
```go
var body any
if err := decodeJSONBody(r, &body); err != nil { /* 415 / 400 */ }
writeJSON(w, http.StatusOK, map[string]any{"received": body})
```
- `serde_json::Value` 对应 `any`；`Decoder.UseNumber()` 防止整数变成 `float64`。
- `Json<Value>` 也要求 JSON Content-Type，`json.Decoder` 不检查，所以统一走 `decodeJSONBody`；请求体两边都只能读一次。

## 例 7：GET /whoami —— Header

Rust：
```rust
let ua = headers.get(header::USER_AGENT)
    .and_then(|v| v.to_str().ok()).unwrap_or("unknown");
format!("你的 User-Agent: {ua}")
```
Go：
```go
userAgent := r.Header.Get("User-Agent")
if userAgent == "" { userAgent = "unknown" }
writeText(w, http.StatusOK, "你的 User-Agent: "+userAgent)
```
- `HeaderMap` 可按需取多个头；Go 用 `Header.Get`，名字大小写不敏感。
- Rust 的 `to_str()` 会拒绝非 ASCII 头，Go 的 header 值本来就是字符串；不传 UA 时两边都输出 `unknown`。

## 例 8：PORT、启动与未知路由

Rust：
```rust
let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
let listener = tokio::net::TcpListener::bind(format!("127.0.0.1:{port}")).await?;
println!("服务已启动: http://127.0.0.1:{port}");
axum::serve(listener, app).await?;
```
Go：
```go
port := os.Getenv("PORT")
if port == "" { port = "3000" }
listener, err := net.Listen("tcp", "127.0.0.1:"+port) // 先 bind 再打印
if err != nil { log.Fatalf("监听失败: %v", err) }
fmt.Printf("服务已启动: http://127.0.0.1:%s\n", port)
server.Serve(listener)
```
- 都先 bind 再打印，端口占用时不会出现“已启动”的假日志。
- 未知路由：Axum 404 且 **body 为空**；Go 返回 404 `404 page not found`，状态码一致。方法不匹配两边都是 405，只是 Go body 是 `Method Not Allowed`。
## 速查表

| Rust / Axum | Go 1.22+ |
| --- | --- |
| `Router::new().route(...)` | `http.NewServeMux()` + `HandleFunc` |
| `.route("/x/{id}", get(h))` | `mux.HandleFunc("GET /x/{id}", h)` |
| 根路由 `/` | 精确根用 `/{$}` |
| `Path<T>` | `r.PathValue("id")` + 手动解析 |
| `Query<T>` | `r.URL.Query().Get(...)` + 手动解析 |
| `Json<T>` | `json.NewDecoder(r.Body).Decode(&v)` |
| `State<AppState>` | 结构体指针 + 方法 handler |
| `Arc<RwLock<T>>` | `sync.RWMutex` |
| `(StatusCode, Json<T>)` | `writeJSON(w, status, v)` |
| `Result<T, E>` + `?` | `if err != nil { ...; return }` |
| Json 自动 415/400/422 | 手动 Content-Type + 解码 + 指针判空 |
| 未知路由：404 空 body | 404 `404 page not found` |
## 给 Go 主栈的学习建议

- 把 Axum 提取器想成“框架替你写的参数解析 + 错误响应”；读 handler 先看参数列表，就知道它读什么。
- 写 Go 版本先列失败分支：415、400、422、业务 400、404。Axum 帮你分类，Go 要自己分。
- 共享状态用 `RWMutex` 保护最小临界区；不要持锁做 JSON 编码、网络调用或日志 I/O。和 serde 对齐注意：缺字段用指针、数字用 `UseNumber`、map 输出自动排序。
