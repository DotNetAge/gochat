package ollama

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

// Client is an Ollama LLM client.
//
// Ollama allows you to run large language models locally on your machine.
// This client connects to a local Ollama server (default: localhost:11434)
// and provides the same interface as cloud-based providers.
//
// No API key is required. The client uses a longer default timeout (60s)
// since local models may take more time to generate responses.
type Client struct {
	*core.BaseClient
}

// Ollama-specific wire format
type ollamaMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Images     []string   `json:"images,omitempty"`       // base64-encoded images
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`   // for assistant messages
	Thinking   string     `json:"thinking,omitempty"`     // thinking 模型的思考内容
	ToolCallID string     `json:"tool_call_id,omitempty"` // 用于 role=tool 的工具结果回传
}

// toolCall 表示 ollama 返回的工具调用。
// 注意：ollama 原生 API 的 function.arguments 是 JSON **对象**（如 {"location":"北京"}），
// 而 OpenAI 标准是 JSON **字符串**（如 "{\"location\":\"北京\"}"）。
// 这里用 json.RawMessage 接收原始对象，再在转换层序列化为字符串以对齐 core.ToolCall.Arguments。
type toolCall struct {
	ID       string `json:"id,omitempty"`
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type toolDefinition struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

type ollamaRequest struct {
	Model     string           `json:"model"`
	Messages  []ollamaMessage  `json:"messages"`
	Stream    bool             `json:"stream,omitempty"`
	Options   *ollamaOptions   `json:"options,omitempty"`
	Tools     []toolDefinition `json:"tools,omitempty"`
	Format    string           `json:"format,omitempty"`     // json mode
	KeepAlive string           `json:"keep_alive,omitempty"` // duration to keep model in memory
}

// ollamaOptions 对应 ollama 原生 /api/chat 的 options 字段。
// ollama 原生支持这些采样参数（与 OpenAI 参数语义对齐）：
//   - temperature / top_p / top_k / num_predict(max_tokens) / stop
//   - frequency_penalty / presence_penalty（ollama 新版已支持，与 OpenAI 同名）
//   - repeat_penalty（ollama 独有，近似 presence_penalty+frequency_penalty 的合并，
//     core.Options 无对应字段，保留供未来扩展）
//
// 用指针类型区分"未设置"与"零值"，与 core.Options 保持一致。
type ollamaOptions struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	NumPredict       *int     `json:"num_predict,omitempty"` // max tokens
	TopP             *float64 `json:"top_p,omitempty"`
	TopK             *int     `json:"top_k,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	RepeatPenalty    *float64 `json:"repeat_penalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
}

type ollamaResponse struct {
	Model           string         `json:"model"`
	CreatedAt       string         `json:"created_at"`
	Message         *ollamaMessage `json:"message,omitempty"`
	Done            bool           `json:"done"`
	DoneReason      string         `json:"done_reason,omitempty"`
	PromptEvalCount int            `json:"prompt_eval_count,omitempty"`
	EvalCount       int            `json:"eval_count,omitempty"`
}

// minOllamaTemperature 是透传给 ollama 的最小有效 temperature。
// ollama 在 temperature=0 时会无限循环卡死（ollama 已知问题），
// 用极小正值近似贪婪采样，保证上层显式设置 temperature=0（贪婪解码意图）
// 时 ollama 也能正常返回，行为与 OpenAI/DeepSeek 的 temperature=0 对齐。
const minOllamaTemperature = 0.001

// argumentsToString 把 ollama 返回的工具调用参数归一化为 OpenAI 标准的 JSON 字符串。
//
// ollama 原生 API 的 function.arguments 是 JSON **对象**
// （如 {"location":"北京"}），而 OpenAI 标准与 core.ToolCall.Arguments 期望
// 的是 JSON **字符串**（如 "{\"location\":\"北京\"}"）。
// 这里把原始对象直接转成字符串形式，正好满足 OpenAI 兼容契约；
// 若 ollama 某版本已返回字符串或返回 null/空，则相应处理。
func argumentsToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	s := strings.TrimSpace(string(raw))
	if s == "null" || s == "" {
		return ""
	}
	// 若已是 JSON 字符串字面量（以双引号开头），解包取原始文本
	if s[0] == '"' {
		var str string
		if err := json.Unmarshal(raw, &str); err == nil {
			return str
		}
	}
	return s
}

// New creates a new Ollama client
func NewOllamaClient(config core.Config) (*Client, error) {
	if config.Model == "" {
		config.Model = "qwen3.5:0.8b"
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}

	baseClient := core.NewClient(config)

	client := &Client{
		BaseClient: baseClient,
	}

	client.SetChatFunc(client.doChat)
	client.SetStreamFunc(client.doChatStream)

	return client, nil
}

