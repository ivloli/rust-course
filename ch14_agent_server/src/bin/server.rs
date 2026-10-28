use std::collections::HashMap;
use std::convert::Infallible;
use std::sync::Arc;
use std::time::Duration;

use axum::{
    Json, Router,
    extract::State,
    http::StatusCode,
    response::sse::{Event, KeepAlive, Sse},
    routing::{get, post},
};
use ch14_agent_server::{Agent, Message, ToolRegistry, ToolTrace, from_env};
use futures_util::{Stream, StreamExt, stream};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};
use tokio::sync::{Mutex, RwLock};

type Session = Arc<Mutex<Vec<Message>>>;
type ApiError = (StatusCode, Json<Value>);

#[derive(Clone)]
struct AppState {
    agent: Arc<Agent>,
    sessions: Arc<RwLock<HashMap<String, Session>>>,
}

#[derive(Deserialize)]
struct ChatRequest {
    session_id: Option<String>,
    message: String,
}

#[derive(Serialize)]
struct ChatResponse {
    session_id: String,
    answer: String,
    steps: usize,
    tools: Vec<ToolTrace>,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .init();

    let state = AppState {
        agent: Arc::new(Agent::new(from_env(), ToolRegistry::default_tools())),
        sessions: Arc::new(RwLock::new(HashMap::new())),
    };

    let app = Router::new()
        .route("/health", get(health))
        .route("/chat", post(chat))
        .route("/chat/stream", post(chat_stream))
        .with_state(state);

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let addr = format!("127.0.0.1:{port}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    println!("Agent 服务已启动: http://{addr}");
    println!(
        "非流式: curl -X POST http://{addr}/chat -H 'content-type: application/json' -d '{{\"session_id\":\"s1\",\"message\":\"3 乘以 7 等于多少？\"}}'"
    );
    println!(
        "流式:   curl -N -X POST http://{addr}/chat/stream -H 'content-type: application/json' -d '{{\"session_id\":\"s1\",\"message\":\"北京天气怎么样？\"}}'"
    );

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .unwrap();
}

async fn health(State(state): State<AppState>) -> Json<Value> {
    Json(json!({
        "status": "ok",
        "llm": state.agent.llm_name(),
        "sessions": state.sessions.read().await.len(),
    }))
}

async fn chat(
    State(state): State<AppState>,
    Json(request): Json<ChatRequest>,
) -> Result<Json<ChatResponse>, ApiError> {
    let session_id = request.session_id.unwrap_or_else(|| "default".to_string());
    let history = session(&state, &session_id).await;

    // 同一 session 的请求串行执行；不同 session 互不阻塞
    let mut history = history.lock().await;
    let reply = state
        .agent
        .run(&mut history, &request.message)
        .await
        .map_err(|error| internal_error(error.to_string()))?;

    Ok(Json(ChatResponse {
        session_id,
        answer: reply.answer,
        steps: reply.steps,
        tools: reply.tools,
    }))
}

/// SSE 演示：Agent 先完成推理，再把答案按块推送。
/// 真实产品通常把 LLM 的 token 流直接转发给客户端。
async fn chat_stream(
    State(state): State<AppState>,
    Json(request): Json<ChatRequest>,
) -> Result<Sse<impl Stream<Item = Result<Event, Infallible>>>, ApiError> {
    let session_id = request.session_id.unwrap_or_else(|| "default".to_string());
    let history = session(&state, &session_id).await;

    let answer = {
        let mut history = history.lock().await;
        state
            .agent
            .run(&mut history, &request.message)
            .await
            .map_err(|error| internal_error(error.to_string()))?
            .answer
    };

    let mut chunks: Vec<String> = answer
        .chars()
        .collect::<Vec<_>>()
        .chunks(6)
        .map(|chunk| chunk.iter().collect())
        .collect();
    chunks.push("[DONE]".to_string());

    let events = stream::iter(chunks).then(|chunk| async move {
        tokio::time::sleep(Duration::from_millis(60)).await;
        Ok::<_, Infallible>(Event::default().data(chunk))
    });

    Ok(Sse::new(events).keep_alive(KeepAlive::default()))
}

async fn session(state: &AppState, session_id: &str) -> Session {
    let mut sessions = state.sessions.write().await;
    sessions
        .entry(session_id.to_string())
        .or_insert_with(|| Arc::new(Mutex::new(Vec::new())))
        .clone()
}

fn internal_error(message: String) -> ApiError {
    (
        StatusCode::INTERNAL_SERVER_ERROR,
        Json(json!({ "error": message })),
    )
}

async fn shutdown_signal() {
    tokio::signal::ctrl_c().await.expect("无法监听 Ctrl-C");
    tracing::info!("收到 Ctrl-C，优雅关闭");
}
