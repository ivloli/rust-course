// Package agent 是 Rust 版 src/agent.rs 的 Go 平行实现。
package agent

import (
	"context"
	"fmt"

	"ch14_go/llm"
	"ch14_go/tools"
)

const systemPrompt = "你是一个简洁的中文助手。需要计算或查询天气时，必须调用提供的工具，不要凭空编造结果。"

// ToolTrace 对应 Rust 的 ToolTrace，JSON 字段名必须与 Rust 一致。
type ToolTrace struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
}

// Reply 对应 Rust 的 AgentReply。
type Reply struct {
	Answer string      `json:"answer"`
	Steps  int         `json:"steps"`
	Tools  []ToolTrace `json:"tools"`
}

// Agent 对应 Rust 的 Agent。
type Agent struct {
	llm      llm.Llm
	tools    *tools.Registry
	MaxSteps int
}

// New 对应 Agent::new，默认 max_steps = 6。
func New(model llm.Llm, registry *tools.Registry) *Agent {
	return &Agent{llm: model, tools: registry, MaxSteps: 6}
}

func (a *Agent) LLMName() string {
	return a.llm.Name()
}

// Run 是经典的 Agent 循环：
//  1. 把对话历史 + 工具清单发给模型
//  2. 模型要求调用工具 -> 本地执行 -> 结果写回历史
//  3. 模型不再调用工具 -> 返回最终回答
//
// Rust 的 &mut Vec<Message> 在 Go 里用 *[]Message：append 可能换底层数组，
// 必须把新 slice 写回调用方，否则会话历史会丢。
func (a *Agent) Run(ctx context.Context, history *[]llm.Message, userInput string) (Reply, error) {
	if len(*history) == 0 {
		*history = append(*history, llm.System(systemPrompt))
	}
	*history = append(*history, llm.User(userInput))

	specs := a.tools.Specs()
	trace := make([]ToolTrace, 0) // 初始化成空切片，JSON 才会是 [] 而不是 null

	for step := 1; step <= a.MaxSteps; step++ {
		reply, err := a.llm.Chat(ctx, *history, specs)
		if err != nil {
			return Reply{}, err
		}
		calls := reply.ToolCalls
		*history = append(*history, reply) // 先把 assistant 消息写回历史

		if len(calls) == 0 {
			return Reply{Answer: reply.Content, Steps: step, Tools: trace}, nil
		}

		for _, call := range calls {
			result, err := a.tools.Call(ctx, call.Function.Name, call.Function.Arguments)
			if err != nil {
				// Rust 把错误文本回填给模型，而不是向上抛。
				result = "工具执行失败: " + err.Error()
			}
			trace = append(trace, ToolTrace{
				Name:      call.Function.Name,
				Arguments: call.Function.Arguments,
				Result:    result,
			})
			*history = append(*history, llm.Tool(call.ID, result))
		}
	}

	return Reply{}, fmt.Errorf(
		"超过最大步数 %d 仍未得到最终答案（可能是工具调用死循环）",
		a.MaxSteps,
	)
}