// DefaultOllamaClient 创建默认的 Ollama 客户端
// 使用 qwen3.5:0.8b 模型，适用于本地智能分块和图提取
func DefaultOllamaClient() (*Client, error) {
	return NewOllamaClient(core.Config{
		Model:   "qwen3.5:0.8b",
		BaseURL: "http://localhost:11434",
	})
}

// doChatStream performs a streaming chat completion
func (c *Client) doChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	reqBody := c.buildRequest(messages, options, true)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/api/chat", c.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, core.NewNetworkError("failed to read error response", err)
		}
		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan core.StreamEvent, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		// seenToolCalls 标记本次流是否出现过工具调用，
		// 用于在 done 帧把 done_reason 由 "stop" 映射为 "tool_calls"。
		seenToolCalls := false

		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 1024*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				ch <- core.StreamEvent{Type: core.EventError, Err: ctx.Err()}
				return
			default:
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				continue
			}

			if chunk.Message != nil {
				if chunk.Message.Thinking != "" {
					ch <- core.StreamEvent{
						Type:    core.EventThinking,
						Content: chunk.Message.Thinking,
					}
				}

				if chunk.Message.Content != "" {
					ch <- core.StreamEvent{
						Type:    core.EventContent,
						Content: chunk.Message.Content,
					}
				}

				// 处理工具调用：ollama 原生流式会在某一帧的 message.tool_calls
				// 一次性给出完整的工具调用（不同于 OpenAI 的增量分片），
				// 这里转成 core.ToolCallDelta 发出 EventToolCall，
				// 由 core.Stream 累加并在 EventDone 时 finalize。
				if len(chunk.Message.ToolCalls) > 0 {
					seenToolCalls = true
					deltas := make([]core.ToolCallDelta, len(chunk.Message.ToolCalls))
					for i, tc := range chunk.Message.ToolCalls {
						deltas[i] = core.ToolCallDelta{
							Index:     i,
							ID:        tc.ID,
							Type:      "function",
							Name:      tc.Function.Name,
							Arguments: argumentsToString(tc.Function.Arguments),
						}
					}
					ch <- core.StreamEvent{
						Type:           core.EventToolCall,
						ToolCallDeltas: deltas,
					}
				}
			}

			if chunk.Done {
				usage := &core.Usage{
					PromptTokens:     chunk.PromptEvalCount,
					CompletionTokens: chunk.EvalCount,
					TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
				}
				// 语义对齐：ollama 即便触发工具调用，done_reason 通常仍是 "stop"，
				// 而 OpenAI 标准在工具调用时返回 "tool_calls"。
				// 这里在检测到工具调用时把 finish_reason 映射为 "tool_calls"，
				// 保证 goharness executor 的事件分发逻辑与 OpenAI/DeepSeek 一致。
				finishReason := chunk.DoneReason
				if seenToolCalls && (finishReason == "" || finishReason == "stop") {
					finishReason = "tool_calls"
				}
				ch <- core.StreamEvent{
					Type:         core.EventDone,
					FinishReason: finishReason,
					Usage:        usage,
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

// doChat performs the actual chat request
func (c *Client) doChat(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	reqBody := c.buildRequest(messages, options, stream)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/api/chat", c.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, core.NewNetworkError("failed to read error response", err)
		}
		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	// Ollama returns streaming format even for non-streaming requests
	// We need to accumulate all chunks
	var content string
	var reasoningContent string
	var usage core.Usage
	var toolCalls []core.ToolCall
	var finishReason string

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		var chunk ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}

		if chunk.Message != nil {
			content += chunk.Message.Content

			// qwen3 models use thinking field for reasoning
			reasoningContent += chunk.Message.Thinking

			// Extract tool calls from message
			for _, tc := range chunk.Message.ToolCalls {
				toolCalls = append(toolCalls, core.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: argumentsToString(tc.Function.Arguments),
				})
			}
		}

		if chunk.Done {
			usage = core.Usage{
				PromptTokens:     chunk.PromptEvalCount,
				CompletionTokens: chunk.EvalCount,
				TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
			}
			finishReason = chunk.DoneReason
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ollama scan error: %w", err)
	}

	// 语义对齐：ollama 触发工具调用时 done_reason 通常仍是 "stop"，
	// 而 OpenAI 标准在工具调用时返回 "tool_calls"。
	if len(toolCalls) > 0 && (finishReason == "" || finishReason == "stop") {
		finishReason = "tool_calls"
	}

	return &core.Response{
		Model:            c.ResolveModel(options),
		Content:          content,
		ReasoningContent: reasoningContent,
		FinishReason:     finishReason,
		Message: core.Message{
			Role: core.RoleAssistant,
			Content: []core.ContentBlock{
				{Type: core.ContentTypeText, Text: content},
			},
			ToolCalls: toolCalls,
		},
		ToolCalls: toolCalls,
		Usage:     &usage,
	}, nil
}

