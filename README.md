# Rust 实战教程：从语言基础到 Axum + AI Agent

一套面向 **网络服务 / AI Agent 开发** 的 Rust 分章教程。每一章都是一个独立可运行的 Cargo crate，代码即讲义，章节 README 负责讲解概念、阅读顺序与练习。

## 快速开始

```bash
cd ~/rust-course
cargo check --workspace          # 首次会下载并编译依赖
cargo run -p ch01_variables      # 从第 1 章开始
```

环境要求：Rust 1.85+（使用 edition 2024）。本仓库在 Rust 1.99 / macOS (arm64) 上验证。

## 目录

| 章节 | 主题 | 运行命令 |
| --- | --- | --- |
| 01 | 变量、类型与可变性 | `cargo run -p ch01_variables` |
| 02 | 函数、控制流与模式匹配 | `cargo run -p ch02_control_flow` |
| 03 | 所有权、借用与生命周期 | `cargo run -p ch03_ownership` |
| 04 | 结构体、枚举与 Option | `cargo run -p ch04_structs_enums` |
| 05 | 集合、迭代器与闭包 | `cargo run -p ch05_collections` |
| 06 | 错误处理：Result 与自定义错误 | `cargo run -p ch06_errors` |
| 07 | Trait、泛型与多态 | `cargo run -p ch07_traits_generics` |
| 08 | 模块、测试与工程结构 | `cargo run -p ch08_modules` / `cargo test -p ch08_modules` |
| 09 | 线程与共享状态（std） | `cargo run -p ch09_threads` |
| 10 | 异步编程与 Tokio | `cargo run -p ch10_async_tokio` |
| 11 | TCP 与 HTTP 网络编程 | `cargo run -p ch11_networking` |
| 12 | Axum 基础：路由、提取器、状态 | `cargo run -p ch12_axum_basics` |
| 13 | Axum 进阶：中间件、错误、优雅关闭、性能 | `cargo run -p ch13_axum_advanced` |
| 14 | 综合实战：LLM Agent + Axum 服务 | `cargo run -p ch14_agent_server --bin agent` / `--bin server` |

## 推荐学习路线

1. **第 1 周**：01–05。重点是所有权、借用、枚举与迭代器，每天跑一遍并完成练习。
2. **第 2 周**：06–08。把错误处理与 trait 用熟，开始写自己的小 crate 与测试。
3. **第 3 周**：09–11。理解线程 vs 异步、TCP 字节流与消息边界。
4. **第 4 周**：12–14。用 Axum 写服务，最后把 Agent 工具调用循环跑通，再尝试接入真实模型。

如果时间有限，且目标是「尽快写 Web 服务 + Agent」：优先读 03、04、06、07、10、12、13、14，遇到不懂的语法再回补前几章。

## 项目结构

```
rust-course/
├── Cargo.toml              # workspace：统一依赖版本
├── ch01_variables/         # 每章一个 crate
│   ├── Cargo.toml
│   ├── README.md           # 概念讲解 + 代码导读 + 练习
│   └── src/main.rs
├── ...
└── ch14_agent_server/
    ├── src/lib.rs
    ├── src/llm.rs          # LLM 抽象：Mock / OpenAI 兼容
    ├── src/tools.rs        # 工具注册表
    ├── src/agent.rs        # 工具调用循环
    ├── src/bin/agent.rs    # CLI
    └── src/bin/server.rs   # Axum + SSE
```

## 第 14 章的环境变量

```bash
export OPENAI_API_KEY="sk-..."            # 不设置则使用离线 Mock
export OPENAI_BASE_URL="https://api.openai.com/v1"  # 可选
export OPENAI_MODEL="gpt-4o-mini"         # 可选
```

## 质量检查

```bash
cargo fmt --all --check
cargo clippy --workspace --all-targets -- -D warnings
cargo test --workspace
```

## 学习建议

- **先运行，再读代码，再改代码**：每章 README 的「动手练习」比读十遍书有效。
- **编译错误是最好的老师**：第 03、06 章故意留了被注释掉的错误示例，逐个放开看报错。
- **不要怕 clone**：初学阶段为了绕开借用检查而 clone 可以接受，但要标记出来，之后再优化。
- **用 `rust-analyzer`**：VS Code / JetBrains 安装 Rust 插件，能直接看到类型与生命周期提示。
- **对照官方文档**：`cargo doc --open` 查看依赖的 API；`std` 文档是最高频的查阅对象。

## 依赖说明

- `tokio`：异步运行时
- `axum` 0.8：Web 框架（基于 hyper 1.x / tower）
- `tower-http`：CORS、超时、追踪等中间件
- `serde` / `serde_json`：序列化
- `reqwest`：HTTP 客户端（第 11、14 章）
- `anyhow`：应用层错误汇总（第 14 章）
- `async-trait`：异步 trait 对象（第 14 章）
- `tracing` / `tracing-subscriber`：结构化日志

## 下一步方向

- 数据库：`sqlx` + PostgreSQL
- gRPC：`tonic` + `prost`
- 可观测性：OpenTelemetry + `tracing-opentelemetry`
- 性能：`criterion` 基准测试、`tokio-console`、`cargo flamegraph`
- 工程化：`thiserror`、`cargo nextest`、CI（fmt + clippy + test）
