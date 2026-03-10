package base

import (
	"context"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/pkg/core"
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

// Client implements a base client with common functionality
type Client struct {
	config     Config
	httpClient *http.Client
}

// New creates a new base client
func New(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Retry executes a function with retry logic
func (c *Client) Retry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(core.ExponentialBackoff(attempt, time.Second)):
			}
		}

		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if !core.IsRetryableError(err) {
				break
			}
		}
	}

	return lastErr
}

// HTTPClient returns the underlying HTTP client
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// Config returns the client configuration
func (c *Client) Config() Config {
	return c.config
}
