use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

use anyhow::{Context, Result, bail};
use async_trait::async_trait;
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

/// 对话消息。字段与 OpenAI Chat Completions 协议保持一致。
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    pub role: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_calls: Option<Vec<ToolCall>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tool_call_id: Option<String>,
}

impl Message {
    pub fn system(content: impl Into<String>) -> Self {
        Self {
            role: "system".into(),
            content: Some(content.into()),
            tool_calls: None,
            tool_call_id: None,
        }
    }

    pub fn user(content: impl Into<String>) -> Self {
        Self {
            role: "user".into(),
            content: Some(content.into()),
            tool_calls: None,
            tool_call_id: None,
        }
    }

    pub fn assistant(content: impl Into<String>) -> Self {
        Self {
            role: "assistant".into(),
            content: Some(content.into()),
            tool_calls: None,
            tool_call_id: None,
        }
    }

    pub fn assistant_with_tool_calls(content: Option<String>, tool_calls: Vec<ToolCall>) -> Self {
        Self {
            role: "assistant".into(),
            content,
            tool_calls: Some(tool_calls),
            tool_call_id: None,
        }
    }

    pub fn tool(tool_call_id: &str, content: impl Into<String>) -> Self {
        Self {
            role: "tool".into(),
            content: Some(content.into()),
            tool_calls: None,
            tool_call_id: Some(tool_call_id.into()),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FunctionCall {
    pub name: String,
    /// 模型返回的参数是 JSON 字符串，需要工具自己解析。
    pub arguments: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolCall {
    pub id: String,
    #[serde(rename = "type")]
    pub kind: String,
    pub function: FunctionCall,
}

impl ToolCall {
    pub fn new(name: &str, arguments: String) -> Self {
        static NEXT_ID: AtomicU64 = AtomicU64::new(1);
        Self {
            id: format!("call_{}", NEXT_ID.fetch_add(1, Ordering::Relaxed)),
            kind: "function".into(),
            function: FunctionCall {
                name: name.into(),
                arguments,
            },
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToolSpec {
    #[serde(rename = "type")]
    pub kind: String,
    pub function: FunctionSpec,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FunctionSpec {
    pub name: String,
    pub description: String,
    pub parameters: Value,
}

impl ToolSpec {
    pub fn function(name: &str, description: &str, parameters: Value) -> Self {
        Self {
            kind: "function".into(),
            function: FunctionSpec {
                name: name.into(),
                description: description.into(),
                parameters,
            },
        }
    }
}

/// 把「调用模型」抽象成 trait：Mock 与真实 API 可以互换。
#[async_trait]
pub trait Llm: Send + Sync {
    async fn chat(&self, messages: &[Message], tools: &[ToolSpec]) -> Result<Message>;

    fn name(&self) -> &str {
        "llm"
    }
}

/// 有 OPENAI_API_KEY 时使用真实模型，否则退回离线 Mock。
pub fn from_env() -> Box<dyn Llm> {
    match std::env::var("OPENAI_API_KEY") {
        Ok(key) if !key.trim().is_empty() => {
            let base_url = std::env::var("OPENAI_BASE_URL")
                .unwrap_or_else(|_| "https://api.openai.com/v1".to_string());
            let model = std::env::var("OPENAI_MODEL").unwrap_or_else(|_| "gpt-4o-mini".to_string());
            Box::new(OpenAiLlm::new(key, base_url, model))
        }
        _ => Box::new(MockLlm),
    }
}

pub struct OpenAiLlm {
    client: reqwest::Client,
    api_key: String,
    base_url: String,
    model: String,
}

impl OpenAiLlm {
    pub fn new(api_key: String, base_url: String, model: String) -> Self {
        let client = reqwest::Client::builder()
            .timeout(Duration::from_secs(60))
            .build()
            .expect("构建 reqwest 客户端失败");
        Self {
            client,
            api_key,
            base_url,
            model,
        }
    }
}

#[derive(Serialize)]
struct ChatRequestBody<'a> {
    model: &'a str,
    messages: &'a [Message],
    #[serde(skip_serializing_if = "Option::is_none")]
    tools: Option<&'a [ToolSpec]>,
    temperature: f32,
}

#[derive(Deserialize)]
struct ChatResponseBody {
    choices: Vec<Choice>,
}

#[derive(Deserialize)]
struct Choice {
    message: Message,
}

#[async_trait]
impl Llm for OpenAiLlm {
    async fn chat(&self, messages: &[Message], tools: &[ToolSpec]) -> Result<Message> {
        let url = format!("{}/chat/completions", self.base_url.trim_end_matches('/'));
        let body = ChatRequestBody {
            model: &self.model,
            messages,
            tools: if tools.is_empty() { None } else { Some(tools) },
            temperature: 0.2,
        };

        let response = self
            .client
            .post(&url)
            .bearer_auth(&self.api_key)
            .json(&body)
            .send()
            .await
            .context("请求 LLM 失败")?;

        let status = response.status();
        let text = response.text().await.context("读取 LLM 响应失败")?;
        if !status.is_success() {
            bail!("LLM 返回错误状态 {status}: {text}");
        }

        let parsed: ChatResponseBody =
            serde_json::from_str(&text).context("解析 LLM 响应 JSON 失败")?;
        let message = parsed
            .choices
            .into_iter()
            .next()
            .context("LLM 响应中没有 choices")?
            .message;
        Ok(message)
    }

    fn name(&self) -> &str {
        &self.model
    }
}

/// 离线 Mock：用固定脚本演示「工具调用 → 工具结果 → 最终回答」的完整循环。
pub struct MockLlm;

#[async_trait]
impl Llm for MockLlm {
    async fn chat(&self, messages: &[Message], _tools: &[ToolSpec]) -> Result<Message> {
        tokio::time::sleep(Duration::from_millis(40)).await;

        let turn = match messages.iter().rposition(|message| message.role == "user") {
            Some(index) => &messages[index..],
            None => messages,
        };
        let user_text = turn
            .first()
            .and_then(|message| message.content.clone())
            .unwrap_or_default();

        let tool_results: Vec<&str> = turn
            .iter()
            .filter(|message| message.role == "tool")
            .filter_map(|message| message.content.as_deref())
            .collect();

        let wants_calc = user_text.contains('乘')
            || user_text.contains('*')
            || user_text.contains('算')
            || user_text.contains("等于");
        let wants_weather = user_text.contains("天气");
        let has_calc = tool_results.iter().any(|result| !result.contains("°C"));
        let has_weather = tool_results.iter().any(|result| result.contains("°C"));

        if wants_calc && !has_calc {
            return Ok(Message::assistant_with_tool_calls(
                None,
                vec![ToolCall::new(
                    "calculator",
                    json!({ "a": 3, "b": 7, "op": "mul" }).to_string(),
                )],
            ));
        }

        if wants_weather && !has_weather {
            return Ok(Message::assistant_with_tool_calls(
                None,
                vec![ToolCall::new(
                    "weather",
                    json!({ "city": "北京" }).to_string(),
                )],
            ));
        }

        if !tool_results.is_empty() {
            return Ok(Message::assistant(format!(
                "我查到的结果：{}。",
                tool_results.join("；")
            )));
        }

        Ok(Message::assistant(
            "（Mock 模式）你好，我是离线演示用的模拟助手。设置 OPENAI_API_KEY 后我会调用真实模型。",
        ))
    }

    fn name(&self) -> &str {
        "mock"
    }
}
