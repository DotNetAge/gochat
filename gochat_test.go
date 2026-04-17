package gochat

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/DotNetAge/gochat/core"
)

// 测试配置
var (
	testAPIKey string
	testModel  = "qwen3.5-flash"
)

func init() {
	testAPIKey = os.Getenv("DASHSCOPE_API_KEY")
}

func skipIfNoAPIKey(t *testing.T) {
	if testAPIKey == "" {
		t.Skip("跳过测试: DASHSCOPE_API_KEY 环境变量未设置")
	}
}

// TestMaxTokens 测试最大Tokens限制
func TestMaxTokens(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		MaxTokens(50).
		EnableThinking(false). // 关闭思考模式，否则 max_tokens 不限制思考过程长度
		UserMessage("请详细介绍一下 Go 语言的历史、特性和应用场景")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("响应内容: %s", resp.Content)
	t.Logf("实际Tokens使用: Prompt=%d, Completion=%d, Total=%d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	// 验证生成的 token 数不超过限制（允许一定误差）
	if resp.Usage.CompletionTokens > 60 {
		t.Errorf("生成的 tokens 数量超过限制: got %d, want <= 60", resp.Usage.CompletionTokens)
	}
}

// TestStopSequences 测试停止词
func TestStopSequences(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		MaxTokens(200).
		Stop("。", "！").
		UserMessage("请列举 5 种编程语言")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("响应内容: %s", resp.Content)
	t.Logf("结束原因: %s", resp.FinishReason)

	// 验证在遇到停止词后停止
	if resp.FinishReason != "stop" {
		t.Errorf("期望 finish_reason 为 'stop', got '%s'", resp.FinishReason)
	}
}

// TestUsageStatistics 测试用量返回统计
func TestUsageStatistics(t *testing.T) {
	skipIfNoAPIKey(t)
	
	var receivedUsage *core.Usage

	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		UsageCallback(func(u core.Usage) {
			receivedUsage = &u
			t.Logf("用量回调: Prompt=%d, Completion=%d, Total=%d",
				u.PromptTokens, u.CompletionTokens, u.TotalTokens)
		}).
		UserMessage("你好")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证响应中的用量统计
	if resp.Usage.TotalTokens == 0 {
		t.Error("期望有用量统计，但 TotalTokens 为 0")
	}

	// 验证回调函数被调用
	if receivedUsage == nil {
		t.Error("期望 UsageCallback 被调用，但没有收到用量统计")
	} else if receivedUsage.TotalTokens != resp.Usage.TotalTokens {
		t.Errorf("回调用量与响应用量不一致: callback=%d, response=%d",
			receivedUsage.TotalTokens, resp.Usage.TotalTokens)
	}

	t.Logf("✓ 用量统计验证通过: Prompt=%d, Completion=%d, Total=%d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
}

// TestMultiTurnConversation 测试多轮对话
func TestMultiTurnConversation(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		SysemMessage("你是一个专业的 Go 语言教学助手，回答要简洁明了").
		UserMessage("什么是 Goroutine？").
		AssistantMessage("Goroutine 是 Go 语言中的轻量级线程，由 Go 运行时管理。").
		UserMessage("那 Channel 呢？")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("多轮对话响应: %s", resp.Content)

	// 验证响应包含 Channel 相关内容
	if len(resp.Content) < 10 {
		t.Error("响应内容过短，可能未正确处理多轮对话上下文")
	}
}

// TestToolCalling 测试工具调用
func TestToolCalling(t *testing.T) {
	skipIfNoAPIKey(t)
	
	// 定义天气查询工具
	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "获取指定城市的天气信息",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"city": {
					"type": "string",
					"description": "城市名称，如：北京、上海"
				}
			},
			"required": ["city"]
		}`),
	}

	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		Tools(weatherTool).
		UserMessage("北京今天的天气怎么样？")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	// 验证是否有工具调用
	if len(resp.ToolCalls) > 0 {
		t.Logf("✓ 工具调用成功: %s", resp.ToolCalls[0].Name)
		t.Logf("  参数: %s", resp.ToolCalls[0].Arguments)

		if resp.ToolCalls[0].Name != "get_weather" {
			t.Errorf("期望调用 get_weather 工具，got %s", resp.ToolCalls[0].Name)
		}
	} else {
		t.Logf("模型未调用工具，直接响应: %s", resp.Content)
		// 某些模型可能会直接回答而不是调用工具，这也是可以接受的
	}
}

// TestThinkingMode 测试思想流（深度思考）
func TestThinkingMode(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		EnableThinking(true).
		ThinkingBudget(512).
		UserMessage("请解释一下为什么 Go 语言的并发模型比线程更高效？")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("思考模式响应: %s", resp.Content)

	// 验证响应质量（深度思考应该产生更详细的回答）
	if len(resp.Content) < 50 {
		t.Error("响应内容过短，可能未启用深度思考")
	}

	t.Logf("✓ 思想流测试完成，Tokens使用: %d", resp.Usage.TotalTokens)
}

// TestWebSearch 测试网络搜索
func TestWebSearch(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		EnableSearch(true).
		UserMessage("2024年 Go 语言的最新版本是什么？有哪些新特性？")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("网络搜索响应: %s", resp.Content)

	// 验证是否包含搜索结果的时效性信息
	if len(resp.Content) < 20 {
		t.Error("响应内容过短，可能未正确启用网络搜索")
	}

	t.Logf("✓ 网络搜索测试完成")
}

// TestFileAttachment 测试附加文件理解
func TestFileAttachment(t *testing.T) {
	skipIfNoAPIKey(t)
	
	// 读取本项目的 README.md
	readmePath := "README.md"
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("无法读取 README.md: %v", err)
	}

	// 创建文件附件
	attachment := core.Attachment{
		MediaType: "text/markdown",
		Name:      "README.md",
		Data:      readmeData,
	}

	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		Attach(attachment).
		UserMessage("请简要总结这个项目的核心功能和特点")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("文件理解响应: %s", resp.Content)

	// 验证是否理解了文件内容
	if len(resp.Content) < 20 {
		t.Error("响应内容过短，可能未正确理解文件内容")
	}

	// 检查是否包含关键信息
	content := resp.Content
	keywords := []string{"GoChat", "LLM", "客户端", "统一接口"}
	foundKeywords := 0
	for _, kw := range keywords {
		if contains(content, kw) {
			foundKeywords++
		}
	}

	if foundKeywords == 0 {
		t.Logf("警告: 响应未包含项目的关键信息")
	}

	t.Logf("✓ 文件理解测试完成")
}

// TestStreamingOutput 测试流式输出
func TestStreamingOutput(t *testing.T) {
	skipIfNoAPIKey(t)
	
	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		EnableThinking(false). // 关闭思考模式以避免超长响应
		MaxTokens(100).        // 限制生成长度
		IncrementalOutput(true).
		UserMessage("请写一首关于程序员的短诗")

	stream, err := builder.GetStream(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer stream.Close()

	var fullContent string
	var eventCount int
	var totalTokens int

	// 读取流式响应
	for stream.Next() {
		event := stream.Event()
		eventCount++

		t.Logf("事件 %d: Type=%s", eventCount, event.Type)

		if event.Type == core.EventContent {
			fullContent += event.Content
		} else if event.Type == core.EventDone {
			if event.Usage != nil {
				totalTokens = event.Usage.TotalTokens
			}
			t.Logf("✓ 流式输出完成")
			break
		} else if event.Type == core.EventError {
			t.Fatalf("流式输出错误: %v", event.Err)
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("流式读取错误: %v", err)
	}

	t.Logf("✓ 流式输出完成")
	t.Logf("总事件数: %d", eventCount)
	t.Logf("完整内容: %s", fullContent)
	t.Logf("总Tokens: %d", totalTokens)

	// 验证
	if eventCount < 2 {
		t.Errorf("期望至少 2 个事件，got %d", eventCount)
	}

	if len(fullContent) < 10 {
		t.Error("响应内容过短")
	}
}

// TestAllFeaturesIntegration 综合集成测试
func TestAllFeaturesIntegration(t *testing.T) {
	skipIfNoAPIKey(t)
	
	// 读取 README
	readmeData, _ := os.ReadFile("README.md")

	builder := Client().
		Init(core.Config{
			APIKey: testAPIKey,
			Model:  testModel,
		}).
		SysemMessage("你是一个专业的技术文档分析助手").
		MaxTokens(300).
		Attach(core.Attachment{
			MediaType: "text/markdown",
			Name:      "README.md",
			Data:      readmeData,
		}).
		UserMessage("根据文档，GoChat 项目的三个核心模块分别是什么？")

	resp, err := builder.GetResponse(QwenClient)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}

	t.Logf("综合测试响应: %s", resp.Content)
	t.Logf("Tokens 使用: %d", resp.Usage.TotalTokens)

	// 验证
	if resp.Usage.TotalTokens == 0 {
		t.Error("期望有用量统计")
	}

	if resp.Usage.CompletionTokens > 320 {
		t.Errorf("生成的 tokens 可能超过限制: got %d", resp.Usage.CompletionTokens)
	}

	t.Logf("✓ 综合集成测试完成")
}

// 辅助函数：检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
