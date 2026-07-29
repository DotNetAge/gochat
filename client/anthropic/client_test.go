package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  core.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: core.Config{
				APIKey: "test-key",
				Model:  "claude-3-opus-20240229",
			},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: core.Config{
				APIKey: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAnthropic(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		var reqBody anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "claude-3-opus-20240229", reqBody.Model)

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "test response"},
			},
			Model:      "claude-3-opus-20240229",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{
		APIKey:  "test-key",
		Model:   "claude-3-opus-20240229",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test prompt"),
	}
	response, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)
	assert.Equal(t, "test response", response.Content)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody anthropicRequest
		json.NewDecoder(r.Body).Decode(&reqBody)

		assert.True(t, reqBody.Stream)

		flusher, _ := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"type":"content_block_start","index":0}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
			`data: {"type":"content_block_stop","index":0}`,
			`data: {"type":"message_delta","usage":{"output_tokens":20}}`,
			`data: {"type":"message_stop"}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{
		APIKey:  "test-key",
		Model:   "claude-3-opus-20240229",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test prompt"),
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

	assert.Equal(t, "Hello world", result)
}

func TestClient_ChatWithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Verify system prompt was set
		assert.Equal(t, "You are a helpful assistant", reqBody.System)

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "response"},
			},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 20},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithSystemPrompt("You are a helpful assistant"),
	)
	require.NoError(t, err)
	assert.Equal(t, "response", response.Content)
}

func TestClient_Chat_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`))
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		MaxRetries: 0,
	})
	require.NoError(t, err)

	messages := []core.Message{core.NewUserMessage("test")}
	_, err = client.Chat(context.Background(), messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication_error")
}

// TestClient_ChatToolCall 验证非流式工具调用：
// 1. tools 字段用 input_schema 发送（Anthropic 格式）
// 2. tool_use content block 正确转为 core.ToolCall
// 3. stop_reason "tool_use" 映射为 finish_reason "tool_calls"
func TestClient_ChatToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody anthropicRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))

		// 验证 tools 用 input_schema 发送
		require.Len(t, reqBody.Tools, 1)
		assert.Equal(t, "get_weather", reqBody.Tools[0].Name)
		assert.Equal(t, `{"type":"object","properties":{"location":{"type":"string"}}}`, string(reqBody.Tools[0].InputSchema))

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_abc",
					Name:  "get_weather",
					Input: map[string]interface{}{"location": "Beijing"},
				},
			},
			StopReason: "tool_use",
			Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 5},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get weather",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
	}
	resp, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("weather?")},
		core.WithTools(weatherTool))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "toolu_abc", resp.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Name)
	assert.Equal(t, `{"location":"Beijing"}`, resp.ToolCalls[0].Arguments)
	assert.Equal(t, "tool_calls", resp.FinishReason, "tool_use 应映射为 tool_calls")
}

// TestClient_ToolCallStream 验证流式工具调用解析（核心修复）：
// 1. content_block_start 携带 tool_use 的 id 和 name
// 2. input_json_delta 分片累加 partial_json
// 3. content_block_stop 发出完整 ToolCallDelta
// 4. message_delta 的 stop_reason="tool_use" 映射为 finish_reason="tool_calls"
func TestClient_ToolCallStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_456","name":"get_weather","input":{}}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"loc"}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"ation\":\"Beijing\"}"}}`,
			`data: {"type":"content_block_stop","index":0}`,
			`data: {"type":"message_delta","delta":{"type":"stop_reason","stop_reason":"tool_use"},"usage":{"input_tokens":10,"output_tokens":5}}`,
			`data: {"type":"message_stop"}`,
		}
		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get weather",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	}
	stream, err := client.ChatStream(context.Background(),
		[]core.Message{core.NewUserMessage("weather?")},
		core.WithTools(weatherTool))
	require.NoError(t, err)
	defer stream.Close()

	var sawToolCall bool
	var finishReason string
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case core.EventToolCall:
			sawToolCall = true
			require.Len(t, ev.ToolCallDeltas, 1)
			d := ev.ToolCallDeltas[0]
			assert.Equal(t, "toolu_456", d.ID)
			assert.Equal(t, "get_weather", d.Name)
			assert.Equal(t, `{"location":"Beijing"}`, d.Arguments)
		case core.EventDone:
			finishReason = ev.FinishReason
		}
	}
	require.NoError(t, stream.Err())

	assert.True(t, sawToolCall, "应收到 EventToolCall 事件")
	assert.Equal(t, "tool_calls", finishReason)

	tcs := stream.ToolCalls()
	require.Len(t, tcs, 1)
	assert.Equal(t, `{"location":"Beijing"}`, tcs[0].Arguments)
}

// TestClient_ToolResultPassthrough 验证工具结果回传：
// role=tool 消息转为 user 消息中的 tool_result block（Anthropic 格式）
func TestClient_ToolResultPassthrough(t *testing.T) {
	var gotReq anthropicRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = anthropicRequest{}
		json.NewDecoder(r.Body).Decode(&gotReq)
		json.NewEncoder(w).Encode(anthropicResponse{
			Content: []contentBlock{{Type: "text", Text: "ok"}},
			StopReason: "end_turn",
		})
	}))
	defer server.Close()

	client, err := NewAnthropic(core.Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("weather?"),
		{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{{ID: "toolu_789", Name: "get_weather", Arguments: `{"location":"Shanghai"}`}}},
		{Role: core.RoleTool, ToolCallID: "toolu_789", Content: []core.ContentBlock{{Type: core.ContentTypeText, Text: `{"temp":"25C"}`}}},
	}
	_, err = client.Chat(context.Background(), messages)
	require.NoError(t, err)

	// user → assistant(tool_use) → user(tool_result)
	require.GreaterOrEqual(t, len(gotReq.Messages), 3)
	// assistant 消息含 tool_use block（Content 是 interface{}，需重新序列化解析）
	assert.Equal(t, "assistant", gotReq.Messages[1].Role)
	blocksJSON, _ := json.Marshal(gotReq.Messages[1].Content)
	var blocks []contentBlock
	require.NoError(t, json.Unmarshal(blocksJSON, &blocks), "assistant content 应为 contentBlock 数组")
	require.Len(t, blocks, 1)
	assert.Equal(t, "tool_use", blocks[0].Type)
	assert.Equal(t, "toolu_789", blocks[0].ID)
	// tool 结果转为 user 消息含 tool_result block
	assert.Equal(t, "user", gotReq.Messages[2].Role)
	blocks2JSON, _ := json.Marshal(gotReq.Messages[2].Content)
	var blocks2 []contentBlock
	require.NoError(t, json.Unmarshal(blocks2JSON, &blocks2), "tool result content 应为 contentBlock 数组")
	require.Len(t, blocks2, 1)
	assert.Equal(t, "tool_result", blocks2[0].Type)
	assert.Equal(t, "toolu_789", blocks2[0].ToolUseID)
}

// TestClient_StopReasonMapping 验证 stop_reason 到 finish_reason 的映射
func TestClient_StopReasonMapping(t *testing.T) {
	tests := []struct {
		anthropic string
		openai    string
	}{
		{"end_turn", "stop"},
		{"stop_sequence", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
	}
	for _, tt := range tests {
		t.Run(tt.anthropic, func(t *testing.T) {
			got := mapStopReason(&streamDelta{StopReason: tt.anthropic})
			assert.Equal(t, tt.openai, got)
		})
	}
}
