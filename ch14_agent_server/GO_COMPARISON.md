# 第 14 章 Go 视角逐例对照

面向 Go 主栈读者。Rust 代码在 `src/`，Go 平行实现在 `go/`，两边的 CLI 输出、HTTP JSON 与 SSE 事件体完全一致（离线 Mock 模式下）。

```bash
# Rust
cd ~/rust-course
cargo run -p ch14_agent_server --bin agent
cargo run -p ch14_agent_server --bin server

# Go（Go 1.22+，只用标准库）
cd ~/rust-course/ch14_agent_server/go
go run ./cmd/agent
go run ./cmd/server
```

> Rust 的 tracing 默认 `info` 级别，会在 CLI 的 stdout 打印两行带 ANSI 颜色的「执行工具」日志；Go 版 CLI 不打印日志。做逐字节对比时先 `grep -v INFO` 过滤 Rust 输出，过滤后 `diff` 为空。

## 先记住三个总差异

1. **异步抽象**：Rust 用 `#[async_trait] trait Llm` + `Box<dyn Llm>`，方法返回 `Future`；Go 用普通 `interface`，方法是阻塞调用，并把 `context.Context` 作为第一个参数传递取消/超时。
2. **运行时与网络库**：Rust 是 tokio + reqwest + axum，future 需要运行时驱动；Go 的 `net/http` 每个请求一个 goroutine，`http.Client` 自带超时，`ServeMux` 直接注册 handler。
3. **流式输出**：Rust 用 `Sse<impl Stream<Item = Result<Event, Infallible>>>` 把事件流交给 axum；Go 手动设置 `text/event-stream`，每写一个事件就 `Flusher.Flush()`。

## 模块 1：Message 与 JSON 模型（llm.go）

Rust 用 `Option<T>` + `#[serde(skip_serializing_if = "Option::is_none")]` 表达可选字段：

```rust
pub struct Message {
    pub role: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
}
```

Go 用零值 + `omitempty`：

```go
type Message struct {
    Role       string     `json:"role"`
    Content    string     `json:"content,omitempty"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
}
```

Go 视角要点：

- **`omitempty` 与 `Option::None` 并不完全等价**：它同时会省略空字符串和空 slice。本项目的 assistant 消息要么有 content、要么有 tool_calls，所以序列化结果一致；如果业务上需要区分「空字符串」和「没有字段」，要用 `*string`。
- **构造函数**：Rust 的 `Message::system/user/tool` 关联函数在 Go 里是包级函数 `llm.System/User/Tool`。
- **`type` 字段**：Rust 用 `#[serde(rename = "type")]` 避免关键字冲突；Go 结构体字段叫 `Type`，tag 直接写 `"type"`。
- **工具调用 ID**：Rust 用 `static NEXT_ID: AtomicU64` 生成 `call_1`；Go 用 `atomic.Uint64` + `fmt.Sprintf("call_%d", id)`，两边都只在进程内唯一。

## 模块 2：Llm 接口与 FromEnv

Rust：

```rust
#[async_trait]
pub trait Llm: Send + Sync {
    async fn chat(&self, messages: &[Message], tools: &[ToolSpec]) -> Result<Message>;
    fn name(&self) -> &str { "llm" }
}
pub fn from_env() -> Box<dyn Llm> { /* 有 Key 用 OpenAiLlm，否则 MockLlm */ }
```

Go：

```go
type Llm interface {
    Chat(ctx context.Context, messages []Message, tools []ToolSpec) (Message, error)
    Name() string
}

func FromEnv() Llm {
    if strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
        return MockLLM{}
    }
    // OPENAI_BASE_URL / OPENAI_MODEL 有默认值
    return NewOpenAILLM(key, baseURL, model)
}
```

Go 视角要点：
- **`Send + Sync` 不用写**：Go 的接口值本身可以被多个 goroutine 共享，前提是实现方无共享可变状态。`MockLLM` 是空结构体，`OpenAILLM` 只读配置 + 并发安全的 `http.Client`。
- **`&[Message]` ↔ `[]Message`**：Go 的 slice 是「指针 + 长度 + 容量」，传参不会复制底层数组；只有 append 导致扩容时才需要把新 slice 写回（见 Agent 一节）。
- **返回 `Box<dyn Llm>` ↔ 返回接口值**：`FromEnv()` 直接返回 `Llm` 接口，调用方不关心具体类型，这与 Rust 的 trait object 分发是同一思路。
- **`Result<Message>` ↔ `(Message, error)`**：Go 没有 `?`，每个错误都要显式 `if err != nil` 返回。

