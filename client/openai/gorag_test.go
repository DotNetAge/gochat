package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/require"
)

// sseScanner 逐行读取 SSE 流，仅返回 data: 后的内容
type sseScanner struct {
	r   *bufio.Reader
	cur string
}

func newSSEScanner(r io.Reader) *sseScanner {
	return &sseScanner{r: bufio.NewReader(r)}
}

func (s *sseScanner) Next() bool {
	for {
		line, err := s.r.ReadString('\n')
		if err != nil && line == "" {
			return false
		}
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "data: ") {
			if err != nil && line == "" {
				return false
			}
			continue
		}
		s.cur = strings.TrimPrefix(line, "data: ")
		return true
	}
}

func (s *sseScanner) Data() string { return s.cur }

// goragEnv 读取 GORAG_* 环境变量，未设置时跳过测试
func goragEnv(t *testing.T) (baseURL, apiKey, model string) {
	t.Helper()
	baseURL = os.Getenv("GORAG_BASE_URL")
	apiKey = os.Getenv("GORAG_API_KEY")
	model = os.Getenv("GORAG_MODEL")
	if baseURL == "" || apiKey == "" || model == "" {
		t.Skip("跳过测试：GORAG_BASE_URL/GORAG_API_KEY/GORAG_MODEL 未设置")
	}
	return
}

// TestGorag_RawHTTP 通过原始 HTTP 请求查看模型返回的真实 JSON 结构，
// 用于确认 XML 标记的具体形态（出现在 content 字段还是单独字段）。
func TestGorag_RawHTTP(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "用一句话介绍你自己，并输出 1+1 的结果。"},
		},
		"temperature": 0.0,
		"stream":      false,
	}
	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(url, "/v1") {
		url = url + "/v1"
	}
	url = url + "/chat/completions"

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("HTTP 状态码: %d", resp.StatusCode)
	t.Logf("原始响应:\n%s", string(body))

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("请求失败: %s", string(body))
	}

	// 解析到通用结构中，单独打印 content 字段
	var parsed struct {
		Choices []struct {
			Message struct {
				Role             string          `json:"role"`
				Content          json.RawMessage `json:"content"`
				ReasoningContent string          `json:"reasoning_content,omitempty"`
				ToolCalls        json.RawMessage `json:"tool_calls,omitempty"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.NotEmpty(t, parsed.Choices)

	choice := parsed.Choices[0]
	t.Logf("finish_reason=%s", choice.FinishReason)
	t.Logf("role=%s", choice.Message.Role)
	t.Logf("content (raw)=%s", string(choice.Message.Content))
	if choice.Message.ReasoningContent != "" {
		t.Logf("reasoning_content=%s", choice.Message.ReasoningContent)
	}
	if len(choice.Message.ToolCalls) > 0 {
		t.Logf("tool_calls=%s", string(choice.Message.ToolCalls))
	}
}

// TestGorag_ClientChat 通过当前 OpenAI 客户端调用，观察 ResponseFromWire 解析后的结果。
func TestGorag_ClientChat(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("用一句话介绍你自己，并输出 1+1 的结果。"),
	}

	resp, err := client.Chat(context.Background(), messages, core.WithTemperature(0.0))
	require.NoError(t, err)

	t.Logf("ID=%s", resp.ID)
	t.Logf("Model=%s", resp.Model)
	t.Logf("FinishReason=%s", resp.FinishReason)
	t.Logf("Content (解析后)=\n%s", resp.Content)
	if resp.ReasoningContent != "" {
		t.Logf("ReasoningContent=\n%s", resp.ReasoningContent)
	}
	t.Logf("ToolCalls 数量=%d", len(resp.ToolCalls))
	if resp.Usage != nil {
		t.Logf("Usage: prompt=%d completion=%d total=%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestGorag_ClientChatStream 通过流式调用观察分片内容，确认 XML 标记是否跨分片到达。
func TestGorag_ClientChatStream(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("用一句话介绍你自己，并输出 1+1 的结果。"),
	}

	stream, err := client.ChatStream(context.Background(), messages, core.WithTemperature(0.0))
	require.NoError(t, err)
	defer stream.Close()

	var contentBuf, thinkingBuf strings.Builder
	chunkCount := 0
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatalf("流错误: %v", ev.Err)
		}
		chunkCount++
		switch ev.Type {
		case core.EventContent:
			contentBuf.WriteString(ev.Content)
		case core.EventThinking:
			thinkingBuf.WriteString(ev.Content)
		case core.EventToolCall:
			t.Logf("[chunk#%d] ToolCall delta: %+v", chunkCount, ev.ToolCallDeltas)
		case core.EventDone:
			t.Logf("[chunk#%d] Done, finish=%s", chunkCount, ev.FinishReason)
		}
	}
	require.NoError(t, stream.Err())

	t.Logf("共收到 %d 个 chunk", chunkCount)
	t.Logf("流式 Content=\n%s", contentBuf.String())
	if thinkingBuf.Len() > 0 {
		t.Logf("流式 Thinking=\n%s", thinkingBuf.String())
	}
}

// TestGorag_ToolCalling 测试带工具调用的场景，确认 XML 标记是否影响工具调用解析。
func TestGorag_ToolCalling(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "获取指定城市的天气",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {"type": "string", "description": "城市名"}
			},
			"required": ["location"]
		}`),
	}

	messages := []core.Message{
		core.NewUserMessage("北京今天天气怎么样？请调用工具查询。"),
	}

	resp, err := client.Chat(context.Background(), messages,
		core.WithTools(weatherTool),
		core.WithTemperature(0.0),
	)
	require.NoError(t, err)

	t.Logf("FinishReason=%s", resp.FinishReason)
	t.Logf("Content=\n%s", resp.Content)
	t.Logf("ToolCalls 数量=%d", len(resp.ToolCalls))
	for i, tc := range resp.ToolCalls {
		t.Logf("ToolCall#%d: id=%s name=%s args=%s", i, tc.ID, tc.Name, tc.Arguments)
	}

	// 把 content 完整 dump 出来，便于检查是否混入了 XML
	if resp.Content != "" {
		fmt.Printf("\n========== 完整 Content ==========\n%s\n==================================\n", resp.Content)
	}
}

