// Package tools 是 Rust 版 src/tools.rs 的 Go 平行实现。
//
// Rust 的 trait Tool 同时描述「给模型看的 spec」与「给程序跑的 call」；
// Go 用 interface 表达同一件事。ToolRegistry 在两边都负责按名字分发。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"ch14_go/llm"
)

// Tool 对应 Rust 的 trait Tool。
// Rust 的 async fn call(&self, arguments: &str) -> Result<String>
// 在 Go 里是 Call(ctx, arguments) (string, error)。
type Tool interface {
	Spec() llm.ToolSpec
	Call(ctx context.Context, arguments string) (string, error)
}

// Registry 对应 Rust 的 ToolRegistry。
type Registry struct {
	tools []Tool
}

// DefaultTools 对应 ToolRegistry::default_tools()。
func DefaultTools() *Registry {
	return &Registry{tools: []Tool{Calculator{}, Weather{}}}
}

// Specs 对应 Rust 的 specs()：用 iter().map() 收集所有工具描述。
func (r *Registry) Specs() []llm.ToolSpec {
	specs := make([]llm.ToolSpec, 0, len(r.tools))
	for _, tool := range r.tools {
		specs = append(specs, tool.Spec())
	}
	return specs
}

// Call 按名字查找工具并执行。
// 未知工具返回 error，Agent 会把错误文本回填给模型，让它自我修正。
func (r *Registry) Call(ctx context.Context, name, arguments string) (string, error) {
	for _, tool := range r.tools {
		if tool.Spec().Function.Name == name {
			return tool.Call(ctx, arguments)
		}
	}
	return "", fmt.Errorf("未知工具: %s", name)
}

// Calculator 对应 Rust 的 Calculator。
type Calculator struct{}

// 用指针接收 *float64，模拟 Rust 反序列化时「缺字段就报错」的行为；
// Go 的普通 float64 缺字段会静默变成 0，这是两地 JSON 处理的常见差异。
type calculatorArgs struct {
	A  *float64 `json:"a"`
	B  *float64 `json:"b"`
	Op string   `json:"op"`
}

func (Calculator) Spec() llm.ToolSpec {
	return llm.FunctionToolSpec(
		"calculator",
		"对两个数字执行一次四则运算。op 取值: add / sub / mul / div",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a":  map[string]any{"type": "number", "description": "左操作数"},
				"b":  map[string]any{"type": "number", "description": "右操作数"},
				"op": map[string]any{"type": "string", "enum": []string{"add", "sub", "mul", "div"}},
			},
			"required": []string{"a", "b", "op"},
		},
	)
}

func (Calculator) Call(_ context.Context, arguments string) (string, error) {
	var args calculatorArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("解析计算器参数失败: %w", err)
	}
	if args.A == nil || args.B == nil {
		return "", fmt.Errorf("缺少参数 a 或 b")
	}
	a, b := *args.A, *args.B

	var value float64
	switch args.Op {
	case "add":
		value = a + b
	case "sub":
		value = a - b
	case "mul":
		value = a * b
	case "div":
		if b == 0 {
			return "", fmt.Errorf("除数不能为 0")
		}
		value = a / b
	default:
		return "", fmt.Errorf("不支持的运算: %s", args.Op)
	}

	// Rust: if value.fract() == 0.0 { format!("{}", value as i64) }
	// Go 用 math.Trunc 判断整数，再转 int64，保证 21 而不是 21.0。
	if value == math.Trunc(value) {
		return strconv.FormatInt(int64(value), 10), nil
	}
	return strconv.FormatFloat(value, 'g', -1, 64), nil
}

// Weather 对应 Rust 的 Weather：30ms 延迟后返回固定 mock 数据。
type Weather struct{}

type weatherArgs struct {
	City string `json:"city"`
}

func (Weather) Spec() llm.ToolSpec {
	return llm.FunctionToolSpec(
		"weather",
		"查询指定城市的当前天气（示例工具，返回 mock 数据）",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{"type": "string", "description": "城市名，例如：北京"},
			},
			"required": []string{"city"},
		},
	)
}

func (Weather) Call(ctx context.Context, arguments string) (string, error) {
	var args weatherArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("解析天气参数失败: %w", err)
	}
	if args.City == "" {
		return "", fmt.Errorf("缺少参数 city")
	}

	// Rust: tokio::time::sleep(30ms).await。
	timer := time.NewTimer(30 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timer.C:
	}

	return fmt.Sprintf("%s：晴，24°C，东风 2 级（mock 数据）", args.City), nil
}
