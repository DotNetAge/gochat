package openai

import (
	"bytes"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
)

func TestMessagesToWire(t *testing.T) {
	// 测试基本文本消息
	messages := []core.Message{
		core.NewUserMessage("Hello"),
		core.NewSystemMessage("You are helpful"),
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 2)
	assert.Equal(t, core.RoleUser, wireMessages[0].Role)
	assert.Equal(t, "Hello", wireMessages[0].Content)
	assert.Equal(t, core.RoleSystem, wireMessages[1].Role)
	assert.Equal(t, "You are helpful", wireMessages[1].Content)

	// 测试带系统提示的消息
	wireMessages2 := MessagesToWire(messages, "System prompt")
	assert.Len(t, wireMessages2, 3)
	assert.Equal(t, core.RoleSystem, wireMessages2[0].Role)
	assert.Equal(t, "System prompt", wireMessages2[0].Content)

	// 测试多模态消息
	messages3 := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{Type: core.ContentTypeText, Text: "Check this image"},
				{Type: core.ContentTypeImage, MediaType: "image/png", Data: "base64data"},
			},
		},
	}
	wireMessages3 := MessagesToWire(messages3, "")
	assert.Len(t, wireMessages3, 1)
	contentParts, ok := wireMessages3[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, contentParts, 2)
	assert.Equal(t, "text", contentParts[0].Type)
	assert.Equal(t, "image_url", contentParts[1].Type)
}

func TestToolsToWire(t *testing.T) {
	tools := []core.Tool{
		{
			Name:        "calculator",
			Description: "Perform calculations",
			Parameters:  []byte(`{"type":"object","properties":{}}`),
		},
	}

	wireTools := ToolsToWire(tools)
	assert.Len(t, wireTools, 1)
	assert.Equal(t, "function", wireTools[0].Type)
	assert.Equal(t, "calculator", wireTools[0].Function.Name)
	assert.Equal(t, "Perform calculations", wireTools[0].Function.Description)
}

func TestResponseFromWire(t *testing.T) {
	// 测试基本响应
	resp := &ChatCompletionResponse{
		ID:    "chatcmpl-123",
		Model: "gpt-4",
		Choices: []Choice{
			{
				Message: Message{
					Role:    "assistant",
					Content: "Hello world",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	coreResp := ResponseFromWire(resp)
	assert.Equal(t, "chatcmpl-123", coreResp.ID)
	assert.Equal(t, "gpt-4", coreResp.Model)
	assert.Equal(t, "Hello world", coreResp.Content)
	assert.Equal(t, "stop", coreResp.FinishReason)
	assert.NotNil(t, coreResp.Usage)
	assert.Equal(t, 15, coreResp.Usage.TotalTokens)

	// 测试工具调用
	respWithTools := &ChatCompletionResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{
							ID:   "call_123",
							Type: "function",
						},
					},
				},
			},
		},
	}
	respWithTools.Choices[0].Message.ToolCalls[0].Function.Name = "get_weather"
	respWithTools.Choices[0].Message.ToolCalls[0].Function.Arguments = `{"location":"London"}`
	coreRespWithTools := ResponseFromWire(respWithTools)
	assert.Len(t, coreRespWithTools.ToolCalls, 1)
	assert.Equal(t, "get_weather", coreRespWithTools.ToolCalls[0].Name)

	// 测试空响应
	emptyResp := ResponseFromWire(nil)
	assert.NotNil(t, emptyResp)
}

func TestParseSSEStream_EdgeCases(t *testing.T) {
	// 测试格式错误的 SSE
	sseData := "invalid data\n\n"
	reader := bytes.NewReader([]byte(sseData))
	ch := ParseSSEStream(reader)
	for range ch {
		// Should not panic or return invalid chunks
	}

	// 测试空数据
	sseData2 := "data: \n\n"
	reader2 := bytes.NewReader([]byte(sseData2))
	ch2 := ParseSSEStream(reader2)
	for range ch2 {
	}
}

func TestStreamChunkToEvent(t *testing.T) {
	// 测试内容事件
	chunk := StreamChunk{
		Choices: []StreamChoice{
			{
				Delta: &Delta{
					Content: "Hello",
				},
			},
		},
	}

	event := StreamChunkToEvent(chunk)
	assert.Equal(t, core.EventContent, event.Type)
	assert.Equal(t, "Hello", event.Content)

	// 测试思考事件
	chunk2 := StreamChunk{
		Choices: []StreamChoice{
			{
				Delta: &Delta{
					ReasoningContent: "I'm thinking",
				},
			},
		},
	}

	event2 := StreamChunkToEvent(chunk2)
	assert.Equal(t, core.EventThinking, event2.Type)
	assert.Equal(t, "I'm thinking", event2.Content)

	// 测试完成事件
	chunk3 := StreamChunk{
		Choices: []StreamChoice{
			{
				FinishReason: "stop",
			},
		},
	}

	event3 := StreamChunkToEvent(chunk3)
	assert.Equal(t, core.EventDone, event3.Type)
}