// TestGorag_WithThinking 开启思考模式，GLM-4 系列常把 <think>...</think> 写进 content。
func TestGorag_WithThinking(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("请仔细推理：一个池塘里有荷叶，每天数量翻倍，30 天长满，问第几天长满一半？"),
	}

	// 非流式 + thinking
	resp, err := client.Chat(context.Background(), messages,
		core.WithThinking(0),
		core.WithTemperature(0.0),
	)
	require.NoError(t, err)

	t.Logf("[非流式] FinishReason=%s", resp.FinishReason)
	t.Logf("[非流式] Content=\n%s", resp.Content)
	t.Logf("[非流式] ReasoningContent=\n%s", resp.ReasoningContent)
	fmt.Printf("\n========== 非流式 Content ==========\n%s\n", resp.Content)
	if resp.ReasoningContent != "" {
		fmt.Printf("========== 非流式 ReasoningContent ==========\n%s\n", resp.ReasoningContent)
	}

	// 流式 + thinking
	stream, err := client.ChatStream(context.Background(), messages,
		core.WithThinking(0),
		core.WithTemperature(0.0),
	)
	require.NoError(t, err)
	defer stream.Close()

	var contentBuf, thinkingBuf strings.Builder
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatalf("流错误: %v", ev.Err)
		}
		switch ev.Type {
		case core.EventContent:
			contentBuf.WriteString(ev.Content)
		case core.EventThinking:
			thinkingBuf.WriteString(ev.Content)
		}
	}
	require.NoError(t, stream.Err())

	t.Logf("[流式] Content=\n%s", contentBuf.String())
	t.Logf("[流式] Thinking=\n%s", thinkingBuf.String())
	fmt.Printf("\n========== 流式 Content ==========\n%s\n", contentBuf.String())
	fmt.Printf("\n========== 流式 Thinking(独立事件) ==========\n%s\n", thinkingBuf.String())
}

