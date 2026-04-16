package azureopenai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
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
	core.Config
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
	*core.BaseClient
	endpoint   string
	apiVersion string
}

// New creates a new Azure OpenAI client
func NewAzureOpenAI(config Config) (*Client, error) {
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

	baseClient := core.NewClient(config.Config)

	return &Client{
		BaseClient: baseClient,
		endpoint:   config.Endpoint,
		apiVersion: config.APIVersion,
	}, nil
}

// Chat performs a non-streaming chat completion
func (c *Client) Chat(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Response, error) {
	options := core.ApplyOptions(opts...)
	messages = core.ProcessAttachments(messages, options.Attachments)

	var response *core.Response
	err := c.Retry(ctx, func() error {
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
	model := c.ResolveModel(options)
	reqBody := openai.ChatCompletionRequest{
		Model:        model,
		Messages:     openai.MessagesToWire(messages, options.SystemPrompt),
		Temperature:  c.ResolveTemperature(options),
		MaxTokens:    c.ResolveMaxTokens(options),
		TopP:         c.ResolveTopP(options),
		Stop:         options.Stop,
		Stream:       true,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.ReasoningEffort = "high"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openai.ToolsToWire(options.Tools)
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
	req.Header.Set("api-key", core.GetAPIKey(c.Config().APIKey))

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

// doChat performs the actual chat request
func (c *Client) doChat(ctx context.Context, messages []core.Message, options core.Options, stream bool) (*core.Response, error) {
	model := c.ResolveModel(options)
	reqBody := openai.ChatCompletionRequest{
		Model:        model,
		Messages:     openai.MessagesToWire(messages, options.SystemPrompt),
		Temperature:  c.ResolveTemperature(options),
		MaxTokens:    c.ResolveMaxTokens(options),
		TopP:         c.ResolveTopP(options),
		Stop:         options.Stop,
		Stream:       stream,
		EnableSearch: options.EnableSearch,
	}

	if options.Thinking {
		reqBody.ReasoningEffort = "high"
	}

	if len(options.Tools) > 0 {
		reqBody.Tools = openai.ToolsToWire(options.Tools)
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
	req.Header.Set("api-key", core.GetAPIKey(c.Config().APIKey))

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
