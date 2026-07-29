package ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// 测试文件结构说明
// ============================================================
// 本文件包含两组测试，参照 deepseek/client_test.go 的 mock 模式重构，
// 确保三个 provider（openai/deepseek/ollama）测试覆盖一致化：
//
// 1. Mock 测试组（TestMock_*）：不依赖真实 ollama 服务，用 httptest 模拟
//    ollama 原生 /api/chat 的 NDJSON 流式响应，精确验证 wire format 转换。
//    这是 CI 友好的主测试，覆盖所有对齐 OpenAI/DeepSeek 的改动点。
//
// 2. 真实集成测试组（TestLive_*）：对本地 ollama (localhost:11434) 发起真实
//    请求，验证端到端兼容性。默认 skip，设 OLLAMA_LIVE=1 环境变量时运行。
//
// 对齐改动点对应的测试：
//   - arguments object→string 转换     → TestMock_ChatToolCall / TestMock_ToolCallStream
//   - 流式 tool call 解析（核心修复）   → TestMock_ToolCallStream
//   - done_reason→finish_reason 映射   → TestMock_ToolCallStream / TestMock_FinishReasonMapping
//   - 采样参数透传                      → TestMock_Options
//   - role=tool 的 tool_call_id 透传    → TestMock_ToolCallIDPassthrough
//   - temperature=0 保护                → TestMock_Temperature0Guard
//   - thinking 字段解析                 → TestMock_ChatStreamThinking

// ============================================================
// Mock helpers
// ============================================================

// newMockClient 创建一个指向 httptest mock server 的 ollama client。
// server.URL 形如 http://127.0.0.1:xxxxx，作为 BaseURL 传给 client。
func newMockClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewOllamaClient(core.Config{
		BaseURL: server.URL,
		Model:   "test-model",
	})
	require.NoError(t, err)
	return client
}

// ndjsonLine 构造 ollama 原生流式 NDJSON 的一行（含末尾换行）。
// ollama /api/chat 流式响应是每行一个独立 JSON 对象（非 SSE）。
func ndjsonLine(msg *ollamaMessage, done bool, doneReason string, promptEval, eval int) string {
	resp := ollamaResponse{
		Model:           "test-model",
		CreatedAt:       "2026-07-29T12:00:00Z",
		Message:         msg,
		Done:            done,
		DoneReason:      doneReason,
		PromptEvalCount: promptEval,
		EvalCount:       eval,
	}
	b, _ := json.Marshal(resp)
	return string(b) + "\n"
}

// writeNDJSON 向 ResponseWriter 写入多行 NDJSON。
func writeNDJSON(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, l := range lines {
		io.WriteString(w, l)
	}
}

// decodeRequest 从请求体解析出 ollamaRequest，便于断言请求构造。
func decodeRequest(t *testing.T, r io.Reader) ollamaRequest {
	t.Helper()
	var req ollamaRequest
	require.NoError(t, json.NewDecoder(r).Decode(&req))
	return req
}

// ============================================================
// Mock 测试组
// ============================================================

// TestMock_Chat 验证基础非流式对话：请求路径为 /api/chat、模型名透传、
// content 累加、usage 统计、finish_reason 解析。
// ollama 非流式也返回 NDJSON 多 chunk，需累加。
func TestMock_Chat(t *testing.T) {
	var gotPath string
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		req := decodeRequest(t, r.Body)
		assert.Equal(t, "test-model", req.Model)
		assert.Equal(t, false, req.Stream) // Chat 非流式
		assert.Equal(t, "user", req.Messages[0].Role)
		writeNDJSON(w,
			ndjsonLine(&ollamaMessage{Role: "assistant", Content: "hello "}, false, "", 0, 0),
			ndjsonLine(&ollamaMessage{Role: "assistant", Content: "world"}, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 5, 2),
		)
	})

	resp, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	assert.Equal(t, "/api/chat", gotPath)
	assert.Equal(t, "hello world", resp.Content, "多 chunk content 应累加")
	assert.Equal(t, "assistant", resp.Message.Role)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 7, resp.Usage.TotalTokens, "usage 应为 prompt+completion")
}