// TestGorag_RawWithThinking 原始 HTTP 请求 + enable_thinking，查看 content 是否含 <think>。
func TestGorag_RawWithThinking(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "请仔细推理：一个池塘里有荷叶，每天数量翻倍，30 天长满，问第几天长满一半？"},
		},
		"temperature": 0.0,
		"stream":      false,
		"extra_body": map[string]interface{}{
			"enable_thinking": true,
		},
	}
	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(url, "/v1") {
		url = url + "/v1"
	}
	url = url + "/chat/completions"

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("HTTP 状态码: %d", resp.StatusCode)
	t.Logf("原始响应:\n%s", string(body))
	fmt.Printf("\n========== 原始响应 ==========\n%s\n", string(body))
}

// TestGorag_RawStream 原始流式请求，逐行打印 SSE，确认 <think> 是否出现在 content delta 中。
func TestGorag_RawStream(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "请仔细推理：一个池塘里有荷叶，每天数量翻倍，30 天长满，问第几天长满一半？"},
		},
		"stream":      true,
		"stream_options": map[string]interface{}{"include_usage": true},
	}
	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := strings.TrimSuffix(baseURL, "/")
	if !strings.HasSuffix(url, "/v1") {
		url = url + "/v1"
	}
	url = url + "/chat/completions"

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	scanner := newSSEScanner(resp.Body)
	var fullContent, fullReasoning strings.Builder
	lineNo := 0
	for scanner.Next() {
		lineNo++
		data := scanner.Data()
		if data == "[DONE]" {
			t.Logf("[line#%d] [DONE]", lineNo)
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Role             string `json:"role"`
					Content          string `json:"content"`
					ReasoningContent string `json:"reasoning_content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			t.Logf("[line#%d] 解析失败: %v, raw=%s", lineNo, err, data)
			continue
		}
		if len(chunk.Choices) == 0 {
			t.Logf("[line#%d] 无 choices, raw=%s", lineNo, data)
			continue
		}
		c := chunk.Choices[0]
		if c.Delta.Content != "" {
			fullContent.WriteString(c.Delta.Content)
		}
		if c.Delta.ReasoningContent != "" {
			fullReasoning.WriteString(c.Delta.ReasoningContent)
		}
		if c.FinishReason != "" {
			t.Logf("[line#%d] finish_reason=%s", lineNo, c.FinishReason)
		}
	}

	fmt.Printf("\n========== 流式拼接 Content (%d 字符) ==========\n%s\n", fullContent.Len(), fullContent.String())
	fmt.Printf("\n========== 流式拼接 ReasoningContent (%d 字符) ==========\n%s\n", fullReasoning.Len(), fullReasoning.String())
	t.Logf("content 含 <think>=%v, 含 </think>=%v",
		strings.Contains(fullContent.String(), "<think>"),
		strings.Contains(fullContent.String(), "</think>"))
	t.Logf("content 含 <tool_call>=%v", strings.Contains(fullContent.String(), "<tool_call>"))
}

// TestGorag_TopLevelEnableThinking SiliconFlow 可能要求 enable_thinking 在顶层而非 extra_body。
func TestGorag_TopLevelEnableThinking(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	// 顶层 enable_thinking
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "请仔细推理：一个池塘里有荷叶，每天数量翻倍，30 天长满，问第几天长满一半？"},
		},
		"stream":          false,
		"enable_thinking": true,
	}
	jsonData, err := json.Marshal(reqBody)
	require.NoError(t, err)

	url := strings.TrimSuffix(baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Logf("[顶层 enable_thinking=true] HTTP %d:\n%s", resp.StatusCode, string(body))

	// 解析 content 看是否含 <think>
	var parsed struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Choices) > 0 {
		c := parsed.Choices[0].Message
		fmt.Printf("\n========== 顶层 enable_thinking Content ==========\n%s\n", c.Content)
		fmt.Printf("\n========== 顶层 enable_thinking ReasoningContent ==========\n%s\n", c.ReasoningContent)
		t.Logf("content 含 <think>=%v", strings.Contains(c.Content, "<think>"))
	}
}

// TestGorag_NaturalLanguageTool 用自然语言在 system prompt 中描述工具，
// 观察模型是否以 XML 形式（<tool_call>...</tool_call>）输出工具调用。
func TestGorag_NaturalLanguageTool(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	systemPrompt := `你可以使用以下工具。需要调用工具时，请输出：
<tool_call>
{"name": "工具名", "arguments": {参数}}
</tool_call>

可用工具：
- get_weather(location: string): 获取指定城市的天气`

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("北京今天天气怎么样？"),
	}

	resp, err := client.Chat(context.Background(), messages,
		core.WithSystemPrompt(systemPrompt),
		core.WithTemperature(0.0),
	)
	require.NoError(t, err)

	t.Logf("FinishReason=%s", resp.FinishReason)
	t.Logf("Content=\n%s", resp.Content)
	fmt.Printf("\n========== 自然语言工具 Content ==========\n%s\n", resp.Content)
	t.Logf("content 含 <tool_call>=%v", strings.Contains(resp.Content, "<tool_call>"))
	t.Logf("ToolCalls 数量=%d (应为0，因为未用 API tools)", len(resp.ToolCalls))
}

// TestGorag_JSONExtraction 模拟 gorag Refiller 的实际场景：要求模型输出严格 JSON。
// 这是最可能触发 <think>...</think> 前缀（从而破坏 JSON 解析）的场景。
func TestGorag_JSONExtraction(t *testing.T) {
	baseURL, apiKey, model := goragEnv(t)

	systemPrompt := `你是一名精准的实体关系提取助手。
给定一个 JSON 数组形式的文本分块，请从中提取实体和关系。
输出严格的 JSON，仅包含两个字段：
- "entities": [{"name": string, "entity_type": string, "properties": {}}]
- "relations": [{"subject": string, "predicate": string, "object": string}]

规则：
- JSON 输出必须使用英文标点，禁止出现中文引号、中文逗号或中文冒号。
- 如果未找到实体或关系，返回 {"entities":[],"relations":[]}。`

	userContent := `[{"id":"c1","title":"苹果公司","summary":"苹果公司简介","content":"苹果公司（Apple Inc.）是一家美国科技公司，总部位于加州库比蒂诺，由史蒂夫·乔布斯创立。主要产品包括 iPhone、Mac 和 iPad。"}]`

	client, err := NewOpenAI(core.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewSystemMessage(systemPrompt),
		core.NewUserMessage("请从以下分块中提取实体和关系：\n\n" + userContent),
	}

	// 非流式
	resp, err := client.Chat(context.Background(), messages, core.WithTemperature(0.0))
	require.NoError(t, err)

	t.Logf("[非流式] FinishReason=%s", resp.FinishReason)
	t.Logf("[非流式] Content=\n%s", resp.Content)
	fmt.Printf("\n========== 非流式 JSON 提取 Content ==========\n%s\n", resp.Content)
	t.Logf("[非流式] content 含 <think>=%v", strings.Contains(resp.Content, "<think>"))
	t.Logf("[非流式] content 含 <tool_call>=%v", strings.Contains(resp.Content, "<tool_call>"))
	t.Logf("[非流式] content 含 ```json=%v", strings.Contains(resp.Content, "```json"))

	// 流式
	stream, err := client.ChatStream(context.Background(), messages, core.WithTemperature(0.0))
	require.NoError(t, err)
	defer stream.Close()

	var sb strings.Builder
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatalf("流错误: %v", ev.Err)
		}
		if ev.Type == core.EventContent {
			sb.WriteString(ev.Content)
		}
	}
	require.NoError(t, stream.Err())
	t.Logf("[流式] Content=\n%s", sb.String())
	fmt.Printf("\n========== 流式 JSON 提取 Content ==========\n%s\n", sb.String())
	t.Logf("[流式] content 含 <think>=%v", strings.Contains(sb.String(), "<think>"))
}
