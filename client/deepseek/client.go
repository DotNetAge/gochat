package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

// Client is an DeepSeek LLM client.
//
// It implements the core.Client interface and provides access to DeepSeek's
// chat completion API, including GPT-3.5, GPT-4, and other models.
//
// The client handles authentication, retries, error handling, and streaming
// automatically. It supports all DeepSeek features including tool calling,
// multimodal inputs, and extended thinking (for o1/o3 models).
type Client struct {
	*core.BaseClient
}

// New creates a new DeepSeek client
func NewDeepSeek(config core.Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "deepseek-chat"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com"
	}

	baseClient := core.NewClient(config)

	client := &Client{
		BaseClient: baseClient,
	}

	// Inject chat and stream functions
	client.SetChatFunc(client.doChatInternal)
	client.SetStreamFunc(client.doChatStream)

	return client, nil
}

// ChatStream performs a streaming chat completion
func (c *Client) ChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	return c.doChatStream(ctx, messages, opts...)
}

// doChatStream performs the actual streaming chat request
func (c *Client) doChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	// Build request
	model := c.BaseClient.ResolveModel(options)
	reqBody := openai.ChatCompletionRequest{
		Model:        model,
		Messages:     openai.MessagesToWire(messages, options.SystemPrompt),
		Temperature:  c.BaseClient.ResolveTemperature(options),
		MaxTokens:    c.BaseClient.ResolveMaxTokens(options),
		TopP:         c.BaseClient.ResolveTopP(options),
		Stop:         options.Stop,
		Stream:       true,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.Model = "deepseek-reasoner"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openai.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.Config().BaseURL)
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

		chunkCh := openai.ParseSSEStream(resp.Body)
		for chunk := range chunkCh {
			select {
			case <-ctx.Done():
				ch <- core.StreamEvent{Type: core.EventError, Err: ctx.Err()}
				return
			default:
			}

			event := openai.StreamChunkToEvent(chunk)
			ch <- event
		}
	}()

	return core.NewStream(ch, resp.Body), nil
}

// doChatInternal performs the actual chat request
func (c *Client) doChatInternal(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	model := c.BaseClient.ResolveModel(options)
	reqBody := openai.ChatCompletionRequest{
		Model:        model,
		Messages:     openai.MessagesToWire(messages, options.SystemPrompt),
		Temperature:  c.BaseClient.ResolveTemperature(options),
		MaxTokens:    c.BaseClient.ResolveMaxTokens(options),
		TopP:         c.BaseClient.ResolveTopP(options),
		Stop:         options.Stop,
		Stream:       stream,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.Model = "deepseek-reasoner"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openai.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.Config().BaseURL)
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

	var respData openai.ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, core.NewValidationError("failed to unmarshal response", err)
	}

	return openai.ResponseFromWire(&respData), nil
}
