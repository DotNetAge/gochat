package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/DotNetAge/gochat/core"
)

// Client is an Anthropic (Claude) LLM client.
//
// It implements the core.Client interface and provides access to Anthropic's
// Claude models (Claude 3 Opus, Sonnet, Haiku, etc.).
//
// The client handles Anthropic-specific API requirements including the
// anthropic-version header, x-api-key authentication, and Claude's unique
// message format with separate system prompts.
//
// Tool calling is fully supported in both streaming and non-streaming modes,
// aligned with OpenAI/DeepSeek/Ollama provider behavior:
//   - Tools are sent via the tools field (input_schema, not parameters)
//   - Tool calls arrive as tool_use content blocks
//   - Tool results are sent back as tool_result content blocks
//   - Streaming uses content_block_start/delta/stop events
//   - stop_reason "tool_use" is mapped to "tool_calls" for cross-provider parity
type Client struct {
	*core.BaseClient
}

// ---- Anthropic wire format types ----

type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []contentBlock
}

type contentBlock struct {
	Type      string                 `json:"type"` // "text", "image", "tool_use", "tool_result", "thinking"
	Text      string                 `json:"text,omitempty"`
	Source    *imageSource           `json:"source,omitempty"`
	ID        string                 `json:"id,omitempty"`         // for tool_use
	Name      string                 `json:"name,omitempty"`       // for tool_use
	Input     map[string]interface{} `json:"input,omitempty"`      // for tool_use
	ToolUseID string                 `json:"tool_use_id,omitempty"` // for tool_result
	Thinking  string                 `json:"thinking,omitempty"`
	Signature string                 `json:"signature,omitempty"`
}

type imageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// anthropicTool 对应 Anthropic API 的 tools 字段中的工具定义。
// Anthropic 用 input_schema（而非 OpenAI 的 parameters）定义参数 schema。
type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Thinking    *anthropicThinking `json:"thinking,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
}

type anthropicResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []contentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      anthropicUsage `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type streamChunk struct {
	Type         string          `json:"type"`
	Index        int             `json:"index,omitempty"`
	Delta        *streamDelta    `json:"delta,omitempty"`
	ContentBlock *contentBlock   `json:"content_block,omitempty"`
	Usage        *anthropicUsage `json:"usage,omitempty"`
}

type streamDelta struct {
	Type        string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta", "stop_reason"
	Text        string `json:"text,omitempty"`
	Thinking    string `json:"thinking,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"` // for input_json_delta
}

type errorResponse struct {
	Type  string      `json:"type"`
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// New creates a new Anthropic client
func NewAnthropic(config core.Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "claude-3-opus-20240229"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	baseClient := core.NewClient(config)

	client := &Client{
		BaseClient: baseClient,
	}

	client.SetChatFunc(client.doChat)
	client.SetStreamFunc(client.doChatStream)

	return client, nil
}

// doChatStream performs a streaming chat completion
func (c *Client) doChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	// Build request
	reqBody := c.buildRequest(messages, options, true)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/messages", c.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.handleError(resp)
	}

	ch := make(chan core.StreamEvent, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// ctx 取消监听：当 ctx 被取消或超时时，主动关闭 resp.Body，
		// 打断下方 scanner.Scan() 的阻塞读取。
		// 这是修复流式卡死的关键：真实 LLM server 不一定监听客户端 ctx 取消，
		// 不会主动关闭连接，导致 scanner.Scan() 永久阻塞。
		// 关闭 resp.Body 会让 scanner.Scan() 立即返回错误并退出循环。
		go func() {
			<-ctx.Done()
			_ = resp.Body.Close()
		}()

		// toolCallAcc 累加流式 tool_use 的 partial_json 片段。
		// Anthropic 的工具参数通过 input_json_delta 分片到达，
		// 累加后在 content_block_stop 时作为一个完整 ToolCallDelta 发出。
		toolCallAcc := map[int]*streamToolCallState{}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			// ctx 已取消（body 被关闭后 scanner 可能返回最后一行或错误）
			select {
			case <-ctx.Done():
				ch <- core.StreamEvent{Type: core.EventError, Err: ctx.Err()}
				return
			default:
			}

			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk streamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- core.StreamEvent{Type: core.EventError, Err: fmt.Errorf("failed to parse stream chunk: %w", err)}
				return
			}

			// 处理不同的事件类型
			switch chunk.Type {
			case "content_block_start":
				// 新 content block 开始。若是 tool_use，记录 id 和 name。
				if chunk.ContentBlock != nil && chunk.ContentBlock.Type == "tool_use" {
					toolCallAcc[chunk.Index] = &streamToolCallState{
						id:   chunk.ContentBlock.ID,
						name: chunk.ContentBlock.Name,
					}
				}
			case "content_block_delta":
				if chunk.Delta != nil {
					switch chunk.Delta.Type {
					case "thinking_delta":
						if chunk.Delta.Thinking != "" {
							ch <- core.StreamEvent{
								Type:    core.EventThinking,
								Content: chunk.Delta.Thinking,
							}
						}
					case "text_delta":
						if chunk.Delta.Text != "" {
							ch <- core.StreamEvent{
								Type:    core.EventContent,
								Content: chunk.Delta.Text,
							}
						}
					case "input_json_delta":
						// 工具参数分片到达，累加 partial_json
						if chunk.Delta.PartialJSON != "" {
							if state, ok := toolCallAcc[chunk.Index]; ok {
								state.arguments.WriteString(chunk.Delta.PartialJSON)
							}
						}
					}
				}
			case "content_block_stop":
				// content block 结束。若是 tool_use，发出完整的 ToolCallDelta。
				if state, ok := toolCallAcc[chunk.Index]; ok {
					ch <- core.StreamEvent{
						Type: core.EventToolCall,
						ToolCallDeltas: []core.ToolCallDelta{
							{
								Index:     chunk.Index,
								ID:        state.id,
								Type:      "function",
								Name:      state.name,
								Arguments: state.arguments.String(),
							},
						},
					}
					delete(toolCallAcc, chunk.Index)
				}
			case "message_delta":
				// 顶层消息变化，通常携带 stop_reason 和 usage
				finishReason := mapStopReason(chunk.Delta)
				ev := core.StreamEvent{
					Type:         core.EventDone,
					FinishReason: finishReason,
				}
				if chunk.Usage != nil {
					ev.Usage = &core.Usage{
						PromptTokens:     chunk.Usage.InputTokens,
						CompletionTokens: chunk.Usage.OutputTokens,
						TotalTokens:      chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
					}
				}
				ch <- ev
			case "message_stop":
				// message_delta 已发送 EventDone
			case "error":
				ch <- core.StreamEvent{
					Type: core.EventError,
					Err:  fmt.Errorf("stream error: %s", data),
				}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			ch <- core.StreamEvent{Type: core.EventError, Err: err}
		}
	}()

	return core.NewStream(ch, resp.Body), nil
}

