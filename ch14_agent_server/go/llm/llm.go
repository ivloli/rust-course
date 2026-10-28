// Package llm 是 Rust 版 src/llm.rs 的 Go 平行实现。
//
// Rust 用 #[async_trait] trait Llm 把「调用模型」抽象出来，Mock 与真实 API
// 都实现同一个 trait，再装进 Box<dyn Llm>；Go 用 interface + 普通方法，
// 异步则通过 context.Context 与阻塞调用表达（详见 GO_COMPARISON.md）。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Message 对应 Rust 的 struct Message，字段与 OpenAI Chat Completions 协议一致。
// Rust 用 Option<String> + skip_serializing_if；Go 用零值 + omitempty。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Rust 的 Message::system / user / assistant / tool 构造函数，
// 在 Go 里就是几个返回 Message 的普通函数。
func System(content string) Message {
	return Message{Role: "system", Content: content}
}

func User(content string) Message {
	return Message{Role: "user", Content: content}
}

func Assistant(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// AssistantWithToolCalls 对应 Rust 的 Message::assistant_with_tool_calls：
// content 为空，tool_calls 非空，JSON 里 content 会被 omitempty 省略。
func AssistantWithToolCalls(content string, toolCalls []ToolCall) Message {
	return Message{Role: "assistant", Content: content, ToolCalls: toolCalls}
}

// Tool 构造 role=tool 的消息，tool_call_id 指回 assistant 的那次调用。
func Tool(toolCallID, content string) Message {
	return Message{Role: "tool", Content: content, ToolCallID: toolCallID}
}

// FunctionCall 对应 Rust 的 FunctionCall。
// 模型返回的 arguments 是 JSON 字符串，需要工具自己解析。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolCall 对应 Rust 的 ToolCall。Rust 用 #[serde(rename = "type")]
// 把字段 kind 序列化成 "type"；Go 直接在 tag 里写 "type"。
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// nextCallID 对应 Rust 里的 static NEXT_ID: AtomicU64。
var nextCallID atomic.Uint64

// NewToolCall 生成 call_1、call_2…… 格式的 ID。
func NewToolCall(name, arguments string) ToolCall {
	id := nextCallID.Add(1)
	return ToolCall{
		ID:       fmt.Sprintf("call_%d", id),
		Type:     "function",
		Function: FunctionCall{Name: name, Arguments: arguments},
	}
}

// ToolSpec / FunctionSpec 对应 Rust 的同名结构，描述「工具长什么样」。
type ToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type FunctionSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

func FunctionToolSpec(name, description string, parameters any) ToolSpec {
	return ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// Llm 对应 Rust 的 trait Llm。
//
// Rust 的 async fn + #[async_trait] 在这里变成「带 ctx 的阻塞方法」；
// ctx 用来传递取消与超时，相当于 Rust 里 Future 被 drop 时取消。
type Llm interface {
	Chat(ctx context.Context, messages []Message, tools []ToolSpec) (Message, error)
	Name() string
}

// FromEnv 对应 Rust 的 from_env()：有 OPENAI_API_KEY 就走真实 API，
// 否则退回离线 Mock，二者接口完全一致。
func FromEnv() Llm {
	key := os.Getenv("OPENAI_API_KEY")
	if strings.TrimSpace(key) == "" {
		return MockLLM{}
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	return NewOpenAILLM(key, baseURL, model)
}

// OpenAILLM 对应 Rust 的 OpenAiLlm（reqwest -> net/http）。
type OpenAILLM struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
}

func NewOpenAILLM(apiKey, baseURL, model string) *OpenAILLM {
	// Rust 里 reqwest::Client::builder().timeout(60s)；
	// Go 用 http.Client.Timeout，注意它覆盖整个请求（连接 + 读 body）。
	return &OpenAILLM{
		client:  &http.Client{Timeout: 60 * time.Second},
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

type chatRequestBody struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Tools       []ToolSpec `json:"tools,omitempty"`
	Temperature float32    `json:"temperature"`
}

type chatResponseBody struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (o *OpenAILLM) Chat(ctx context.Context, messages []Message, tools []ToolSpec) (Message, error) {
	url := strings.TrimRight(o.baseURL, "/") + "/chat/completions"

	// Rust 用 if tools.is_empty() { None } else { Some(tools) }；
	// Go 用 nil slice + omitempty 达到同样的「空就不出现」。
	var specPayload []ToolSpec
	if len(tools) > 0 {
		specPayload = tools
	}
	body := chatRequestBody{
		Model:       o.model,
		Messages:    messages,
		Tools:       specPayload,
		Temperature: 0.2,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Message{}, fmt.Errorf("序列化 LLM 请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return Message{}, fmt.Errorf("构建 LLM 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey) // Rust: .bearer_auth(&self.api_key)

	resp, err := o.client.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("请求 LLM 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("读取 LLM 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Message{}, fmt.Errorf("LLM 返回错误状态 %s: %s", resp.Status, string(raw))
	}

	var parsed chatResponseBody
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Message{}, fmt.Errorf("解析 LLM 响应 JSON 失败: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return Message{}, fmt.Errorf("LLM 响应中没有 choices")
	}
	return parsed.Choices[0].Message, nil
}

func (o *OpenAILLM) Name() string {
	return o.model
}

// MockLLM 对应 Rust 的 MockLlm：用固定脚本演示
// 「工具调用 -> 工具结果 -> 最终回答」的完整循环。
type MockLLM struct{}

func (MockLLM) Name() string {
	return "mock"
}

func (MockLLM) Chat(ctx context.Context, messages []Message, _ []ToolSpec) (Message, error) {
	// Rust: tokio::time::sleep(40ms).await；Go: 可被 ctx 取消的 timer。
	if err := sleep(ctx, 40*time.Millisecond); err != nil {
		return Message{}, err
	}

	// Rust: messages.iter().rposition(...) 取最后一条 user 之后的部分。
	turn := messages
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			turn = messages[i:]
			break
		}
	}

	userText := ""
	if len(turn) > 0 {
		userText = turn[0].Content
	}

	toolResults := make([]string, 0, len(turn))
	for _, message := range turn {
		if message.Role == "tool" {
			toolResults = append(toolResults, message.Content)
		}
	}

	// 下面的判断顺序必须与 Rust 完全一致，否则会话步数会不同。
	wantsCalc := strings.Contains(userText, "乘") ||
		strings.Contains(userText, "*") ||
		strings.Contains(userText, "算") ||
		strings.Contains(userText, "等于")
	wantsWeather := strings.Contains(userText, "天气")
	hasCalc := false
	hasWeather := false
	for _, result := range toolResults {
		if !strings.Contains(result, "°C") {
			hasCalc = true
		}
		if strings.Contains(result, "°C") {
			hasWeather = true
		}
	}

	if wantsCalc && !hasCalc {
		args := marshalArgs(struct {
			A  float64 `json:"a"`
			B  float64 `json:"b"`
			Op string  `json:"op"`
		}{A: 3, B: 7, Op: "mul"})
		return AssistantWithToolCalls("", []ToolCall{NewToolCall("calculator", args)}), nil
	}

	if wantsWeather && !hasWeather {
		args := marshalArgs(struct {
			City string `json:"city"`
		}{City: "北京"})
		return AssistantWithToolCalls("", []ToolCall{NewToolCall("weather", args)}), nil
	}

	if len(toolResults) > 0 {
		return Assistant("我查到的结果：" + strings.Join(toolResults, "；") + "。"), nil
	}

	return Assistant("（Mock 模式）你好，我是离线演示用的模拟助手。设置 OPENAI_API_KEY 后我会调用真实模型。"), nil
}

// marshalArgs 对应 Rust 的 serde_json::json!({...}).to_string()：
// 紧凑 JSON、无空格，key 顺序由结构体字段声明决定。
func marshalArgs(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err) // 只用于常量结构体，不会失败
	}
	return string(data)
}

// sleep 把 time.Sleep 换成可取消版本：ctx 取消时提前返回，
// 对应 Rust 中 Future 被 drop 后不再继续。
func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