## 模块 3：Mock 与 OpenAI 客户端

Mock 的核心不是「假装智能」，而是脚本化地复现工具循环。两边的判断顺序必须一致：先看本轮是否有工具结果，再决定下一步调用哪个工具。

```go
if wantsCalc && !hasCalc {
    return AssistantWithToolCalls("", []ToolCall{NewToolCall("calculator", `{"a":3,"b":7,"op":"mul"}`)}), nil
}
if wantsWeather && !hasWeather {
    return AssistantWithToolCalls("", []ToolCall{NewToolCall("weather", `{"city":"北京"}`)}), nil
}
if len(toolResults) > 0 {
    return Assistant("我查到的结果：" + strings.Join(toolResults, "；") + "。"), nil
}
```

真实客户端用 `http.Client{Timeout: 60 * time.Second}` + `NewRequestWithContext` + `Authorization: Bearer` 头替代 reqwest。

Go 视角要点：

- **URL 拼接**：Rust 的 `trim_end_matches('/')` 对应 Go 的 `strings.TrimRight(baseURL, "/")`。
- **超时**：`http.Client.Timeout` 覆盖连接、重定向和读完 body 的全过程，和 reqwest 的 `.timeout(60s)` 语义接近；如果要分别控制连接与读超时，用自定义 `Transport`。
- **空工具列表**：Rust 用 `Option<&[ToolSpec]>` + `skip_serializing_if`；Go 用 nil slice + `omitempty`。
- **取消**：`http.NewRequestWithContext` 让客户端断开或上游超时能立即终止请求；Rust 则靠 drop future 取消。
- **Mock 的 sleep**：Go 版把 `time.Sleep` 换成 `select { <-ctx.Done(); <-timer.C }`，这样请求被取消时不会白等 40ms。

## 模块 4：工具与注册表（tools.go）

Rust 的 `trait Tool: Send + Sync` 有 `spec()` 与 `async fn call()` 两个方法：

```rust
#[async_trait]
pub trait Tool: Send + Sync {
    fn spec(&self) -> ToolSpec;
    async fn call(&self, arguments: &str) -> Result<String>;
}
```

Go 的 `interface` 几乎一一对应：

```go
type Tool interface {
    Spec() llm.ToolSpec
    Call(ctx context.Context, arguments string) (string, error)
}
```

`Registry.Call` 都是「按名字线性查找 + 分发」；找不到时返回 `未知工具: <name>`，Agent 会把错误文本回填给模型。

Go 视角要点：

- **参数解析**：Rust 用 serde 反序列化到 `CalculatorArgs { a: f64, b: f64, op: String }`，缺字段直接报错。Go 的 `float64` 缺字段会静默变 0，所以示例用 `*float64` 检测缺失。
- **整数格式化**：Rust `value.fract() == 0.0` 时 `format!("{}", value as i64)`；Go 用 `value == math.Trunc(value)` 判断，再 `strconv.FormatInt(int64(value), 10)`，得到的 `21` 与 Rust 完全一致。
- **除零**：两边都在 `div` 分支显式返回错误 `除数不能为 0`，而不是让浮点产生 `+Inf`。
- **天气工具**：30ms 延迟 + 固定字符串 `城市：晴，24°C，东风 2 级（mock 数据）`。`°C` 同时是 Mock 判断「天气是否已查过」的标记，改动文案会破坏脚本逻辑。
- **JSON Schema**：Rust 用 `json!` 宏，Go 用 `map[string]any`；两者都只是给模型看的描述，本地不做校验。

## 模块 5：Agent 循环（agent.go）

Rust 的 `&mut Vec<Message>` 在 Go 里必须写成 `*[]Message`：

```rust
pub async fn run(&self, history: &mut Vec<Message>, user_input: &str) -> Result<AgentReply>
```

