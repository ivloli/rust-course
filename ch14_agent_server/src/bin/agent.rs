use ch14_agent_server::{Agent, ToolRegistry, from_env};

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .init();

    let agent = Agent::new(from_env(), ToolRegistry::default_tools());

    let prompt = std::env::args().skip(1).collect::<Vec<_>>().join(" ");
    let prompt = if prompt.trim().is_empty() {
        "3 乘以 7 等于多少？北京天气怎么样？".to_string()
    } else {
        prompt
    };

    println!("LLM = {}", agent.llm_name());
    println!("用户 = {prompt}\n");

    let mut history = Vec::new();
    let reply = agent.run(&mut history, &prompt).await?;

    for trace in &reply.tools {
        println!(
            "[工具] {}({}) -> {}",
            trace.name, trace.arguments, trace.result
        );
    }
    println!("\n[助手] {}", reply.answer);
    println!(
        "（共 {} 步，会话历史 {} 条消息）",
        reply.steps,
        history.len()
    );

    Ok(())
}
