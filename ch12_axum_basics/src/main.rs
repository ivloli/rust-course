//! 第 12 章：Axum 基础
//! 运行：cargo run -p ch12_axum_basics
//! 自定义端口：PORT=8080 cargo run -p ch12_axum_basics

use std::collections::HashMap;
use std::sync::{Arc, RwLock};

use axum::{
    Json, Router,
    extract::{Path, Query, State},
    http::{HeaderMap, StatusCode, header},
    routing::{get, post},
};
use serde::{Deserialize, Serialize};
use serde_json::{Value, json};

#[derive(Clone)]
struct AppState {
    users: Arc<RwLock<HashMap<u64, User>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct User {
    id: u64,
    name: String,
    email: String,
}

#[derive(Deserialize)]
struct CreateUser {
    name: String,
    email: String,
}

#[derive(Deserialize)]
struct SearchParams {
    q: Option<String>,
    page: Option<u32>,
}

#[tokio::main]
async fn main() {
    let mut users = HashMap::new();
    users.insert(
        1,
        User {
            id: 1,
            name: "Alice".into(),
            email: "alice@example.com".into(),
        },
    );
    let state = AppState {
        users: Arc::new(RwLock::new(users)),
    };

    // 路由表：路径 + HTTP 方法 -> handler
    let app = Router::new()
        .route("/", get(root))
        .route("/health", get(health))
        .route("/hello/{name}", get(hello))
        .route("/users", get(list_users).post(create_user))
        .route("/users/{id}", get(get_user))
        .route("/search", get(search))
        .route("/echo", post(echo))
        .route("/whoami", get(whoami))
        .with_state(state);

    let port = std::env::var("PORT").unwrap_or_else(|_| "3000".to_string());
    let addr = format!("127.0.0.1:{port}");
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();
    println!("服务已启动: http://{addr}");
    println!("试试:");
    println!("  curl http://{addr}/hello/Rust");
    println!("  curl http://{addr}/users/1");
    println!("  curl 'http://{addr}/search?q=rust&page=2'");
    println!(
        "  curl -X POST http://{addr}/echo -H 'content-type: application/json' -d '{{\"msg\":\"hi\"}}'"
    );
    axum::serve(listener, app).await.unwrap();
}

async fn root() -> &'static str {
    "Rust + Axum 示例服务"
}

async fn health() -> Json<Value> {
    Json(json!({ "status": "ok" }))
}

// Path 提取器：/hello/Rust -> "Rust"
async fn hello(Path(name): Path<String>) -> String {
    format!("Hello, {name}!")
}

async fn list_users(State(state): State<AppState>) -> Json<Vec<User>> {
    let users = state.users.read().unwrap();
    let mut list: Vec<User> = users.values().cloned().collect();
    list.sort_by_key(|user| user.id);
    Json(list)
}

async fn get_user(
    Path(id): Path<u64>,
    State(state): State<AppState>,
) -> Result<Json<User>, (StatusCode, Json<Value>)> {
    let users = state.users.read().unwrap();
    match users.get(&id) {
        Some(user) => Ok(Json(user.clone())),
        None => Err((
            StatusCode::NOT_FOUND,
            Json(json!({ "error": format!("用户 {id} 不存在") })),
        )),
    }
}

// 返回 (StatusCode, Json<T>) 可以直接控制状态码
async fn create_user(
    State(state): State<AppState>,
    Json(payload): Json<CreateUser>,
) -> Result<(StatusCode, Json<User>), (StatusCode, Json<Value>)> {
    if !payload.email.contains('@') {
        return Err((
            StatusCode::BAD_REQUEST,
            Json(json!({ "error": "邮箱格式不正确" })),
        ));
    }

    let mut users = state.users.write().unwrap();
    let id = users.keys().max().copied().unwrap_or(0) + 1;
    let user = User {
        id,
        name: payload.name,
        email: payload.email,
    };
    users.insert(id, user.clone());
    Ok((StatusCode::CREATED, Json(user)))
}

async fn search(Query(params): Query<SearchParams>) -> Json<Value> {
    Json(json!({
        "query": params.q.unwrap_or_default(),
        "page": params.page.unwrap_or(1),
        "note": "真实项目这里会查询数据库"
    }))
}

// Json 提取器必须放在最后一个参数
async fn echo(Json(body): Json<Value>) -> Json<Value> {
    Json(json!({ "received": body }))
}

// 直接读取请求头
async fn whoami(headers: HeaderMap) -> String {
    let user_agent = headers
        .get(header::USER_AGENT)
        .and_then(|value| value.to_str().ok())
        .unwrap_or("unknown");
    format!("你的 User-Agent: {user_agent}")
}
