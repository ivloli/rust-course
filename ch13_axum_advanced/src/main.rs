//! 第 13 章：Axum 进阶
//! 运行：cargo run -p ch13_axum_advanced
//! 自定义端口：PORT=8080 RUST_LOG=debug cargo run -p ch13_axum_advanced

use std::time::{Duration, Instant};

use axum::{
    Json, Router,
    extract::Request,
    http::{HeaderValue, Method, StatusCode, header},
    middleware::{self, Next},
    response::{IntoResponse, Response},
    routing::get,
};
use serde_json::{Value, json};
use tower_http::{
    cors::{Any, CorsLayer},
    timeout::TimeoutLayer,
    trace::TraceLayer,
};

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "info,tower_http=debug".into()),
        )
        .init();

    let app = Router::new()
        .route("/", get(root))
        .route("/health", get(health))
        .route("/slow", get(slow))
        .route("/fail", get(fail))
        // layer 从下往上包裹：Trace 在最外层，Timeout 在最内层
        .layer(TimeoutLayer::with_status_code(
            StatusCode::REQUEST_TIMEOUT,
            Duration::from_millis(500),
        ))
        .layer(middleware::from_fn(log_elapsed))
        .layer(
            CorsLayer::new()
                .allow_origin(Any)
                .allow_methods([Method::GET, Method::POST])
                .allow_headers([header::CONTENT_TYPE]),
        )
        .layer(TraceLayer::new_for_http());

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let addr = format!("127.0.0.1:{port}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    tracing::info!("listening on http://{addr}");
    println!("试试: curl http://{addr}/slow   （500ms 超时，应返回 408）");

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .unwrap();
}

async fn root() -> &'static str {
    "Axum 进阶示例：/health /slow /fail"
}

async fn health() -> Json<Value> {
    Json(json!({ "status": "ok" }))
}

async fn slow() -> &'static str {
    tokio::time::sleep(Duration::from_secs(2)).await;
    "终于完成了"
}

async fn fail() -> Result<&'static str, AppError> {
    Err(AppError::NotFound("资源不存在".to_string()))
}

// 自定义中间件：记录耗时，并写入响应头
async fn log_elapsed(request: Request, next: Next) -> Response {
    let method = request.method().clone();
    let uri = request.uri().clone();
    let started = Instant::now();

    let mut response = next.run(request).await;

    let elapsed = started.elapsed();
    let millis = HeaderValue::from_str(&elapsed.as_millis().to_string()).unwrap();
    response.headers_mut().insert("x-response-time-ms", millis);

    tracing::info!("{method} {uri} -> {} in {:?}", response.status(), elapsed);
    response
}

// 统一错误类型：handler 返回 Err(AppError) 即可自动变成 JSON 响应
#[derive(Debug)]
enum AppError {
    NotFound(String),
    Internal(String),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match self {
            AppError::NotFound(message) => (StatusCode::NOT_FOUND, message),
            AppError::Internal(message) => (StatusCode::INTERNAL_SERVER_ERROR, message),
        };
        (status, Json(json!({ "error": message }))).into_response()
    }
}

impl From<std::io::Error> for AppError {
    fn from(error: std::io::Error) -> Self {
        AppError::Internal(error.to_string())
    }
}

async fn shutdown_signal() {
    tokio::signal::ctrl_c().await.expect("无法监听 Ctrl-C 信号");
    tracing::info!("收到 Ctrl-C，开始优雅关闭：等待进行中的请求完成");
}
