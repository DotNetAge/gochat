package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/compatible"
	"github.com/DotNetAge/gochat/pkg/core"
)

func getDashScopeClient(t *testing.T, model string) *compatible.Client {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY is not set. Skipping test.")
	}
	client, err := compatible.New(compatible.Config{
		Config: base.Config{
			APIKey:  apiKey,
			Model:   model,
			BaseURL: "https://dashscope.aliyuncs.com/compatible-mode",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}

// 1. 测试多轮对话状态流转
func TestMultiTurnChat(t *testing.T) {
	fmt.Println("\n=== Test 1: Multi-turn Conversation (Model: qwen-plus) ===")
	client := getDashScopeClient(t, "qwen-plus")
	ctx := context.Background()

	// 第一轮
	fmt.Println("User: 我有3个苹果和2个香蕉。")
	messages := []core.Message{
		core.NewUserMessage("我有3个苹果和2个香蕉。"),
	}

	resp1, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Turn 1 failed: %v", err)
	}
	fmt.Printf("Assistant: %s\n", resp1.Content)

	// 将助手的回复加入历史
	messages = append(messages, core.Message{
		Role: core.RoleAssistant,
		Content: []core.ContentBlock{
			{Type: core.ContentTypeText, Text: resp1.Content},
		},
	})

	// 第二轮
	fmt.Println("\nUser: 我一共有多少个水果？")
	messages = append(messages, core.NewUserMessage("我一共有多少个水果？"))

	resp2, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Turn 2 failed: %v", err)
	}
	fmt.Printf("Assistant: %s\n", resp2.Content)
	fmt.Println("=== Multi-turn Test Completed ===")
}

// 2. 测试函数调用 (Tool Calling)
func TestToolCalling(t *testing.T) {
	fmt.Println("\n=== Test 2: Tool/Function Calling (Model: qwen-max) ===")
	client := getDashScopeClient(t, "qwen-max")
	ctx := context.Background()

	tools := []core.Tool{
		{
			Name: "get_weather",
			Description: "获取指定城市的天气情况",
			Parameters: json.RawMessage(`{"type": "object", "properties": {"city": {"type": "string", "description": "城市名称，例如：杭州、北京"}}, "required": ["city"]}`),
		},
	}

	messages := []core.Message{
		core.NewUserMessage("杭州今天天气怎么样？"),
	}

	fmt.Println("User: 杭州今天天气怎么样？")
	
	resp, err := client.Chat(ctx, messages, core.WithTools(tools...))
	if err != nil {
		t.Fatalf("Chat request failed: %v", err)
	}

	if len(resp.ToolCalls) == 0 {
		t.Fatalf("Expected model to call a tool, but it didn't. Response: %s", resp.Content)
	}

	tc := resp.ToolCalls[0]
	fmt.Printf("Model decided to call tool: %s\n", tc.Name)
	fmt.Printf("With arguments: %s\n", tc.Arguments)

	fmt.Println("--> Executing local function get_weather()...")
	mockWeatherResult := `{"temperature": "25°C", "condition": "晴天", "city": "杭州"}`
	fmt.Printf("--> Function returned: %s\n", mockWeatherResult)

	messages = append(messages, core.Message{
		Role: core.RoleAssistant,
		ToolCalls: []core.ToolCall{tc},
	})

	messages = append(messages, core.Message{
		Role:       core.RoleTool,
		ToolCallID: tc.ID,
		Content: []core.ContentBlock{
			{Type: core.ContentTypeText, Text: mockWeatherResult},
		},
	})

	finalResp, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Final chat request failed: %v", err)
	}
	fmt.Printf("\nFinal Assistant Response: %s\n", finalResp.Content)
	fmt.Println("=== Tool Calling Test Completed ===")
}

// 3. 测试深度思考 (Deep Thinking / Reasoning)
func TestDeepThinking(t *testing.T) {
	fmt.Println("\n=== Test 3: Deep Thinking (Model: deepseek-r1 via DashScope) ===")
	client := getDashScopeClient(t, "deepseek-r1")
	ctx := context.Background()

	messages := []core.Message{
		core.NewUserMessage("10个橘子，我吃了一个，送给朋友两个，还剩几个？请一步步推导。"),
	}

	fmt.Println("User: 10个橘子，我吃了一个，送给朋友两个，还剩几个？请一步步推导。")
	
	stream, err := client.ChatStream(ctx, messages, core.WithThinking(0))
	if err != nil {
		t.Fatalf("Stream request failed: %v", err)
	}
	defer stream.Close()

	fmt.Print("\n[🤔 Thinking Process]:\n")
	isThinking := false
	isAnswering := false

	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatalf("Stream error: %v", ev.Err)
		}

		if ev.Type == core.EventThinking {
			if !isThinking {
				isThinking = true
			}
			fmt.Print(ev.Content)
		} else if ev.Type == core.EventContent {
			if !isAnswering {
				fmt.Print("\n\n[💡 Final Answer]:\n")
				isAnswering = true
			}
			fmt.Print(ev.Content)
		}
	}
	fmt.Println("\n\n=== Deep Thinking Test Completed ===")
}


// 4. 测试文本文件读取与内容分析 (Document Reading & Analysis)
func TestDocumentReading(t *testing.T) {
	fmt.Println("\n=== Test 4: Document Reading & Analysis (Model: qwen-plus) ===")
	client := getDashScopeClient(t, "qwen-plus")
	ctx := context.Background()

	// 1. 创建一个临时的测试文本文件
	tempFile := "test_document.txt"
	fileContent := `项目名称：GoChat 架构重构方案
日期：2026年3月
核心目标：
1. 统一各大模型服务商的接口（OpenAI, Anthropic, Qwen, Ollama等）。
2. 支持流式响应 (Streaming) 与多模态 (Multimodal)。
3. 原生支持深度思考过程 (Deep Thinking / Reasoning Chain) 的截获。
4. 增强 Tool Calling 函数调用能力。
进度：目前已全部完成并测试通过。`

	err := os.WriteFile(tempFile, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile) // 测试结束后清理文件

	// 2. 读取文件内容
	contentBytes, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// 3. 构建给大模型的消息
	prompt := fmt.Sprintf("请阅读以下名为 '%s' 的文件内容，并用一句话总结它的核心目标：\n\n<file_content>\n%s\n</file_content>", tempFile, string(contentBytes))
	
	messages := []core.Message{
		core.NewSystemMessage("你是一个专业的文件阅读与内容总结助手。"),
		core.NewUserMessage(prompt),
	}

	fmt.Printf("User: 请阅读文件 '%s' 并一句话总结。\n", tempFile)
	
	// 4. 发送给大模型进行分析
	resp, err := client.Chat(ctx, messages)
	if err != nil {
		t.Fatalf("Chat request failed: %v", err)
	}

	fmt.Printf("\nAssistant: %s\n", resp.Content)
	fmt.Println("=== Document Reading Test Completed ===")
}
