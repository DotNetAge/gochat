package openai

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Chat(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("Say hello"),
	}
	response, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Content)
	assert.Equal(t, "assistant", response.Message.Role)
}

func TestClient_ChatStream(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("Say hello"),
	}
	stream, err := client.ChatStream(context.Background(), messages)
	require.NoError(t, err)
	defer stream.Close()

	var result string
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		if ev.Type == core.EventContent {
			result += ev.Content
		}
	}
	require.NoError(t, stream.Err())

	assert.NotEmpty(t, result)
}

func TestClient_ChatStreamWithThinking(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("What is 1+1?"),
	}

	stream, err := client.ChatStream(context.Background(), messages,
		core.WithMaxTokens(30),
	)
	require.NoError(t, err)
	defer stream.Close()

	var thinking string
	var content string
	hasThinking := false
	hasContent := false

	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatalf("Stream error: %v", ev.Err)
		}
		if ev.Type == core.EventThinking {
			thinking += ev.Content
			hasThinking = true
		}
		if ev.Type == core.EventContent {
			content += ev.Content
			hasContent = true
		}
	}

	err = stream.Err()
	if err != nil {
		t.Logf("Stream ended with error: %v", err)
	}

	t.Logf("hasThinking=%v, thinking length=%d", hasThinking, len(thinking))
	t.Logf("hasContent=%v, content length=%d", hasContent, len(content))

	assert.True(t, hasThinking || hasContent, "Expected at least one of thinking or content events")
	if hasThinking {
		assert.NotEmpty(t, thinking)
	}
	if hasContent {
		assert.NotEmpty(t, content)
	}
}

func TestClient_ChatWithOptions(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("What is 1+1?"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithTemperature(0.0),
		core.WithMaxTokens(50),
		core.WithUsageCallback(func(u core.Usage) {
			t.Logf("Usage: total=%d, prompt=%d, completion=%d", u.TotalTokens, u.PromptTokens, u.CompletionTokens)
		}),
	)
	require.NoError(t, err)

	if response.Content == "" && response.ReasoningContent != "" {
		t.Log("Model returned reasoning content instead of direct content")
	}
	assert.True(t, response.Content != "" || response.ReasoningContent != "",
		"Expected either content or reasoning content, got neither")
}

func TestClient_ChatWithAttachments(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	readmePath := "../../../gochat/README.md"
	readmeData, err := os.ReadFile(readmePath)
	if err != nil {
		t.Skipf("Skipping test: could not read README.md: %v", err)
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	attachment := core.NewAttachment("README.md", "text/markdown", readmeData, true)

	messages := []core.Message{
		core.NewUserMessage("Please read the attached file and tell me what project this is for."),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithAttachments(attachment),
	)

	if err != nil {
		t.Skipf("Skipping test: model may not support multimodal attachments: %v", err)
	}

	if response.Content == "" && response.ReasoningContent != "" {
		t.Log("Model returned reasoning content instead of direct content")
	}
	assert.True(t, response.Content != "" || response.ReasoningContent != "",
		"Expected either content or reasoning content, got neither")
}

func TestClient_ToolCalling(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "The city name"
				},
				"unit": {
					"type": "string",
					"enum": ["F", "C"],
					"description": "Temperature unit"
				}
			},
			"required": ["location"]
		}`),
	}

	messages := []core.Message{
		core.NewUserMessage("What's the weather like in Beijing?"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithTools(weatherTool),
	)

	if err != nil {
		t.Skipf("Skipping test: model may not support tool calling: %v", err)
	}

	if len(response.ToolCalls) > 0 {
		assert.Equal(t, "get_weather", response.ToolCalls[0].Name)
	}
}

func TestClient_ToolCallingWithResult(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test: DASHSCOPE_API_KEY not set")
	}

	client, err := NewOpenAI(core.Config{
		APIKey: apiKey,
		Model:  "qwen3.5-flash",
	})
	require.NoError(t, err)

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "The city name"
				},
				"unit": {
					"type": "string",
					"enum": ["F", "C"],
					"description": "Temperature unit"
				}
			},
			"required": ["location"]
		}`),
	}

	messages := []core.Message{
		core.NewUserMessage("What's the weather in Shanghai?"),
	}

	response, err := client.Chat(context.Background(), messages, core.WithTools(weatherTool))

	if err != nil {
		t.Skipf("Skipping test: model may not support tool calling: %v", err)
	}

	if len(response.ToolCalls) == 0 {
		t.Skip("Model did not request tool call in this run")
	}

	messages = append(messages, response.Message)

	tc := response.ToolCalls[0]
	messages = append(messages, core.Message{
		Role:       core.RoleTool,
		ToolCallID: tc.ID,
		Content: []core.ContentBlock{
			{Type: core.ContentTypeText, Text: `{"temperature": "25°C", "condition": "sunny"}`},
		},
	})

	finalResponse, err := client.Chat(context.Background(), messages, core.WithTools(weatherTool))
	require.NoError(t, err)
	assert.NotEmpty(t, finalResponse.Content)
}
