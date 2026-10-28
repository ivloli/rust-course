use anyhow::{Result, anyhow};
use serde::Serialize;

use crate::llm::{Llm, Message};
use crate::tools::ToolRegistry;

#[derive(Debug, Clone, Serialize)]
pub struct ToolTrace {
    pub name: String,
    pub arguments: String,
    pub result: String,
}

#[derive(Debug, Serialize)]
pub struct AgentReply {
    pub answer: String,
    pub steps: usize,
    pub tools: Vec<ToolTrace>,
}

pub struct Agent {
    llm: Box<dyn Llm>,
    tools: ToolRegistry,
    max_steps: usize,
}

impl Agent {
    pub fn new(llm: Box<dyn Llm>, tools: ToolRegistry) -> Self {
        Self {
            llm,
            tools,
            max_steps: 6,
        }
    }

    pub fn llm_name(&self) -> &str {
        self.llm.name()
    }

    /// 经典的 Agent 循环：
    /// 1. 把对话历史 + 工具清单发给模型
    /// 2. 模型要求调用工具 -> 本地执行 -> 结果写回历史
    /// 3. 模型不再调用工具 -> 返回最终回答
    pub async fn run(&self, history: &mut Vec<Message>, user_input: &str) -> Result<AgentReply> {
        if history.is_empty() {
            history.push(Message::system(
                "你是一个简洁的中文助手。需要计算或查询天气时，必须调用提供的工具，不要凭空编造结果。",
            ));
        }
        history.push(Message::user(user_input));

        let specs = self.tools.specs();
        let mut tool_trace = Vec::new();

        for step in 1..=self.max_steps {
            let reply = self.llm.chat(history, &specs).await?;
            let calls = reply.tool_calls.clone().unwrap_or_default();
            history.push(reply.clone());

            if calls.is_empty() {
                return Ok(AgentReply {
                    answer: reply.content.unwrap_or_default(),
                    steps: step,
                    tools: tool_trace,
                });
            }

            for call in calls {
                tracing::info!(
                    tool = %call.function.name,
                    arguments = %call.function.arguments,
                    "执行工具"
                );
                let result = match self
                    .tools
                    .call(&call.function.name, &call.function.arguments)
                    .await
                {
                    Ok(result) => result,
                    Err(error) => format!("工具执行失败: {error}"),
                };
                tool_trace.push(ToolTrace {
                    name: call.function.name.clone(),
                    arguments: call.function.arguments.clone(),
                    result: result.clone(),
                });
                history.push(Message::tool(&call.id, result));
            }
        }

        Err(anyhow!(
            "超过最大步数 {} 仍未得到最终答案（可能是工具调用死循环）",
            self.max_steps
        ))
    }
}
