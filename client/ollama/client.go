package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`     // base64-encoded images
	ToolCalls []toolCall `json:"tool_calls,omitempty"` // for assistant messages
	Thinking  string     `json:"thinking,omitempty"`   // qwen3 thinking content
	ID        string     `json:"id,omitempty"`         // message ID for tool calls
}

type toolCall struct {
	ID       string `json:"id,omitempty"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
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
	Format    string           `json:"format,omitempty"`    // json mode
	KeepAlive string           `json:"keep_alive,omitempty"` // duration to keep model in memory
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // max tokens
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
			}

			if chunk.Done {
				usage := &core.Usage{
					PromptTokens:     chunk.PromptEvalCount,
					CompletionTokens: chunk.EvalCount,
					TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
				}
				ch <- core.StreamEvent{
					Type:  core.EventDone,
					Usage: usage,
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
					Arguments: tc.Function.Arguments,
				})
			}
		}

		if chunk.Done {
			usage = core.Usage{
				PromptTokens:     chunk.PromptEvalCount,
				CompletionTokens: chunk.EvalCount,
				TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
			}
		}
	}

	return &core.Response{
		Model:            c.ResolveModel(options),
		Content:          content,
		ReasoningContent: reasoningContent,
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
			Role:    msg.Role,
			Content: msg.TextContent(),
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
				ollamaMsg.ToolCalls = append(ollamaMsg.ToolCalls, toolCall{
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Name,
						Arguments: tc.Arguments,
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
	if options.Temperature != nil || options.MaxTokens != nil {
		req.Options = &ollamaOptions{}
		if options.Temperature != nil {
			req.Options.Temperature = *options.Temperature
		} else {
			req.Options.Temperature = c.Config().Temperature
		}
		if options.MaxTokens != nil {
			req.Options.NumPredict = *options.MaxTokens
		} else if c.Config().MaxTokens > 0 {
			req.Options.NumPredict = c.Config().MaxTokens
		}
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
