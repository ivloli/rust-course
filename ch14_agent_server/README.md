# 14 综合实战：LLM Agent + Axum 服务

这是全课程的收尾：把前 13 章用到的 trait、错误处理、Tokio、Axum、状态管理组合成一个可运行的 AI Agent 服务。

## 架构

```
用户
  │  POST /chat {"session_id":"s1","message":"..."}
  ▼
Axum Router ──► 每会话历史 Arc<Mutex<Vec<Message>>>
  │
  ▼
Agent::run ──循环──► Llm::chat(历史, 工具清单)
  │                     │
  │                     ├─ 返回 tool_calls ──► ToolRegistry 执行工具
  │                     │                      │
  │                     └──────── 结果写回历史 ◄┘
  │
  └─ 无 tool_calls ──► 最终回答
```

- `src/llm.rs`：`Llm` trait + `MockLlm` + `OpenAiLlm`（任何 OpenAI 兼容 API）
- `src/tools.rs`：`Tool` trait + 计算器 / 天气工具 + `ToolRegistry`
- `src/agent.rs`：最多 6 步的工具调用循环
- `src/bin/agent.rs`：命令行入口
- `src/bin/server.rs`：Axum 服务，支持会话与 SSE

## 一、离线运行（默认 Mock，无需 API Key）

```bash
cargo run -p ch14_agent_server --bin agent
# 或自定义问题：
cargo run -p ch14_agent_server --bin agent -- "3*7 等于多少？北京天气如何？"
```

> Go 主栈读者：每个模块与接口的 Go 平行写法见 [GO_COMPARISON.md](GO_COMPARISON.md)，可运行代码在 `go/`（`go run ./cmd/agent` / `go run ./cmd/server`）。

预期输出会依次出现两次工具调用（calculator、weather），最后给出汇总答案。

## 二、接入真实模型（OpenAI 兼容）

```bash
export OPENAI_API_KEY="sk-..."
# 可选：切换兼容服务商
export OPENAI_BASE_URL="https://api.openai.com/v1"
export OPENAI_MODEL="gpt-4o-mini"

cargo run -p ch14_agent_server --bin agent -- "帮我算一下 3*7，再看看北京天气"
```

> 任何兼容 Chat Completions + function calling 的服务都可以（自建网关、DeepSeek、通义、Moonshot 等），只需要改 `OPENAI_BASE_URL` 与 `OPENAI_MODEL`。

## 三、启动 HTTP 服务

```bash
cargo run -p ch14_agent_server --bin server
# 另开终端：
curl http://127.0.0.1:3000/health

curl -X POST http://127.0.0.1:3000/chat \
  -H 'content-type: application/json' \
  -d '{"session_id":"s1","message":"3 乘以 7 等于多少？"}'

# 同一个 session 的第二轮，历史会被保留
curl -X POST http://127.0.0.1:3000/chat \
  -H 'content-type: application/json' \
  -d '{"session_id":"s1","message":"北京天气怎么样？"}'

# SSE 流式输出（-N 关闭 curl 缓冲）
curl -N -X POST http://127.0.0.1:3000/chat/stream \
  -H 'content-type: application/json' \
  -d '{"session_id":"s2","message":"北京天气怎么样？"}'
```

端口冲突时：`PORT=8080 cargo run -p ch14_agent_server --bin server`。

## 代码导读

1. `llm.rs` 的 `Message` 同时实现 `Serialize` / `Deserialize`，请求与响应共用一个结构；`skip_serializing_if` 让可选字段不出现在 JSON 里。
2. `Llm` 是 async trait：用 `#[async_trait]` 才能放进 `Box<dyn Llm>`（原生 AFIT 目前不能 dyn）。
3. `from_env()` 是「配置决定实现」的典型写法：有 Key 走真实 API，没有就走 Mock，接口完全一致。
4. `tools.rs` 中每个工具都提供 JSON Schema 描述；模型靠描述决定是否调用、参数怎么填。
5. `ToolRegistry::call` 按名字查找工具；未知工具返回错误，Agent 会把错误文本回填给模型，让它自我修正。
6. `agent.rs` 的核心是 `loop` + `tool_calls` 判断；`max_steps` 是防止模型陷入工具死循环的保险丝。
7. `server.rs` 用 `Arc<RwLock<HashMap<String, Arc<Mutex<Vec<Message>>>>>>`：会话表本身可并发读写，单个会话内串行。
8. SSE handler 先把 Agent 跑完，再分块推送；真实产品可在 `OpenAiLlm` 中打开 `stream: true`，边收 token 边转发。

## 生产化清单

- **安全**：工具是代码执行入口，必须做参数校验与权限控制；示例计算器只接受数字运算，不要直接 `eval`。
- **Prompt 注入**：工具返回的内容可能包含恶意指令，回填给模型前应标注来源，并限制工具权限。
- **超时与重试**：`reqwest` 客户端已设 60s 超时；生产环境应对 429/5xx 做指数退避。
- **成本**：限制 `max_steps`、历史长度与输出 token；记录每会话 token 消耗。
- **并发**：单会话串行是正确性要求；跨会话可用连接池 + 信号量限制并发下游请求。
- **可观测性**：为每次请求建立 tracing span，记录模型、工具、耗时、错误。
- **持久化**：会话历史应落 Redis/数据库，进程重启不丢失。
- **流式**：真正的首 token 延迟优化需要透传上游 SSE，而不是等 Agent 结束后再切块。

## 动手练习

1. 新增 `time` 工具：返回当前时间（`std::time::SystemTime`），并让 Mock 支持它。
2. 新增 `http_get` 工具（用 `reqwest`），并加上域名白名单——思考为什么白名单是必须的。
3. 给 `Agent` 增加「工具调用并行执行」：用 `futures_util::future::join_all` 并行跑同一步的多个工具。
4. 把会话历史限制为最近 20 条，避免上下文无限增长。
5. 给 `/chat` 增加请求级超时（参考第 13 章），并验证超时后客户端收到 504/408。
6. 把 SSE 改成真正的 token 流：在 `OpenAiLlm` 中请求 `stream: true`，解析 `data:` 行并转发。
7. 为 Agent 循环写测试：使用 MockLlm 断言工具被调用、最终回答包含计算结果。

## 注意

- 不要把 API Key 写进代码或提交到 git；本课程只从环境变量读取。
- Mock 输出是脚本化的，不代表真实模型行为；接入真实模型后再观察多步工具调用的差异。
