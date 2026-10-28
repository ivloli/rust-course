// 第 12 章的 Go 平行版本：只用标准库 net/http，HTTP 行为与
// ch12_axum_basics/src/main.rs 一一对应。
//
// 运行：
//
//	cd ch12_axum_basics/go && go run .
//	PORT=8080 go run .
//
// 从 Go 看 Axum 的对照关系：
//
//	http.ServeMux             <-> axum::Router + .route(...)
//	"GET /hello/{name}"       <-> .route("/hello/{name}", get(hello))
//	r.PathValue("name")       <-> Path<String> 提取器
//	r.URL.Query()             <-> Query<SearchParams> 提取器
//	json.NewDecoder(r.Body)   <-> Json<CreateUser> 提取器
//	r.Header.Get("User-Agent")<-> HeaderMap 提取器
//	*AppState + sync.RWMutex  <-> State<AppState> + Arc<RwLock<HashMap>>
//	writeJSON(w, code, v)     <-> (StatusCode, Json<T>) / IntoResponse
//
// 标准库没有 serde：JSON 字段名靠 struct tag，Content-Type、JSON 语法、
// 缺字段、路径/查询参数类型都要 handler 自己检查；Axum 的提取器会自动拒绝。
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// User 对应 Rust 的 struct User；json tag 起 serde 派生的作用。
type User struct {
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUser 对应 Rust 的 #[derive(Deserialize)] struct CreateUser。
// 字段用指针是为了区分「缺字段」和「空字符串」：serde 会把缺字段当成
// 422 反序列化错误，而普通 string 字段只会静默拿到零值 ""。
type CreateUser struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// AppState 对应 Rust 的 State<AppState> + Arc<RwLock<HashMap<u64, User>>>。
// Go 没有 async 运行时，用 sync.RWMutex 保护内存表即可；handler 不跨 await 持锁。
type AppState struct {
	mu    sync.RWMutex
	users map[uint64]User
}

// errUnsupportedMediaType 是手工模拟 Axum Json 提取器 415 拒绝的哨兵错误。
var errUnsupportedMediaType = errors.New("unsupported media type")

func main() {
	state := &AppState{
		users: map[uint64]User{
			1: {ID: 1, Name: "Alice", Email: "alice@example.com"},
		},
	}

	// ServeMux 的 "METHOD /path" 模式对应 Router::route + get/post。
	// "GET /{$}" 中的 {$} 表示只匹配根路径，否则 "GET /" 会吞掉所有未知 GET。
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", state.root)
	mux.HandleFunc("GET /health", state.health)
	mux.HandleFunc("GET /hello/{name}", state.hello)
	mux.HandleFunc("GET /users", state.listUsers)
	mux.HandleFunc("POST /users", state.createUser)
	mux.HandleFunc("GET /users/{id}", state.getUser)
	mux.HandleFunc("GET /search", state.search)
	mux.HandleFunc("POST /echo", state.echo)
	mux.HandleFunc("GET /whoami", state.whoami)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	addr := "127.0.0.1:" + port

	// 先 Listen 再打印，和 Rust 的 TcpListener::bind 后 println 一致：
	// 端口被占用时不会打出“已启动”的假日志。
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听 %s 失败: %v", addr, err)
	}
	fmt.Printf("服务已启动: http://%s\n", addr)
	fmt.Println("试试:")
	fmt.Printf("  curl http://%s/hello/Rust\n", addr)
	fmt.Printf("  curl http://%s/users/1\n", addr)
	fmt.Printf("  curl 'http://%s/search?q=rust&page=2'\n", addr)
	fmt.Printf("  curl -X POST http://%s/echo -H 'content-type: application/json' -d '{\"msg\":\"hi\"}'\n", addr)

	server := &http.Server{Handler: mux}
	if err := server.Serve(listener); err != nil {
		log.Fatal(err)
	}
}

func (s *AppState) root(w http.ResponseWriter, r *http.Request) {
	// axum 的 &'static str 会变成 text/plain; charset=utf-8。
	writeText(w, http.StatusOK, "Rust + Axum 示例服务")
}

func (s *AppState) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// hello 对应 async fn hello(Path(name): Path<String>)。
// Go 1.22 的 {name} 通配符与 axum 0.8 的 {name} 语法一致，值从 PathValue 取。
func (s *AppState) hello(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	writeText(w, http.StatusOK, "Hello, "+name+"!")
}

func (s *AppState) listUsers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	list := make([]User, 0, len(s.users))
	for _, user := range s.users {
		list = append(list, user)
	}
	s.mu.RUnlock()

	// Go map 和 Rust HashMap 都不保证顺序，返回前都要按 id 排序。
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	writeJSON(w, http.StatusOK, list)
}

