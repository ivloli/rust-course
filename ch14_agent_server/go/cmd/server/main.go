// HTTP 服务，对应 Rust 的 src/bin/server.rs（Axum -> net/http）。
//
// 路由与 Rust 一致：
//
//	GET  /health
//	POST /chat
//	POST /chat/stream   (Server-Sent Events)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"ch14_go/agent"
	"ch14_go/llm"
	"ch14_go/tools"
)

// session 对应 Rust 的 Arc<Mutex<Vec<Message>>>。
// mutex 保护单个会话的历史：同一 session 串行，不同 session 互不阻塞。
type session struct {
	mu      sync.Mutex
	history []llm.Message
}

// appState 对应 Rust 的 AppState。
// Rust 用 RwLock<HashMap<String, Session>>；Go 用「全局 Mutex + map」，
// 只在查找/创建会话时短暂加锁。
type appState struct {
	agent    *agent.Agent
	mu       sync.Mutex
	sessions map[string]*session
}

func newAppState() *appState {
	return &appState{
		agent:    agent.New(llm.FromEnv(), tools.DefaultTools()),
		sessions: make(map[string]*session),
	}
}

// session 对应 Rust 的 async fn session(...)：不存在则创建。
func (s *appState) session(id string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.sessions[id]; ok {
		return existing
	}
	created := &session{history: make([]llm.Message, 0)}
	s.sessions[id] = created
	return created
}

func (s *appState) sessionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// chatRequest 对应 Rust 的 ChatRequest。
// Rust 的 Option<String> 用 *string 模拟：区分「没传」与「传了空串」。
type chatRequest struct {
	SessionID *string `json:"session_id"`
	Message   string  `json:"message"`
}

// chatResponse 对应 Rust 的 ChatResponse，字段名与顺序保持一致。
type chatResponse struct {
	SessionID string            `json:"session_id"`
	Answer    string            `json:"answer"`
	Steps     int               `json:"steps"`
	Tools     []agent.ToolTrace `json:"tools"`
}

// errorBody 对应 Rust 的 json!({ "error": message })。
type errorBody struct {
	Error string `json:"error"`
}

func (s *appState) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"llm":      s.agent.LLMName(),
		"sessions": s.sessionCount(),
	})
}

func (s *appState) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	request, ok := decodeChatRequest(w, r)
	if !ok {
		return
	}
	sessionID := "default"
	if request.SessionID != nil {
		sessionID = *request.SessionID
	}

	current := s.session(sessionID)
	current.mu.Lock()
	reply, err := s.agent.Run(r.Context(), &current.history, request.Message)
	current.mu.Unlock()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, chatResponse{
		SessionID: sessionID,
		Answer:    reply.Answer,
		Steps:     reply.Steps,
		Tools:     reply.Tools,
	})
}

// handleChatStream 对应 Rust 的 chat_stream：
// Agent 先完成推理，再把最终答案按 rune 每 6 个一组推送，最后发 [DONE]。
func (s *appState) handleChatStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	request, ok := decodeChatRequest(w, r)
	if !ok {
		return
	}
	sessionID := "default"
	if request.SessionID != nil {
		sessionID = *request.SessionID
	}

	current := s.session(sessionID)
	current.mu.Lock()
	reply, err := s.agent.Run(r.Context(), &current.history, request.Message)
	current.mu.Unlock()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: err.Error()})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "当前 ResponseWriter 不支持流式输出"})
		return
	}

	// 必须在 WriteHeader 之前设置头；text/event-stream 是 SSE 的约定。
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	// Rust: answer.chars().collect::<Vec<_>>().chunks(6)
	runes := []rune(reply.Answer)
	chunks := make([]string, 0, len(runes)/6+1)
	for i := 0; i < len(runes); i += 6 {
		end := i + 6
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	chunks = append(chunks, "[DONE]")

	for _, chunk := range chunks {
		select {
		case <-r.Context().Done(): // 客户端断开：对应 Rust 中 Stream 被 drop
			return
		case <-time.After(60 * time.Millisecond):
		}
		// Axum 的 Event::default().data(x) 输出格式就是 "data: x\n\n"。
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		flusher.Flush()
	}
}

func decodeChatRequest(w http.ResponseWriter, r *http.Request) (chatRequest, bool) {
	var request chatRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "读取请求体失败"})
		return request, false
	}
	if err := json.Unmarshal(body, &request); err != nil {
		// Axum 对 JSON 语法错误返回 400，对字段缺失返回 422；
		// Go 这里统一用 400，细节差异见 GO_COMPARISON.md。
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "请求 JSON 解析失败: " + err.Error()})
		return request, false
	}
	return request, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		http.Error(w, "内部错误", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	addr := "127.0.0.1:" + port

	state := newAppState()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", state.handleHealth)
	mux.HandleFunc("/chat", state.handleChat)
	mux.HandleFunc("/chat/stream", state.handleChatStream)

	server := &http.Server{Addr: addr, Handler: mux}

	// Rust: tokio::signal::ctrl_c；Go 同时处理 SIGINT 与 SIGTERM。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Agent 服务已启动: http://%s\n", addr)
	fmt.Printf(
		"非流式: curl -X POST http://%s/chat -H 'content-type: application/json' -d '{\"session_id\":\"s1\",\"message\":\"3 乘以 7 等于多少？\"}'\n",
		addr,
	)
	fmt.Printf(
		"流式:   curl -N -X POST http://%s/chat/stream -H 'content-type: application/json' -d '{\"session_id\":\"s1\",\"message\":\"北京天气怎么样？\"}'\n",
		addr,
	)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "服务启动失败: %v\n", err)
		os.Exit(1)
	}
}