// streamToolCallState 累加流式 tool_use 的 partial_json 片段
type streamToolCallState struct {
	id        string
	name      string
	arguments strings.Builder
}

// doChat performs the actual chat request
func (c *Client) doChat(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	reqBody := c.buildRequest(messages, options, stream)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/messages", c.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	c.setHeaders(req)

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, core.NewNetworkError("failed to read response", err)
	}

	var respData anthropicResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, core.NewValidationError("failed to unmarshal response", err)
	}

	return c.convertResponse(&respData), nil
}

// buildRequest builds an Anthropic API request
func (c *Client) buildRequest(messages []core.Message, options core.Options, stream bool) anthropicRequest {
	var anthropicMessages []anthropicMessage
	var systemPrompt string

	// Extract system messages
	for _, msg := range messages {
		if msg.Role == core.RoleSystem {
			systemPrompt = msg.TextContent()
		} else {
			anthropicMessages = append(anthropicMessages, c.convertMessage(msg))
		}
	}

	// Add system prompt from options
	if options.SystemPrompt != "" {
		if systemPrompt != "" {
			systemPrompt = options.SystemPrompt + "\n\n" + systemPrompt
		} else {
			systemPrompt = options.SystemPrompt
		}
	}

	req := anthropicRequest{
		Model:     c.ResolveModel(options),
		Messages:  anthropicMessages,
		System:    systemPrompt,
		MaxTokens: 4096, // Anthropic requires max_tokens
		Stream:    stream,
	}

	if options.Thinking {
		budget := options.ThinkingBudget
		if budget < 1024 {
			budget = 1024 // minimum required
		}
		if req.MaxTokens <= budget {
			req.MaxTokens = budget + 1024
		}
		req.Thinking = &anthropicThinking{
			Type:         "enabled",
			BudgetTokens: budget,
		}
	} else {
		req.Temperature = c.ResolveTemperature(options)
	}

	// 处理工具定义（Anthropic 用 input_schema 而非 parameters）
	if len(options.Tools) > 0 {
		req.Tools = make([]anthropicTool, len(options.Tools))
		for i, t := range options.Tools {
			req.Tools[i] = anthropicTool{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: t.Parameters,
			}
		}
	}

	return req
}

