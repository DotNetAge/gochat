package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openaicompat"
	"github.com/DotNetAge/gochat/pkg/core"
)

// Config defines configuration for the DeepSeek client.
//
// Embed base.Config to inherit common settings like APIKey, Model, Temperature, etc.
//
// Example:
//
//	config := deepseek.Config{
//	    Config: base.Config{
//	        APIKey:      "sk-...",
//	        Model:       "gpt-4",
//	        Temperature: 0.7,
//	        MaxTokens:   1000,
//	    },
//	}
//	client, err := deepseek.New(config)
type Config struct {
	base.Config
}

// Client is an DeepSeek LLM client.
//
// It implements the core.Client interface and provides access to DeepSeek's
// chat completion API, including GPT-3.5, GPT-4, and other models.
//
// The client handles authentication, retries, error handling, and streaming
// automatically. It supports all DeepSeek features including tool calling,
// multimodal inputs, and extended thinking (for o1/o3 models).
type Client struct {
	base *base.Client
}

// New creates a new DeepSeek client
func New(config Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "deepseek-chat"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com"
	}

	baseClient := base.New(config.Config)

	return &Client{
		base: baseClient,
	}, nil
}

// Chat performs a non-streaming chat completion
func (c *Client) Chat(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Response, error) {
	options := core.ApplyOptions(opts...)

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

	// Build request
	model := c.resolveModel(options)
	reqBody := openaicompat.ChatCompletionRequest{
		Model:       model,
		Messages:    openaicompat.MessagesToWire(messages, options.SystemPrompt),
		Temperature: c.resolveTemperature(options),
		MaxTokens:   c.resolveMaxTokens(options),
		TopP:        c.resolveTopP(options),
		Stop:        options.Stop,
		Stream:      true,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.Model = "deepseek-reasoner"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openaicompat.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.base.Config().AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.base.Config().AuthToken))
	} else if c.base.Config().APIKey != "" {
		token := core.GetAPIKey(c.base.Config().APIKey)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := c.base.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, core.NewNetworkError("failed to read response", err)
		}

		var errResp openaicompat.ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, core.NewAPIError(fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
		}

		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan core.StreamEvent, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		chunkCh := openaicompat.ParseSSEStream(resp.Body)
		for chunk := range chunkCh {
			select {
			case <-ctx.Done():
				ch <- core.StreamEvent{Type: core.EventError, Err: ctx.Err()}
				return
			default:
			}

			event := openaicompat.StreamChunkToEvent(chunk)
			ch <- event
		}
	}()

	return core.NewStream(ch, resp.Body), nil
}

// doChat performs the actual chat request
func (c *Client) doChat(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	model := c.resolveModel(options)
	reqBody := openaicompat.ChatCompletionRequest{
		Model:       model,
		Messages:    openaicompat.MessagesToWire(messages, options.SystemPrompt),
		Temperature: c.resolveTemperature(options),
		MaxTokens:   c.resolveMaxTokens(options),
		TopP:        c.resolveTopP(options),
		Stop:        options.Stop,
		Stream:      stream,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.Model = "deepseek-reasoner"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openaicompat.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.base.Config().AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.base.Config().AuthToken))
	} else if c.base.Config().APIKey != "" {
		token := core.GetAPIKey(c.base.Config().APIKey)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := c.base.HTTPClient().Do(req)
	if err != nil {
		return nil, core.NewNetworkError("failed to send request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, core.NewNetworkError("failed to read response", err)
		}

		var errResp openaicompat.ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, core.NewAPIError(fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
		}

		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, core.NewNetworkError("failed to read response", err)
	}

	var respData openaicompat.ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, core.NewValidationError("failed to unmarshal response", err)
	}

	return openaicompat.ResponseFromWire(&respData), nil
}

// Helper methods to resolve options
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

func (c *Client) resolveTopP(opts core.Options) float64 {
	if opts.TopP != nil {
		return *opts.TopP
	}
	return 0 // 0 means not set
}
