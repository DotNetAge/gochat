package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/DotNetAge/gochat/core"
)

// Client is an OpenAI LLM client.
//
// It implements the core.Client interface and provides access to OpenAI's
// chat completion API, including GPT-3.5, GPT-4, and other models.
//
// The client handles authentication, retries, error handling, and streaming
// automatically. It supports all OpenAI features including tool calling,
// multimodal inputs, and extended thinking (for o1/o3 models).
//
// IMPORTANT FOR ALIBABA CLOUD QWEN USERS:
// This client uses OpenAI compatibility mode by default (dashscope.aliyuncs.com/compatible-mode/v1).
// Compatibility mode has limitations:
// - Does NOT support enable_search (internet search)
// - Does NOT support enable_thinking (deep thinking mode)
// - Does NOT support incremental_output
// - Only supports standard OpenAI parameters
//
// For full Qwen feature support, use client/dashscope instead.
type Client struct {
	*core.BaseClient
}

// New creates a new OpenAI client
func NewOpenAI(config core.Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "qwen3.5-flash"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	baseClient := core.NewClient(config)

	client := &Client{
		BaseClient: baseClient,
	}

	// Inject chat and stream functions
	client.SetChatFunc(client.doChat)
	client.SetStreamFunc(client.doChatStream)

	return client, nil
}

// doChatStream performs the actual streaming chat request
func (c *Client) doChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	// Build request
	model := c.BaseClient.ResolveModel(options)
	reqBody := ChatCompletionRequest{
		Model:       model,
		Messages:    MessagesToWire(messages, options.SystemPrompt),
		Temperature: c.BaseClient.ResolveTemperature(options),
		MaxTokens:   c.BaseClient.ResolveMaxTokens(options),
		TopP:        c.BaseClient.ResolveTopP(options),
		Stop:        options.Stop,
		Stream:      true,
		ExtraBody:   make(map[string]interface{}),
	}

	// Qwen-specific parameters MUST be in extra_body
	if options.EnableSearch {
		reqBody.ExtraBody["enable_search"] = true
	}
	if options.TopK != nil {
		reqBody.ExtraBody["top_k"] = *options.TopK
	}
	if options.IncrementalOutput {
		reqBody.ExtraBody["incremental_output"] = true
	}
	if options.ThinkingBudget > 0 {
		reqBody.ExtraBody["thinking_budget"] = options.ThinkingBudget
	}

	// Handle stream options for usage
	reqBody.StreamOptions = map[string]interface{}{
		"include_usage": true,
	}

	// Handle PresencePenalty and ParallelToolCalls (OpenAI standard)
	reqBody.PresencePenalty = options.PresencePenalty
	reqBody.FrequencyPenalty = options.FrequencyPenalty
	reqBody.ParallelToolCalls = options.ParallelToolCalls
	reqBody.ToolChoice = options.ToolChoice

	// Handle ResponseFormat
	if options.ResponseFormat != "" {
		reqBody.ResponseFormat = map[string]string{"type": options.ResponseFormat}
	}

	// Handle EnableThinking
	if options.Thinking {
		reqBody.ExtraBody["enable_thinking"] = true
	} else {
		reqBody.ExtraBody["enable_thinking"] = false
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	// NOTES: 所有兼容OpenAI的请求都约定俗成地使用/v1/chat/completions作为请求路径的结尾
	baseURL := strings.TrimSuffix(c.Config().BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL = baseURL + "/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Config().AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Config().AuthToken))
	} else if c.Config().APIKey != "" {
		token := core.GetAPIKey(c.Config().APIKey)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.ParseErrorResponse(resp)
	}

	ch := make(chan core.StreamEvent, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		chunkCh := ParseSSEStream(resp.Body)
		for chunk := range chunkCh {
			select {
			case <-ctx.Done():
				ch <- core.StreamEvent{Type: core.EventError, Err: ctx.Err()}
				return
			default:
			}

			event := StreamChunkToEvent(chunk)
			ch <- event
		}
	}()

	return core.NewStream(ch, resp.Body), nil
}

// doChat performs the actual chat request
func (c *Client) doChat(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	model := c.BaseClient.ResolveModel(options)
	reqBody := ChatCompletionRequest{
		Model:            model,
		Messages:         MessagesToWire(messages, options.SystemPrompt),
		Temperature:      c.BaseClient.ResolveTemperature(options),
		MaxTokens:        c.BaseClient.ResolveMaxTokens(options),
		TopP:             c.BaseClient.ResolveTopP(options),
		Stop:             options.Stop,
		Stream:           stream,
		ExtraBody:        make(map[string]interface{}),
	}

	// Qwen-specific parameters MUST be in extra_body
	if options.EnableSearch {
		reqBody.ExtraBody["enable_search"] = true
	}
	if options.TopK != nil {
		reqBody.ExtraBody["top_k"] = *options.TopK
	}
	if options.IncrementalOutput {
		reqBody.ExtraBody["incremental_output"] = true
	}
	if options.ThinkingBudget > 0 {
		reqBody.ExtraBody["thinking_budget"] = options.ThinkingBudget
	}

	// Handle PresencePenalty and ParallelToolCalls (OpenAI standard)
	reqBody.PresencePenalty = options.PresencePenalty
	reqBody.FrequencyPenalty = options.FrequencyPenalty
	reqBody.ParallelToolCalls = options.ParallelToolCalls
	reqBody.ToolChoice = options.ToolChoice

	// Handle ResponseFormat
	if options.ResponseFormat != "" {
		reqBody.ResponseFormat = map[string]string{"type": options.ResponseFormat}
	}

	// Handle EnableThinking
	if options.Thinking {
		reqBody.ExtraBody["enable_thinking"] = true
	} else {
		reqBody.ExtraBody["enable_thinking"] = false
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}
	baseURL := strings.TrimSuffix(c.Config().BaseURL, "/")
	if !strings.HasSuffix(baseURL, "/v1") {
		baseURL = baseURL + "/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Config().AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Config().AuthToken))
	} else if c.Config().APIKey != "" {
		token := core.GetAPIKey(c.Config().APIKey)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := c.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.ParseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, core.NewNetworkError("failed to read response", err)
	}

	var respData ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, core.NewValidationError("failed to unmarshal response", err)
	}

	return ResponseFromWire(&respData), nil
}
