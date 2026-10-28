// 命令行入口，对应 Rust 的 src/bin/agent.rs。
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"ch14_go/agent"
	"ch14_go/llm"
	"ch14_go/tools"
)

func main() {
	// Rust: Agent::new(from_env(), ToolRegistry::default_tools())
	cli := agent.New(llm.FromEnv(), tools.DefaultTools())

	// Rust: env::args().skip(1).collect::<Vec<_>>().join(" ")
	prompt := strings.Join(os.Args[1:], " ")
	if strings.TrimSpace(prompt) == "" {
		prompt = "3 乘以 7 等于多少？北京天气怎么样？"
	}

	fmt.Printf("LLM = %s\n", cli.LLMName())
	fmt.Printf("用户 = %s\n\n", prompt)

	// Rust: let mut history = Vec::new(); run(&mut history, ...)
	history := make([]llm.Message, 0)
	reply, err := cli.Run(context.Background(), &history, prompt)
	if err != nil {
		// Rust 的 main 返回 anyhow::Result，失败时打印 "Error: ..." 并退出 1。
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, trace := range reply.Tools {
		fmt.Printf("[工具] %s(%s) -> %s\n", trace.Name, trace.Arguments, trace.Result)
	}
	fmt.Printf("\n[助手] %s\n", reply.Answer)
	fmt.Printf("（共 %d 步，会话历史 %d 条消息）\n", reply.Steps, len(history))
}
