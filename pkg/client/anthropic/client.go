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

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/core"
)

// Config defines configuration for the Anthropic (Claude) client.
//
// Embed base.Config to inherit common settings.
//
// Example:
//
//	config := anthropic.Config{
//	    Config: base.Config{
//	        APIKey: "sk-ant-...",
//	        Model:  "claude-3-opus-20240229",
//	    },
//	}
//	client, err := anthropic.New(config)
type Config struct {
	base.Config
}

// Client is an Anthropic (Claude) LLM client.
//
// It implements the core.Client interface and provides access to Anthropic's
// Claude models (Claude 3 Opus, Sonnet, Haiku, etc.).
//
// The client handles Anthropic-specific API requirements including the
// anthropic-version header, x-api-key authentication, and Claude's unique
// message format with separate system prompts.
type Client struct {
	base *base.Client
}

// Anthropic-specific wire format types
type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []contentBlock
}

type contentBlock struct {
	Type      string                 `json:"type"` // "text", "image", "tool_use", "tool_result"
	Text      string                 `json:"text,omitempty"`
	Source    *imageSource           `json:"source,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
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

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
	Thinking    *anthropicThinking `json:"thinking,omitempty"`
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
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
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
func New(config Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "claude-3-opus-20240229"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	baseClient := base.New(config.Config)

	return &Client{
		base: baseClient,
	}, nil
}

// Chat performs a non-streaming chat completion
func (c *Client) Chat(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Response, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	var response *core.Response
	err := c.base.Retry(ctx, func() error {
		resp, err := c.doChat(ctx, messages, options, false)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Call usage callback if provided
	if options.UsageCallback != nil && response.Usage != nil {
		options.UsageCallback(*response.Usage)
	}

	return response, nil
}

// ChatStream performs a streaming chat completion
func (c *Client) ChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	// Build request
	reqBody := c.buildRequest(messages, options, true)

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/messages", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	c.setHeaders(req)

	resp, err := c.base.HTTPClient().Do(req)
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

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
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
				continue
			}

			// Handle different chunk types
			switch chunk.Type {
			case "content_block_delta":
				if chunk.Delta != nil {
					if chunk.Delta.Type == "thinking_delta" && chunk.Delta.Thinking != "" {
						ch <- core.StreamEvent{
							Type:    core.EventThinking,
							Content: chunk.Delta.Thinking,
						}
					} else if chunk.Delta.Text != "" {
						ch <- core.StreamEvent{
							Type:    core.EventContent,
							Content: chunk.Delta.Text,
						}
					}
				}
			case "message_stop":
				ch <- core.StreamEvent{Type: core.EventDone}
			case "message_delta":
				if chunk.Usage != nil {
					ch <- core.StreamEvent{
						Type: core.EventDone,
						Usage: &core.Usage{
							PromptTokens:     chunk.Usage.InputTokens,
							CompletionTokens: chunk.Usage.OutputTokens,
							TotalTokens:      chunk.Usage.InputTokens + chunk.Usage.OutputTokens,
						},
					}
				}
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

	url := fmt.Sprintf("%s/v1/messages", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	c.setHeaders(req)

	resp, err := c.base.HTTPClient().Do(req)
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

	maxTokens := c.resolveMaxTokens(options)
	if maxTokens == 0 {
		maxTokens = 4096 // Anthropic requires max_tokens
	}

	req := anthropicRequest{
		Model:     c.resolveModel(options),
		Messages:  anthropicMessages,
		System:    systemPrompt,
		MaxTokens: maxTokens,
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
		// Temperature must be 1.0 or omitted for thinking models in some API versions, but we'll let it be.
	} else {
		req.Temperature = c.resolveTemperature(options)
	}

	return req
}

// convertMessage converts core.Message to Anthropic format
func (c *Client) convertMessage(msg core.Message) anthropicMessage {
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

// convertResponse converts Anthropic response to core.Response
func (c *Client) convertResponse(resp *anthropicResponse) *core.Response {
	var content string
	var contentBlocks []core.ContentBlock

	var reasoningContent string
	for _, block := range resp.Content {
		if block.Type == "thinking" {
			reasoningContent += block.Thinking
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentTypeThinking,
				Text: block.Thinking,
			})
		} else if block.Type == "text" {
			content += block.Text
			contentBlocks = append(contentBlocks, core.ContentBlock{
				Type: core.ContentTypeText,
				Text: block.Text,
			})
		}
	}

	return &core.Response{
		ID:               resp.ID,
		Model:            resp.Model,
		Content:          content,
		ReasoningContent: reasoningContent,
		FinishReason:     resp.StopReason,
		Message: core.Message{
			Role:    resp.Role,
			Content: contentBlocks,
		},
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

	if c.base.Config().AuthToken != "" {
		req.Header.Set("x-api-key", c.base.Config().AuthToken)
	} else if c.base.Config().APIKey != "" {
		token := core.GetAPIKey(c.base.Config().APIKey)
		req.Header.Set("x-api-key", token)
	}
}

// handleError handles error responses
func (c *Client) handleError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.NewNetworkError("failed to read error response", err)
	}

	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err == nil {
		return core.NewAPIError(fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
	}

	return core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
}

// Helper methods
func (c *Client) resolveModel(opts core.Options) string {
	if opts.Model != "" {
		return opts.Model
	}
	return c.base.Config().Model
}

func (c *Client) resolveTemperature(opts core.Options) float64 {
	if opts.Temperature != nil {
		return *opts.Temperature
	}
	return c.base.Config().Temperature
}

func (c *Client) resolveMaxTokens(opts core.Options) int {
	if opts.MaxTokens != nil {
		return *opts.MaxTokens
	}
	return c.base.Config().MaxTokens
}