// TestMock_ChatStream 验证流式对话：EventContent 事件正确发出、EventDone 带 usage。
func TestMock_ChatStream(t *testing.T) {
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r.Body)
		assert.Equal(t, true, req.Stream, "ChatStream 应 stream=true")
		writeNDJSON(w,
			ndjsonLine(&ollamaMessage{Role: "assistant", Content: "hel"}, false, "", 0, 0),
			ndjsonLine(&ollamaMessage{Role: "assistant", Content: "lo"}, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 3, 2),
		)
	})

	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	defer stream.Close()

	var result string
	var doneEvent *core.Usage
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case core.EventContent:
			result += ev.Content
		case core.EventDone:
			doneEvent = ev.Usage
		}
	}
	require.NoError(t, stream.Err())
	assert.Equal(t, "hello", result)
	require.NotNil(t, doneEvent)
	assert.Equal(t, 5, doneEvent.TotalTokens)
}

// TestMock_ChatStreamThinking 验证流式 thinking 字段解析为 EventThinking。
// 这是 ollama 原生 API 的 thinking 字段（非 OpenAI 的 reasoning_content），
// 必须走 OllamaClient 才能正确解析。
func TestMock_ChatStreamThinking(t *testing.T) {
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeNDJSON(w,
			ndjsonLine(&ollamaMessage{Role: "assistant", Thinking: "let me think"}, false, "", 0, 0),
			ndjsonLine(&ollamaMessage{Role: "assistant", Content: "answer"}, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 1, 1),
		)
	})

	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	defer stream.Close()

	var thinking, content string
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case core.EventThinking:
			thinking += ev.Content
		case core.EventContent:
			content += ev.Content
		}
	}
	require.NoError(t, stream.Err())
	assert.Equal(t, "let me think", thinking)
	assert.Equal(t, "answer", content)
}

// TestMock_ChatToolCall 验证非流式工具调用：
// 1. arguments 为 JSON object（ollama 原生格式）时能正确解析为字符串（OpenAI 标准）
// 2. tool_calls 正确提取到 Response.ToolCalls
// 这是核心修复点：修复前 toolCall.Arguments 定义为 string，ollama 返回 object 时 Unmarshal 静默失败。
func TestMock_ChatToolCall(t *testing.T) {
	// ollama 原生返回的 arguments 是 JSON object（不是字符串）
	argsJSON := json.RawMessage(`{"location":"Beijing"}`)
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		req := decodeRequest(t, r.Body)
		// 验证 tools 透传
		require.Len(t, req.Tools, 1)
		assert.Equal(t, "get_weather", req.Tools[0].Function.Name)

		// 构造带 tool_calls 的响应
		msg := &ollamaMessage{
			Role: "assistant",
			ToolCalls: []toolCall{{
				ID: "call_123",
				Function: struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}{
					Name:      "get_weather",
					Arguments: argsJSON,
				},
			}},
		}
		writeNDJSON(w,
			ndjsonLine(msg, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 10, 5),
		)
	})

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get weather",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
	}
	resp, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("weather in Beijing?")},
		core.WithTools(weatherTool))
	require.NoError(t, err)

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_123", resp.ToolCalls[0].ID)
	assert.Equal(t, "get_weather", resp.ToolCalls[0].Name)
	// 核心断言：object 已归一化为 JSON 字符串（OpenAI 标准）
	assert.Equal(t, `{"location":"Beijing"}`, resp.ToolCalls[0].Arguments)
	// 语义对齐：工具调用时 finish_reason 应为 tool_calls
	assert.Equal(t, "tool_calls", resp.FinishReason)
}