// getUser 对应 Path<u64> + State<AppState> 的 handler。
func (s *AppState) getUser(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		// Axum 的 Path<u64> 在提取阶段就返回 400 纯文本；PathValue 永远给字符串，
		// 所以类型错误必须手写。这里刻意复刻 Axum 的报错文案。
		writeText(w, http.StatusBadRequest, fmt.Sprintf("Invalid URL: Cannot parse `%s` to a `u64`", rawID))
		return
	}

	s.mu.RLock()
	user, ok := s.users[id]
	s.mu.RUnlock()
	if !ok {
		// Axum: Err((StatusCode::NOT_FOUND, Json(json!({"error": ...}))))
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("用户 %d 不存在", id),
		})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// createUser 对应 State<AppState> + Json<CreateUser>。
// Axum 会先判断 Content-Type（415），再解析 JSON（语法错 400、字段错 422）；
// 标准库不会替你做这些，所以下面手写 415/400 分支。
func (s *AppState) createUser(w http.ResponseWriter, r *http.Request) {
	var payload CreateUser
	if err := decodeJSONBody(r, &payload); err != nil {
		if errors.Is(err, errUnsupportedMediaType) {
			writeText(w, http.StatusUnsupportedMediaType,
				"Expected request with `Content-Type: application/json`")
			return
		}
		writeText(w, http.StatusBadRequest, "Failed to parse the request body as JSON: "+err.Error())
		return
	}

	// serde 在字段缺失时返回 422；Go 必须自己用指针判空才能复刻这个状态码。
	if payload.Name == nil || payload.Email == nil {
		missing := "name"
		if payload.Name != nil {
			missing = "email"
		}
		writeText(w, http.StatusUnprocessableEntity,
			"Failed to deserialize the JSON body into the target type: missing field `"+missing+"`")
		return
	}
	name, email := *payload.Name, *payload.Email

	if !strings.Contains(email, "@") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "邮箱格式不正确"})
		return
	}

	// Rust 用写锁包住「算 max + 插入」，保证并发下 id 不重复；
	// Go 同样在 Lock 内完成，不能用 RLock。
	s.mu.Lock()
	var maxID uint64
	for id := range s.users {
		if id > maxID {
			maxID = id
		}
	}
	id := maxID + 1
	user := User{ID: id, Name: name, Email: email}
	s.users[id] = user
	s.mu.Unlock()

	// (StatusCode::CREATED, Json(user)) 对应“状态码 + encoder”。
	writeJSON(w, http.StatusCreated, user)
}

func (s *AppState) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := query.Get("q") // 缺少 q 时 Get 返回 ""，对应 Option::unwrap_or_default

	page := uint64(1) // 对应 Option<u32>::unwrap_or(1)
	if raw := query.Get("page"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			// Query<T> 在 Axum 里同样是“提取阶段 400”，Go 需要手动解析。
			writeText(w, http.StatusBadRequest,
				"Failed to deserialize query string: page: "+err.Error())
			return
		}
		page = parsed
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query": q,
		"page":  page,
		"note":  "真实项目这里会查询数据库",
	})
}

// echo 对应 Json<Value>：请求体原样包进 {"received": ...}。
// json.Decoder.UseNumber 避免把整数变成 float64，尽量保留 serde_json 的数字形态。
func (s *AppState) echo(w http.ResponseWriter, r *http.Request) {
	var body any
	if err := decodeJSONBody(r, &body); err != nil {
		if errors.Is(err, errUnsupportedMediaType) {
			writeText(w, http.StatusUnsupportedMediaType,
				"Expected request with `Content-Type: application/json`")
			return
		}
		writeText(w, http.StatusBadRequest, "Failed to parse the request body as JSON: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"received": body})
}

// whoami 对应 HeaderMap 提取器。
func (s *AppState) whoami(w http.ResponseWriter, r *http.Request) {
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}
	writeText(w, http.StatusOK, "你的 User-Agent: "+userAgent)
}

// decodeJSONBody 手工补上 Axum Json 提取器的第一道检查：Content-Type。
// 标准库的 json.Decoder 不看请求头，直接解析 body。
func decodeJSONBody(r *http.Request, dst any) error {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if !strings.HasPrefix(contentType, "application/json") &&
		!strings.Contains(contentType, "+json") {
		return errUnsupportedMediaType
	}
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	return decoder.Decode(dst)
}

// writeText 对应返回 &str / String 的 handler。
func writeText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(text))
}

// writeJSON 对应 Json<T> / (StatusCode, Json<T>)。
// 先编码到 buffer 再写，避免 Encoder.Encode 末尾多一个换行；
// SetEscapeHTML(false) 让 <、>、& 与 serde_json 一样不被转义。
func writeJSON(w http.ResponseWriter, status int, value any) {
	var buffer strings.Builder
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		log.Printf("JSON 编码失败: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	body := strings.TrimSuffix(buffer.String(), "\n")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