```go
func (a *Agent) Run(ctx context.Context, history *[]llm.Message, userInput string) (Reply, error) {
    if len(*history) == 0 {
        *history = append(*history, llm.System(systemPrompt))
    }
    *history = append(*history, llm.User(userInput))
    // 最多 MaxSteps=6 轮
}
```

Go 视角要点：

- **append 要写回**：`append` 可能分配新底层数组，只写 `append(*history, msg)` 而不赋值会丢消息。这是本章最容易踩的坑。
- **Reply 的 JSON**：`ToolTrace{name, arguments, result}`、`Reply{answer, steps, tools}` 的 tag 与 Rust `Serialize` 字段名逐一对齐；`trace` 必须初始化成空切片，否则空结果会序列化成 `null` 而不是 `[]`。
- **步数语义**：`step` 从 1 开始，模型返回最终回答的那一轮计入步数。默认问题需要 calculator、weather 两次工具调用，所以 `steps = 3`、历史 7 条消息。
- **错误回填**：工具执行失败不向上抛，而是追加一条 `role=tool` 的消息，让模型下一轮自我修正；只有 LLM 调用失败或超过 6 步才返回 error。
- **system prompt 只加一次**：靠 `len(*history) == 0` 判断，和 Rust 完全相同；同一 session 的多轮请求共享这段历史。

## 模块 6：CLI（cmd/agent/main.go）

```go
prompt := strings.Join(os.Args[1:], " ")
if strings.TrimSpace(prompt) == "" { prompt = "3 乘以 7 等于多少？北京天气怎么样？" }
```

输出格式逐字符对齐 Rust：`[工具] name(args) -> result`、空行后的 `[助手] answer`、末尾 `（共 N 步，会话历史 M 条消息）`。

Go 视角要点：

- **命令行参数**：Rust `env::args().skip(1).collect::<Vec<_>>().join(" ")`；Go 是 `os.Args[1:]` + `strings.Join`。
- **错误退出**：Rust 的 `main -> anyhow::Result` 失败时打印 `Error: ...` 并退出 1；Go 显式 `fmt.Fprintf(os.Stderr, "Error: %v\n", err)` + `os.Exit(1)`。
- **验证方式**：`grep -v INFO /tmp/rust14.txt > a && diff -u a /tmp/go14.txt`，过滤 tracing 后无差异。

## 模块 7：HTTP 服务（cmd/server/main.go）

Rust 用 `AppState { agent: Arc<Agent>, sessions: Arc<RwLock<HashMap<String, Session>>> }`，Go 用「全局 `sync.Mutex` + `map[string]*session` + 每个 session 自己的 `sync.Mutex`」；查/建会话只短暂锁全局 map，执行 Agent 时只锁当前会话的 history。

端点对照：

| 端点 | Rust | Go |
| --- | --- | --- |
| `GET /health` | `Json(json!({status, llm, sessions}))` | `map[string]any` + `writeJSON` |
| `POST /chat` | `Json<ChatRequest>` 提取器 | 手动 `json.Unmarshal` |
| `POST /chat/stream` | `Sse<impl Stream<...>>` | 设头 + `Flush` |

Go 视角要点：

- **会话锁粒度**：查/建会话只锁全局 map，执行 Agent 时只锁当前 session，因此不同会话可并行；这与 Rust 的 `RwLock` + `tokio::Mutex` 设计一致。
- **默认会话**：`session_id` 缺省为 `"default"`。Go 用 `*string` 区分「没传」和「传了空串」，与 Rust `Option<String>` 一致。
- **错误响应**：Agent 失败时两边都是 HTTP 500 + `{"error": "..."}`。
- **请求解析差异**：axum 对 JSON 语法错误返回 400、字段缺失返回 422；Go 示例统一返回 400 JSON。正常请求不受影响。
- **优雅关闭**：Rust 用 `with_graceful_shutdown` 监听 Ctrl-C；Go 用 `signal.NotifyContext(SIGINT, SIGTERM)` + `server.Shutdown(5s)`。
- **端口**：两边都读 `PORT`，默认 3000，绑定 `127.0.0.1`。

## 模块 8：SSE 流式输出