// TestMock_ToolCallStream 验证流式工具调用解析（核心修复）：
// 1. message.tool_calls 转为 EventToolCall 事件（修复前完全丢失）
// 2. arguments object→string 转换
// 3. done_reason="stop" → finish_reason="tool_calls" 语义映射
// 这是 goharness 实际使用的路径（GetStream → doChatStream）。
func TestMock_ToolCallStream(t *testing.T) {
	argsJSON := json.RawMessage(`{"location":"Beijing"}`)
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		msg := &ollamaMessage{
			Role: "assistant",
			ToolCalls: []toolCall{{
				ID: "call_456",
				Function: struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}{
					Name:      "get_weather",
					Arguments: argsJSON,
				},
			}},
		}
		writeNDJSON(w,
			ndjsonLine(msg, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 8, 3),
		)
	})

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

	var sawToolCallEvent bool
	var finishReason string
	for stream.Next() {
		ev := stream.Event()
		switch ev.Type {
		case core.EventToolCall:
			sawToolCallEvent = true
			require.Len(t, ev.ToolCallDeltas, 1)
			d := ev.ToolCallDeltas[0]
			assert.Equal(t, "call_456", d.ID)
			assert.Equal(t, "get_weather", d.Name)
			assert.Equal(t, `{"location":"Beijing"}`, d.Arguments)
		case core.EventDone:
			finishReason = ev.FinishReason
		}
	}
	require.NoError(t, stream.Err())

	// 核心断言：流式 tool call 事件被正确捕获（修复前完全丢失）
	assert.True(t, sawToolCallEvent, "应收到 EventToolCall 事件")
	// 语义对齐：工具调用时 finish_reason 应为 tool_calls（ollama 返回 stop，需映射）
	assert.Equal(t, "tool_calls", finishReason)

	// Stream.ToolCalls() 累加结果
	tcs := stream.ToolCalls()
	require.Len(t, tcs, 1)
	assert.Equal(t, `{"location":"Beijing"}`, tcs[0].Arguments)
}

// TestMock_FinishReasonMapping 验证 done_reason 到 finish_reason 的映射：
// - done_reason="stop" 且无工具调用 → finish_reason="stop"
// - done_reason="length" → finish_reason="length"（透传）
// - done_reason="" 且有工具调用 → finish_reason="tool_calls"
func TestMock_FinishReasonMapping(t *testing.T) {
	tests := []struct {
		name        string
		hasToolCall bool
		doneReason  string
		wantFinish  string
	}{
		{"纯文本 stop", false, "stop", "stop"},
		{"纯文本 length", false, "length", "length"},
		{"工具调用 stop→tool_calls", true, "stop", "tool_calls"},
		{"工具调用 空→tool_calls", true, "", "tool_calls"},
		{"工具调用 length 保留", true, "length", "length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
				var msg *ollamaMessage
				if tt.hasToolCall {
					msg = &ollamaMessage{
						Role: "assistant",
						ToolCalls: []toolCall{{
							Function: struct {
								Name      string          `json:"name"`
								Arguments json.RawMessage `json:"arguments"`
							}{Name: "f", Arguments: json.RawMessage(`{}`)},
						}},
					}
				} else {
					msg = &ollamaMessage{Role: "assistant", Content: "hi"}
				}
				writeNDJSON(w,
					ndjsonLine(msg, false, "", 0, 0),
					ndjsonLine(nil, true, tt.doneReason, 1, 1),
				)
			})
			stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("x")})
			require.NoError(t, err)
			defer stream.Close()
			var got string
			for stream.Next() {
				if ev := stream.Event(); ev.Type == core.EventDone {
					got = ev.FinishReason
				}
			}
			assert.Equal(t, tt.wantFinish, got)
		})
	}
}

// TestMock_Options 验证采样参数从 core.Options 透传到 ollama options 字段。
// 这是 ollama 对齐 OpenAI/DeepSeek 的关键：修复前只支持 temperature/num_predict。
func TestMock_Options(t *testing.T) {
	var gotReq ollamaRequest
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = decodeRequest(t, r.Body)
		writeNDJSON(w, ndjsonLine(nil, true, "stop", 1, 1))
	})

	temp := 0.7
	topP := 0.9
	topK := 40
	presence := 1.5
	frequency := 2.0
	_, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("hi")},
		core.WithTemperature(temp),
		core.WithMaxTokens(100),
		core.WithTopP(topP),
		core.WithTopK(topK),
		core.WithStop("END", "STOP"),
		core.WithPresencePenalty(presence),
		core.WithFrequencyPenalty(frequency),
	)
	require.NoError(t, err)

	require.NotNil(t, gotReq.Options)
	assert.Equal(t, temp, *gotReq.Options.Temperature)
	assert.Equal(t, 100, *gotReq.Options.NumPredict)
	assert.Equal(t, topP, *gotReq.Options.TopP)
	assert.Equal(t, topK, *gotReq.Options.TopK)
	assert.Equal(t, []string{"END", "STOP"}, gotReq.Options.Stop)
	assert.Equal(t, presence, *gotReq.Options.PresencePenalty)
	assert.Equal(t, frequency, *gotReq.Options.FrequencyPenalty)
}