// buildRequest builds an Ollama API request
func (c *Client) buildRequest(messages []core.Message, options core.Options, stream bool) ollamaRequest {
	var ollamaMessages []ollamaMessage

	// Add system prompt if provided
	if options.SystemPrompt != "" {
		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    core.RoleSystem,
			Content: options.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range messages {
		ollamaMsg := ollamaMessage{
			Role:       msg.Role,
			Content:    msg.TextContent(),
			ToolCallID: msg.ToolCallID, // role=tool 消息回传工具调用 ID（对齐 OpenAI tool_call_id）
		}

		// Extract base64 images from content blocks
		for _, block := range msg.Content {
			if block.Type == core.ContentTypeImage && block.Data != "" {
				ollamaMsg.Images = append(ollamaMsg.Images, block.Data)
			}
		}

		// Add tool calls if present (for assistant messages)
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				// tc.Arguments 是 OpenAI 标准的 JSON 字符串（如 {"location":"北京"}），
				// ollama 原生 API 期望 arguments 为 JSON 对象。
				// json.RawMessage 直接透传原始 JSON 文本，序列化时即为合法 JSON 对象。
				var argsRaw json.RawMessage
				if tc.Arguments != "" {
					argsRaw = json.RawMessage(tc.Arguments)
				} else {
					argsRaw = json.RawMessage("{}")
				}
				ollamaMsg.ToolCalls = append(ollamaMsg.ToolCalls, toolCall{
					ID: tc.ID,
					Function: struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					}{
						Name:      tc.Name,
						Arguments: argsRaw,
					},
				})
			}
		}

		ollamaMessages = append(ollamaMessages, ollamaMsg)
	}

	req := ollamaRequest{
		Model:    c.ResolveModel(options),
		Messages: ollamaMessages,
		Stream:   stream,
	}

	// Add tools if provided
	if len(options.Tools) > 0 {
		for _, tool := range options.Tools {
			req.Tools = append(req.Tools, toolDefinition{
				Type: "function",
				Function: struct {
					Name        string          `json:"name"`
					Description string          `json:"description"`
					Parameters  json.RawMessage `json:"parameters"`
				}{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			})
		}
	}

	// Add options if any are set
	if opts := c.buildOllamaOptions(options); opts != nil {
		req.Options = opts
	}

	// Add format for JSON mode
	if options.Format != "" {
		req.Format = options.Format
	}

	// Add keep_alive for memory management
	if options.KeepAlive != "" {
		req.KeepAlive = options.KeepAlive
	}

	return req
}

// buildOllamaOptions 从 core.Options 与 client config 构建 ollama 原生 options。
// 优先级：显式传入的 options > client config 默认值 > 不设置（ollama 使用内置默认）。
// 返回 nil 表示无需设置 options 字段。
// 该方法把 OpenAI 风格的采样参数映射到 ollama 原生 options，保证三 provider 行为对齐：
//   - Temperature / TopP / TopK / Stop / PresencePenalty / FrequencyPenalty 直接对应
//   - MaxTokens → NumPredict
//   - Seed 透传（ollama 支持可复现采样）
func (c *Client) buildOllamaOptions(options core.Options) *ollamaOptions {
	opts := &ollamaOptions{}
	anySet := false

	// Temperature
	if options.Temperature != nil {
		t := *options.Temperature
		// ollama 在 temperature=0 时会无限循环卡死，用极小正值近似贪婪采样。
		if t == 0 {
			t = minOllamaTemperature
		}
		opts.Temperature = &t
		anySet = true
	} else if t := c.Config().Temperature; t != 0 {
		opts.Temperature = &t
		anySet = true
	}

	// NumPredict (max tokens)
	if options.MaxTokens != nil {
		opts.NumPredict = options.MaxTokens
		anySet = true
	} else if mt := c.Config().MaxTokens; mt > 0 {
		opts.NumPredict = &mt
		anySet = true
	}

	// TopP
	if options.TopP != nil {
		opts.TopP = options.TopP
		anySet = true
	}

	// TopK
	if options.TopK != nil {
		opts.TopK = options.TopK
		anySet = true
	}

	// Stop sequences
	if len(options.Stop) > 0 {
		opts.Stop = options.Stop
		anySet = true
	}

	// PresencePenalty
	if options.PresencePenalty != nil {
		opts.PresencePenalty = options.PresencePenalty
		anySet = true
	}

	// FrequencyPenalty
	if options.FrequencyPenalty != nil {
		opts.FrequencyPenalty = options.FrequencyPenalty
		anySet = true
	}

	if !anySet {
		return nil
	}
	return opts
}