// convertMessage converts core.Message to Anthropic format.
// 处理三种角色：
//   - user/assistant 的文本和图片内容
//   - assistant 的 tool_calls → tool_use content block
//   - tool 的工具结果 → tool_result content block（Anthropic 要求 tool_result
//     必须放在 user 消息的 content 数组里，而非独立的 role=tool 消息）
func (c *Client) convertMessage(msg core.Message) anthropicMessage {
	// role=tool 的消息需转为 user 消息中的 tool_result block
	if msg.Role == core.RoleTool {
		return anthropicMessage{
			Role: core.RoleUser,
			Content: []contentBlock{
				{
					Type:      "tool_result",
					ToolUseID: msg.ToolCallID,
					Text:      msg.TextContent(),
				},
			},
		}
	}

	// 若 assistant 消息带 tool_calls，转为 tool_use content blocks
	if len(msg.ToolCalls) > 0 {
		var blocks []contentBlock
		// 先输出已有的文本内容（如有）
		if text := msg.TextContent(); text != "" {
			blocks = append(blocks, contentBlock{Type: "text", Text: text})
		}
		for _, tc := range msg.ToolCalls {
			var input map[string]interface{}
			// core.ToolCall.Arguments 是 JSON 字符串，解析为 map
			if tc.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Arguments), &input)
			}
			blocks = append(blocks, contentBlock{
				Type:  "tool_use",
				ID:    tc.ID,
				Name:  tc.Name,
				Input: input,
			})
		}
		return anthropicMessage{
			Role:    msg.Role,
			Content: blocks,
		}
	}

	if len(msg.Content) == 1 && msg.Content[0].Type == core.ContentTypeText {
		return anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content[0].Text,
		}
	}

	// Multimodal content
	var blocks []contentBlock
	for _, block := range msg.Content {
		switch block.Type {
		case core.ContentTypeText:
			blocks = append(blocks, contentBlock{
				Type: "text",
				Text: block.Text,
			})
		case core.ContentTypeImage:
			blocks = append(blocks, contentBlock{
				Type: "image",
				Source: &imageSource{
					Type:      "base64",
					MediaType: block.MediaType,
					Data:      block.Data,
				},
			})
		}
	}

	return anthropicMessage{
		Role:    msg.Role,
		Content: blocks,
	}
}

// mapStopReason 将 Anthropic 的 stop_reason 映射为 OpenAI 标准的 finish_reason，
// 保证三个 provider 的语义一致：
//   - "end_turn" / "stop_sequence" → "stop"
//   - "max_tokens" → "length"
//   - "tool_use" → "tool_calls"
//   - 其它 → 原样透传
func mapStopReason(delta *streamDelta) string {
	if delta == nil {
		return ""
	}
	reason := delta.StopReason
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

// convertResponse converts Anthropic response to core.Response.
// 提取 text 和 tool_use content blocks，tool_use 转为 core.ToolCall。
func (c *Client) convertResponse(resp *anthropicResponse) *core.Response {
	var content string
	var contentBlocks []core.ContentBlock
	var reasoningContent string
	var toolCalls []core.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "thinking":
			reasoningContent += block.Thinking
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentTypeThinking,
				Text: block.Thinking,
			})
		case "text":
			content += block.Text
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentTypeText,
				Text: block.Text,
			})
		case "tool_use":
			// 将 input map 序列化为 JSON 字符串（OpenAI 标准）
			argsBytes, _ := json.Marshal(block.Input)
			tc := core.ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(argsBytes),
			}
			toolCalls = append(toolCalls, tc)
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentTypeText,
				Text: fmt.Sprintf("[tool_use: %s(%s)]", block.Name, string(argsBytes)),
			})
		}
	}

	// 映射 stop_reason 到 OpenAI finish_reason
	finishReason := mapStopReason(&streamDelta{StopReason: resp.StopReason})

	return &core.Response{
		ID:               resp.ID,
		Model:            resp.Model,
		Content:          content,
		ReasoningContent: reasoningContent,
		FinishReason:     finishReason,
		Message: core.Message{
			Role:      resp.Role,
			Content:   contentBlocks,
			ToolCalls: toolCalls,
		},
		ToolCalls: toolCalls,
		Usage: &core.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// setHeaders sets Anthropic-specific headers
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	if c.Config().AuthToken != "" {
		req.Header.Set("x-api-key", c.Config().AuthToken)
	} else if c.Config().APIKey != "" {
		token := core.GetAPIKey(c.Config().APIKey)
		req.Header.Set("x-api-key", token)
	}
}

// handleError handles error responses.
// 使用 NewAPIErrorFromResponse 以获得基于状态码的错误类型分类（429→RateLimit 等），
// 保持与 OpenAI/DeepSeek/Ollama provider 一致的重试判定。
func (c *Client) handleError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewNetworkError("failed to read error response", err)
	}

	// 尝试解析 Anthropic 错误格式以获取更友好的消息
	var errResp errorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		return &core.Error{
			Type:       core.ErrorTypeAPI,
			Message:    fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message),
			StatusCode: resp.StatusCode,
		}
	}

	return core.NewAPIErrorFromResponse(resp.StatusCode, body)
}
