package openaicompat

import (
	"bytes"
	"testing"

	"github.com/DotNetAge/gochat/pkg/core"
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

	// 测试空响应
	emptyResp := ResponseFromWire(nil)
	assert.NotNil(t, emptyResp)

	// 测试无选择的响应
	noChoicesResp := &ChatCompletionResponse{
		ID:    "chatcmpl-456",
		Model: "gpt-4",
		Usage: Usage{TotalTokens: 10},
	}

	coreResp2 := ResponseFromWire(noChoicesResp)
	assert.Equal(t, "chatcmpl-456", coreResp2.ID)
	assert.Equal(t, "gpt-4", coreResp2.Model)
}

func TestParseSSEStream(t *testing.T) {
	// 测试解析 SSE 流
	sseData := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: [DONE]
`

	reader := bytes.NewReader([]byte(sseData))
	ch := ParseSSEStream(reader)

	chunks := []StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	assert.Len(t, chunks, 2)
	assert.Equal(t, "chatcmpl-123", chunks[0].ID)
	assert.Equal(t, "chatcmpl-123", chunks[1].ID)
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
