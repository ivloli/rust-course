//! 第 14 章：LLM Agent + Axum 服务
//!
//! 模块划分：
//! - `llm`   : LLM 客户端抽象，支持 Mock 与 OpenAI 兼容 API
//! - `tools` : 可供模型调用的工具（calculator / weather）
//! - `agent` : 多轮工具调用循环
//!
//! 两个可执行程序：
//! - `agent`  : 命令行 Agent
//! - `server` : HTTP 服务

pub mod agent;
pub mod llm;
pub mod tools;

pub use agent::{Agent, AgentReply, ToolTrace};
pub use llm::{Llm, Message, ToolCall, ToolSpec, from_env};
pub use tools::ToolRegistry;