Rust 用 `answer.chars().collect::<Vec<_>>().chunks(6)` 切块，最后 push `"[DONE]"`；Go 的写法：

```go
runes := []rune(reply.Answer)
for i := 0; i < len(runes); i += 6 {
    end := min(i+6, len(runes))
    fmt.Fprintf(w, "data: %s\n\n", string(runes[i:end]))
    flusher.Flush()
}
```

Go 视角要点：
- **按 rune 分块**：Rust 的 `chars()` 对应 Go 的 `[]rune`；按字节切会把中文和 `°C` 切坏。
- **帧格式**：axum 的 `Event::default().data(x)` 输出就是 `data: x\n\n`，Go 手写同样格式，因此两边 SSE body 逐字节相同。
- **先推理后推送**：两边都是 Agent 跑完再切块，不是真正的 token 流；真实产品应透传上游 `stream: true`。
- **响应头**：Go 额外发送 `Connection: keep-alive`，Rust 不发送；`content-type` 与 `cache-control` 相同。
- **客户端断开**：Go 用 `r.Context().Done()` 提前结束循环；Rust 在 stream 被 drop 时自动停止。

## 速查表

| Rust | Go 1.22+ |
| --- | --- |
| `#[async_trait] trait Llm` | `type Llm interface` |
| `Box<dyn Llm>` | `Llm` 接口值 |
| `async fn chat(&self, ...)` | `Chat(ctx, ...)` 阻塞方法 |
| `Result<T>` / `anyhow::bail!` | `(T, error)` / `fmt.Errorf` |
| `Option<String>` + `skip_serializing_if` | `string` + `omitempty`（或 `*string`） |
| `#[serde(rename = "type")]` | `` `json:"type"` `` |
| `Vec<Message>` + `&mut` | `[]llm.Message` + `*[]llm.Message` |
| `AtomicU64` | `atomic.Uint64` |
| `tokio::time::sleep` | `time.NewTimer` + `select` |
| `reqwest::Client` + `.timeout(60s)` | `http.Client{Timeout: 60s}` |
| `.bearer_auth(key)` | `Header.Set("Authorization", "Bearer "+key)` |
| axum `Router::route` | `http.NewServeMux` + `HandleFunc` |
| axum `Json<T>` 提取器 | `json.Unmarshal` |
| `Arc<RwLock<HashMap<..>>>` | `sync.Mutex` + `map` |
| `tokio::sync::Mutex` | `sync.Mutex` |
| `Sse<impl Stream<...>>` | `Content-Type: text/event-stream` + `Flush` |
| `with_graceful_shutdown` | `server.Shutdown(ctx)` |
| `tracing::info!` | 标准库 `log` / `slog`（示例未打日志） |

## 生产化差异备注

- **工具调用 ID**：两边的 Mock 都用进程内自增 ID（`call_1`、`call_2`……）。如果会话历史持久化到 Redis 并在多实例间共享，ID 会冲突；接真实模型时应直接使用服务商返回的 `id`，它才是回填 `tool_call_id` 的依据。
- **JSON Schema**：示例只把 schema 当描述发给模型，本地不校验。生产环境应在 `Registry.Call` 前用 schema 校验参数类型/枚举，避免模型传入意外字段。
- **超时**：Rust 的 reqwest 超时与 Go 的 `http.Client.Timeout` 都覆盖整个请求；但 Agent 一次 `Run` 可能串行调用多次 LLM，整体耗时是倍数关系，服务端还应加请求级超时（Go 用 `context.WithTimeout`，Rust 用 `tokio::time::timeout`）。
- **取消传播**：Go 的 `ctx` 会一路传到 `http.NewRequestWithContext` 和工具的 timer；Rust 依赖 future drop。注意 `/chat` 在客户端断开后 Go 会取消，Rust 的 axum handler 也会因 future 被 drop 而停止。
- **会话存储**：两边都是内存 map，进程重启即丢失，也没有过期清理；生产环境应换成 Redis/数据库并加 TTL。
- **SSE 细节**：示例每 60ms 发一帧，没有心跳；长连接场景应发送注释行心跳（axum 的 `KeepAlive`，Go 可定时写 `: ping\n\n`），并注意反向代理的缓冲。
