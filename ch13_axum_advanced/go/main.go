// 第 13 章的 Go 平行版本：只用标准库 net/http，覆盖与
// ch13_axum_advanced/src/main.rs 相同的功能：路由、超时、CORS、日志、
// 统一错误 JSON、优雅关闭。
//
// 运行：
//
//	cd ch13_axum_advanced/go && go run .
//	PORT=8080 go run .
//
// 从 Go 看 Axum / tower 的对照关系：
//
//	http.ServeMux                  <-> Router + route
//	tower 的 .layer() 洋葱模型      <-> 一层层包住的 func(http.Handler) http.Handler
//	TimeoutLayer(408, 500ms)       <-> context.WithTimeout + goroutine + 缓冲 ResponseWriter
//	CorsLayer                      <-> cors 中间件（含 OPTIONS 预检）
//	TraceLayer / log_elapsed       <-> trace / logElapsed 中间件
//	IntoResponse for AppError      <-> AppError + writeAppError（状态码 + JSON 编码器）
//	with_graceful_shutdown         <-> signal.NotifyContext + http.Server.Shutdown
//
// 关键点：Rust 的 TimeoutLayer 超时后返 408 空 body；Go 标准库的
// http.TimeoutHandler 默认返 503，且响应头处理不透明，所以这里自己实现。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const requestTimeout = 500 * time.Millisecond

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", root)
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /slow", slow)
	mux.HandleFunc("GET /fail", fail)

	// 对应 Rust 里从内到外的 layer 顺序：
	//   Timeout 最内层 -> log_elapsed -> CORS -> Trace 最外层。
	// Go 没有 tower，用嵌套的函数包出来，最外层最后包。
	var handler http.Handler = timeoutMiddleware(requestTimeout)(mux)
	handler = logElapsed(handler)
	handler = cors(handler)
	handler = trace(handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	addr := "127.0.0.1:" + port

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听 %s 失败: %v", addr, err)
	}
	log.Printf("listening on http://%s", addr)
	fmt.Printf("试试: curl http://%s/slow   （500ms 超时，应返回 408）\n", addr)

	server := &http.Server{Handler: handler}

	// 对应 with_graceful_shutdown(shutdown_signal())。
	// 同时监听 SIGINT（Ctrl-C）和 SIGTERM（容器/k8s 常见）。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		log.Println("收到退出信号，开始优雅关闭：等待进行中的请求完成")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("优雅关闭超时: %v", err)
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	writeText(w, http.StatusOK, "Axum 进阶示例：/health /slow /fail")
}

func health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// slow 对应 async fn slow()：睡 2 秒后返回。
// 唯一的区别是要自己监听 ctx.Done()，否则超时后 goroutine 还会白等 2 秒。
func slow(w http.ResponseWriter, r *http.Request) {
	select {
	case <-time.After(2 * time.Second):
		writeText(w, http.StatusOK, "终于完成了")
	case <-r.Context().Done():
		// 上游 timeout 已返回 408，这里不再写响应。
		return
	}
}

// fail 对应返回 Err(AppError::NotFound("资源不存在")) 的 handler。
func fail(w http.ResponseWriter, r *http.Request) {
	writeAppError(w, &AppError{Status: http.StatusNotFound, Message: "资源不存在"})
}

// AppError 对应 Rust 的 enum AppError + impl IntoResponse。
// Go 没有 trait，统一错误靠一个显式类型和 writeAppError 辅助函数。
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

func writeAppError(w http.ResponseWriter, err *AppError) {
	writeJSON(w, err.Status, map[string]string{"error": err.Message})
}

// timeoutMiddleware 对应 TimeoutLayer::with_status_code(408, 500ms)。
//
// 做法：把 next 放到 goroutine 里跑，写进一个缓冲 ResponseWriter；
// 如果 500ms 内完成就把它复制给真实 ResponseWriter，否则直接写 408。
// 缓冲非常关键：超时后 handler 再写响应不会触发 "superfluous WriteHeader"，
// 也不会把 200 泄漏给已经收到 408 的客户端。
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			buffer := newBufferedResponse()
			done := make(chan struct{})

			go func() {
				defer close(done)
				next.ServeHTTP(buffer, r.WithContext(ctx))
			}()

			select {
			case <-done:
				buffer.flushTo(w)
			case <-ctx.Done():
				// 客户端主动断开时 r.Context() 也会取消，此时没必要再写响应。
				if errors.Is(ctx.Err(), context.DeadlineExceeded) && r.Context().Err() == nil {
					w.WriteHeader(http.StatusRequestTimeout)
				}
			}
		})
	}
}

// logElapsed 对应 middleware::from_fn(log_elapsed)。
// 它自己也用缓冲写入，确保先加上 x-response-time-ms 再提交响应头——
// 否则 /slow 的 408 已经发出，再 Set 头就无效了。
func logElapsed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		buffer := newBufferedResponse()
		next.ServeHTTP(buffer, r)

		elapsed := time.Since(started)
		buffer.header.Set("x-response-time-ms", strconv.FormatInt(elapsed.Milliseconds(), 10))
		buffer.flushTo(w)
		log.Printf("%s %s -> %d in %s", r.Method, r.URL.RequestURI(), buffer.statusCode(), elapsed)
	})
}

// cors 对应 CorsLayer：allow_origin(Any) + GET/POST + content-type。
// tower 的 CORS 对所有 OPTIONS 直接返回 200 预检响应，且它在 log_elapsed
// 外层，所以预检响应没有 x-response-time-ms —— 这里保持同样的顺序。
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Vary", "origin, access-control-request-method, access-control-request-headers")
		header.Set("Access-Control-Allow-Origin", "*")

		if r.Method == http.MethodOptions {
			header.Set("Access-Control-Allow-Methods", "GET,POST")
			header.Set("Access-Control-Allow-Headers", "content-type")
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// trace 对应 TraceLayer::new_for_http()：最外层，记录开始/结束与状态码。
// 它只是透传并记录状态，不做缓冲。
func trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("started %s %s", r.Method, r.URL.RequestURI())
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		log.Printf("finished %s %s -> %d", r.Method, r.URL.RequestURI(), recorder.statusCode())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(p)
}

func (r *statusRecorder) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

// bufferedResponse 是一个只存在于内存里的 ResponseWriter。
type bufferedResponse struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header)}
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(status int) {
	if b.wroteHeader {
		return
	}
	b.status = status
	b.wroteHeader = true
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

func (b *bufferedResponse) statusCode() int {
	if b.status == 0 {
		return http.StatusOK
	}
	return b.status
}

func (b *bufferedResponse) flushTo(w http.ResponseWriter) {
	for key, values := range b.header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(b.statusCode())
	if b.body.Len() > 0 {
		_, _ = w.Write(b.body.Bytes())
	}
}

func writeText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(text))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		log.Printf("JSON 编码失败: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	body := bytes.TrimSuffix(buffer.Bytes(), []byte("\n"))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
