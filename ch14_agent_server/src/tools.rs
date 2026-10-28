use anyhow::{Result, bail};
use async_trait::async_trait;
use serde::Deserialize;
use serde_json::json;

use crate::llm::ToolSpec;

/// 工具接口：描述自己（给模型看）+ 执行自己（给程序跑）。
#[async_trait]
pub trait Tool: Send + Sync {
    fn spec(&self) -> ToolSpec;
    async fn call(&self, arguments: &str) -> Result<String>;
}

pub struct ToolRegistry {
    tools: Vec<Box<dyn Tool>>,
}

impl ToolRegistry {
    pub fn default_tools() -> Self {
        Self {
            tools: vec![Box::new(Calculator), Box::new(Weather)],
        }
    }

    pub fn specs(&self) -> Vec<ToolSpec> {
        self.tools.iter().map(|tool| tool.spec()).collect()
    }

    pub async fn call(&self, name: &str, arguments: &str) -> Result<String> {
        let tool = self
            .tools
            .iter()
            .find(|tool| tool.spec().function.name == name)
            .ok_or_else(|| anyhow::anyhow!("未知工具: {name}"))?;
        tool.call(arguments).await
    }
}

struct Calculator;

#[derive(Deserialize)]
struct CalculatorArgs {
    a: f64,
    b: f64,
    op: String,
}

#[async_trait]
impl Tool for Calculator {
    fn spec(&self) -> ToolSpec {
        ToolSpec::function(
            "calculator",
            "对两个数字执行一次四则运算。op 取值: add / sub / mul / div",
            json!({
                "type": "object",
                "properties": {
                    "a": { "type": "number", "description": "左操作数" },
                    "b": { "type": "number", "description": "右操作数" },
                    "op": { "type": "string", "enum": ["add", "sub", "mul", "div"] }
                },
                "required": ["a", "b", "op"]
            }),
        )
    }

    async fn call(&self, arguments: &str) -> Result<String> {
        let args: CalculatorArgs = serde_json::from_str(arguments)?;
        let value = match args.op.as_str() {
            "add" => args.a + args.b,
            "sub" => args.a - args.b,
            "mul" => args.a * args.b,
            "div" => {
                if args.b == 0.0 {
                    bail!("除数不能为 0");
                }
                args.a / args.b
            }
            other => bail!("不支持的运算: {other}"),
        };

        if value.fract() == 0.0 {
            Ok(format!("{}", value as i64))
        } else {
            Ok(format!("{value}"))
        }
    }
}

struct Weather;

#[derive(Deserialize)]
struct WeatherArgs {
    city: String,
}

#[async_trait]
impl Tool for Weather {
    fn spec(&self) -> ToolSpec {
        ToolSpec::function(
            "weather",
            "查询指定城市的当前天气（示例工具，返回 mock 数据）",
            json!({
                "type": "object",
                "properties": {
                    "city": { "type": "string", "description": "城市名，例如：北京" }
                },
                "required": ["city"]
            }),
        )
    }

    async fn call(&self, arguments: &str) -> Result<String> {
        let args: WeatherArgs = serde_json::from_str(arguments)?;
        tokio::time::sleep(std::time::Duration::from_millis(30)).await;
        Ok(format!("{}：晴，24°C，东风 2 级（mock 数据）", args.city))
    }
}