// TestMock_OptionsNone 验证不设置任何 options 时，请求体不含 options 字段。
func TestMock_OptionsNone(t *testing.T) {
	var gotReq ollamaRequest
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = decodeRequest(t, r.Body)
		writeNDJSON(w, ndjsonLine(nil, true, "stop", 1, 1))
	})
	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	assert.Nil(t, gotReq.Options, "无 options 时不应发送 options 字段")
}

// TestMock_Temperature0Guard 验证 temperature=0 时用 minOllamaTemperature 保护。
// ollama 在 temperature=0 时会无限循环卡死（已知 bug），必须用极小正值近似贪婪采样。
func TestMock_Temperature0Guard(t *testing.T) {
	var gotReq ollamaRequest
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = decodeRequest(t, r.Body)
		writeNDJSON(w, ndjsonLine(nil, true, "stop", 1, 1))
	})
	_, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("hi")},
		core.WithTemperature(0.0),
	)
	require.NoError(t, err)
	require.NotNil(t, gotReq.Options)
	assert.Equal(t, minOllamaTemperature, *gotReq.Options.Temperature,
		"temperature=0 应映射为 minOllamaTemperature 避免 ollama 卡死")
}

// TestMock_ToolCallIDPassthrough 验证 role=tool 消息的 tool_call_id 透传。
// 这是工具结果回传的关键：OpenAI 标准用 tool_call_id 关联工具调用与结果。
func TestMock_ToolCallIDPassthrough(t *testing.T) {
	var gotReq ollamaRequest
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = decodeRequest(t, r.Body)
		writeNDJSON(w, ndjsonLine(nil, true, "stop", 1, 1))
	})

	messages := []core.Message{
		core.NewUserMessage("weather?"),
		{Role: core.RoleAssistant, ToolCalls: []core.ToolCall{{ID: "call_789", Name: "get_weather", Arguments: `{"location":"Shanghai"}`}}},
		{
			Role:       core.RoleTool,
			ToolCallID: "call_789",
			Content:    []core.ContentBlock{{Type: core.ContentTypeText, Text: `{"temp":"25C"}`}},
		},
	}
	_, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)

	// 验证三条消息正确转换
	require.Len(t, gotReq.Messages, 3)
	// assistant 消息带 tool_calls，arguments 字符串应透传为 JSON 对象
	assert.Equal(t, "assistant", gotReq.Messages[1].Role)
	require.Len(t, gotReq.Messages[1].ToolCalls, 1)
	assert.Equal(t, "call_789", gotReq.Messages[1].ToolCalls[0].ID)
	assert.Equal(t, "get_weather", gotReq.Messages[1].ToolCalls[0].Function.Name)
	assert.Equal(t, `{"location":"Shanghai"}`, string(gotReq.Messages[1].ToolCalls[0].Function.Arguments))
	// tool 消息带 tool_call_id
	assert.Equal(t, "tool", gotReq.Messages[2].Role)
	assert.Equal(t, "call_789", gotReq.Messages[2].ToolCallID, "tool_call_id 必须透传以关联工具结果")
}

// TestMock_ArgumentsStringFormat 验证 arguments 已是 JSON 字符串格式时的兼容处理。
// 某些 ollama 版本可能返回字符串形式的 arguments，需兼容。
func TestMock_ArgumentsStringFormat(t *testing.T) {
	// arguments 是 JSON 字符串字面量（带引号）
	argsJSON := json.RawMessage(`"{\"location\":\"Beijing\"}"`)
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		msg := &ollamaMessage{
			Role: "assistant",
			ToolCalls: []toolCall{{
				Function: struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				}{Name: "get_weather", Arguments: argsJSON},
			}},
		}
		writeNDJSON(w,
			ndjsonLine(msg, false, "", 0, 0),
			ndjsonLine(nil, true, "stop", 1, 1),
		)
	})
	resp, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	require.Len(t, resp.ToolCalls, 1)
	// 字符串字面量应解包为原始 JSON 文本
	assert.Equal(t, `{"location":"Beijing"}`, resp.ToolCalls[0].Arguments)
}

// TestMock_ErrorHandling 验证 HTTP 错误时返回 APIError。
func TestMock_ErrorHandling(t *testing.T) {
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})
	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.Error(t, err)
}

