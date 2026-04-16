package core

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Config defines common configuration for all LLM clients
type Config struct {
	APIKey      string // API key for the LLM provider
	AuthToken   string // Auth token for the LLM provider (alternative to APIKey)
	Model       string
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	Temperature float64
	MaxTokens   int
}

// ChatFunc is the function signature for chat execution
type ChatFunc func(ctx context.Context, messages []Message, options Options, stream bool) (*Response, error)

// StreamFunc is the function signature for streaming chat execution
type StreamFunc func(ctx context.Context, messages []Message, opts ...Option) (*Stream, error)

// Client implements a base client with common functionality
type BaseClient struct {
	config     Config
	httpClient *http.Client
	doChat     ChatFunc      // injected chat execution function
	doStream   StreamFunc    // injected stream execution function
}

// New creates a new base client
func NewClient(config Config) *BaseClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &BaseClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SetChatFunc sets the chat execution function
func (c *BaseClient) SetChatFunc(fn ChatFunc) {
	c.doChat = fn
}

// SetStreamFunc sets the stream execution function
func (c *BaseClient) SetStreamFunc(fn StreamFunc) {
	c.doStream = fn
}

// Retry executes a function with retry logic
func (c *BaseClient) Retry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(ExponentialBackoff(attempt, time.Second)):
			}
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if !IsRetryableError(err) {
				break
			}
		}
	}

	return lastErr
}

// HTTPClient returns the underlying HTTP client
func (c *BaseClient) HTTPClient() *http.Client {
	return c.httpClient
}

// Config returns the client configuration
func (c *BaseClient) Config() Config {
	return c.config
}

// ResolveModel returns the model from options if set, otherwise from config
func (c *BaseClient) ResolveModel(opts Options) string {
	if opts.Model != "" {
		return opts.Model
	}
	return c.config.Model
}

// ResolveTemperature returns the temperature from options if set, otherwise from config
func (c *BaseClient) ResolveTemperature(opts Options) float64 {
	if opts.Temperature != nil {
		return *opts.Temperature
	}
	return c.config.Temperature
}

// ResolveMaxTokens returns the max tokens from options if set, otherwise from config
func (c *BaseClient) ResolveMaxTokens(opts Options) int {
	if opts.MaxTokens != nil {
		return *opts.MaxTokens
	}
	return c.config.MaxTokens
}

// ResolveTopP returns the top-p from options if set, otherwise returns 0
func (c *BaseClient) ResolveTopP(opts Options) float64 {
	if opts.TopP != nil {
		return *opts.TopP
	}
	return 0 // 0 means not set
}

// ParseErrorResponse parses error response from HTTP response
func (c *BaseClient) ParseErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError("failed to read error response", err)
	}
	return NewAPIErrorFromResponse(resp.StatusCode, body)
}

// Chat sends messages and returns a complete response
//
// Parameters:
// - ctx: Context for cancellation and timeout
// - messages: Conversation messages
// - opts: Optional parameters (temperature, max tokens, tools, etc.)
//
// Returns:
// - *Response: Complete response with content, usage, and tool calls
// - error: Error if the request fails
func (c *BaseClient) Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error) {
	if c.doChat == nil {
		return nil, NewValidationError("chat function not set", nil)
	}

	options := ApplyOptions(opts...)
	messages = ProcessAttachments(messages, options.Attachments)

	var response *Response
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

	// Call usage callback if provided
	if options.UsageCallback != nil && response.Usage != nil {
		options.UsageCallback(*response.Usage)
	}

	return response, nil
}

// ChatStream sends messages and returns a stream of events
//
// Parameters:
// - ctx: Context for cancellation and timeout
// - messages: Conversation messages
// - opts: Optional parameters (temperature, max tokens, tools, etc.)
//
// Returns:
// - *Stream: Stream of events (content, usage, errors)
// - error: Error if the request fails to start
func (c *BaseClient) ChatStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error) {
	if c.doStream == nil {
		return nil, NewValidationError("stream function not set", nil)
	}

	return c.doStream(ctx, messages, opts...)
}
