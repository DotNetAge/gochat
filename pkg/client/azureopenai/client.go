package azureopenai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openaicompat"
	"github.com/DotNetAge/gochat/pkg/core"
)

// Config defines configuration for the Azure OpenAI client.
//
// Azure OpenAI requires additional configuration beyond the standard OpenAI client:
// - Endpoint: Your Azure OpenAI resource endpoint
// - APIVersion: The API version to use (e.g., "2024-03-01-preview")
// - Model: Your deployment name (not the model name)
//
// Example:
//
//	config := azureopenai.Config{
//	    Config: base.Config{
//	        APIKey: "your-azure-key",
//	        Model:  "my-gpt4-deployment", // deployment name, not "gpt-4"
//	    },
//	    Endpoint:   "https://your-resource.openai.azure.com",
//	    APIVersion: "2024-03-01-preview",
//	}
//	client, err := azureopenai.New(config)
type Config struct {
	base.Config
	Endpoint   string // Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com)
	APIVersion string // API version (e.g., 2024-03-01-preview)
}

// Client is an Azure OpenAI LLM client.
//
// Azure OpenAI provides OpenAI models through Microsoft Azure.
// The API is similar to OpenAI's but uses different authentication
// (api-key header instead of Bearer token) and requires deployment names
// instead of model names.
type Client struct {
	base       *base.Client
	endpoint   string
	apiVersion string
}

// New creates a new Azure OpenAI client
func New(config Config) (*Client, error) {
	if config.APIKey == "" {
		return nil, core.NewValidationError("api key is required", nil)
	}

	if config.Model == "" {
		return nil, core.NewValidationError("model (deployment name) is required", nil)
	}

	if config.Endpoint == "" {
		return nil, core.NewValidationError("endpoint is required", nil)
	}

	if config.APIVersion == "" {
		config.APIVersion = "2024-03-01-preview"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	baseClient := base.New(config.Config)

	return &Client{
		base:       baseClient,
		endpoint:   config.Endpoint,
		apiVersion: config.APIVersion,
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
		reqBody.ReasoningEffort = "high"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openaicompat.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	// Azure OpenAI uses deployment name in URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, model, c.apiVersion)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", core.GetAPIKey(c.base.Config().APIKey))

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
		reqBody.ReasoningEffort = "high"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openaicompat.ToolsToWire(options.Tools)
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, core.NewValidationError("failed to marshal request", err)
	}

	// Azure OpenAI uses deployment name in URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		c.endpoint, model, c.apiVersion)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", core.GetAPIKey(c.base.Config().APIKey))

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
	return 0
}