// TestMock_SystemPrompt 验证 system prompt 作为独立消息插入。
func TestMock_SystemPrompt(t *testing.T) {
	var gotReq ollamaRequest
	client := newMockClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotReq = decodeRequest(t, r.Body)
		writeNDJSON(w, ndjsonLine(nil, true, "stop", 1, 1))
	})
	_, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("hi")},
		core.WithSystemPrompt("you are helpful"),
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(gotReq.Messages), 2)
	assert.Equal(t, "system", gotReq.Messages[0].Role)
	assert.Equal(t, "you are helpful", gotReq.Messages[0].Content)
}

// ============================================================
// 真实集成测试组（需 OLLAMA_LIVE=1 环境变量）
// 对本地 ollama (localhost:11434) 发起真实请求，验证端到端兼容性。
// 默认 skip，避免 CI 依赖本地 ollama 运行。
// ============================================================

func skipUnlessLive(t *testing.T) {
	t.Helper()
	if os.Getenv("OLLAMA_LIVE") == "" {
		t.Skip("Skipping live test: set OLLAMA_LIVE=1 to run against local ollama (localhost:11434)")
	}
}

func newLiveClient(t *testing.T) *Client {
	t.Helper()
	client, err := NewOllamaClient(core.Config{BaseURL: "http://localhost:11434"})
	require.NoError(t, err)
	return client
}

func TestLive_Chat(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	resp, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("Say hello")})
	require.NoError(t, err)
	assert.True(t, resp.Content != "" || resp.ReasoningContent != "")
	assert.Equal(t, "assistant", resp.Message.Role)
}

func TestLive_ChatStream(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("Say hello")})
	require.NoError(t, err)
	defer stream.Close()
	var result string
	for stream.Next() {
		if ev := stream.Event(); ev.Type == core.EventContent {
			result += ev.Content
		}
	}
	require.NoError(t, stream.Err())
	assert.NotEmpty(t, result)
}

func TestLive_ChatStreamWithThinking(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	stream, err := client.ChatStream(context.Background(),
		[]core.Message{core.NewUserMessage("What is 1+1?")},
		core.WithMaxTokens(30))
	require.NoError(t, err)
	defer stream.Close()
	var hasEvent bool
	for stream.Next() {
		ev := stream.Event()
		if ev.Type == core.EventThinking || ev.Type == core.EventContent {
			hasEvent = true
		}
	}
	assert.True(t, hasEvent, "应收到 thinking 或 content 事件")
}

func TestLive_ChatWithOptions(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	resp, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("What is 1+1?")},
		core.WithTemperature(0.0),
		core.WithMaxTokens(500))
	require.NoError(t, err)
	assert.True(t, resp.Content != "" || resp.ReasoningContent != "")
}

func TestLive_ToolCalling(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string","description":"The city name"}},"required":["location"]}`),
	}
	resp, err := client.Chat(context.Background(),
		[]core.Message{core.NewUserMessage("What's the weather like in Beijing?")},
		core.WithTools(weatherTool))
	if err != nil {
		t.Skipf("Skipping: model may not support tool calling: %v", err)
	}
	if len(resp.ToolCalls) > 0 {
		assert.Equal(t, "get_weather", resp.ToolCalls[0].Name)
		assert.NotEmpty(t, resp.ToolCalls[0].Arguments)
	}
}

func TestLive_ToolCallingStream(t *testing.T) {
	skipUnlessLive(t)
	client := newLiveClient(t)
	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string","description":"The city name"}},"required":["location"]}`),
	}
	stream, err := client.ChatStream(context.Background(),
		[]core.Message{core.NewUserMessage("What's the weather like in Beijing?")},
		core.WithTools(weatherTool))
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer stream.Close()
	var sawToolCall bool
	for stream.Next() {
		if ev := stream.Event(); ev.Type == core.EventToolCall {
			sawToolCall = true
			break
		}
	}
	if !sawToolCall || len(stream.ToolCalls()) == 0 {
		t.Skip("Model did not request tool call in this run")
	}
	assert.Equal(t, "get_weather", stream.ToolCalls()[0].Name)
	assert.NotEmpty(t, stream.ToolCalls()[0].Arguments)
}
