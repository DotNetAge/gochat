package compatible

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

// Config configures the OpenAI-compatible client
type Config struct {
	base.Config
}

// Client implements an LLM client for OpenAI-compatible APIs
type Client struct {
	base *base.Client
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Choice represents a choice in the chat completion response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Usage represents usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Delta represents a delta in the streaming response
type Delta struct {
	Content string `json:"content,omitempty"`
}

// StreamChoice represents a choice in the streaming response
type StreamChoice struct {
	Index        int    `json:"index"`
	Delta        *Delta `json:"delta,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// StreamChunk represents a chunk in the streaming response
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

// Error represents an error response
type Error struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error Error `json:"error"`
}

// New creates a new OpenAI-compatible client
func New(config Config) (*Client, error) {
	if config.APIKey == "" && config.AuthToken == "" {
		return nil, core.NewValidationError("either api key or auth token is required", nil)
	}

	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com"
	}

	baseClient := base.New(config.Config)

	return &Client{
		base: baseClient,
	}, nil
}

// Complete generates a completion for the given prompt
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	var response *ChatCompletionResponse

	err := c.base.Retry(ctx, func() error {
		resp, err := c.doComplete(ctx, prompt, false)
		if err != nil {
			return err
		}
		response = resp
		return nil
	})

	if err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", nil
}

// CompleteStream generates a completion for the given prompt and returns a channel for streaming responses
func (c *Client) CompleteStream(ctx context.Context, prompt string) (<-chan string, error) {
	reqBody := ChatCompletionRequest{
		Model: c.base.Config().Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.base.Config().Temperature,
		MaxTokens:   c.base.Config().MaxTokens,
		Stream:      true,
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

		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, core.NewAPIError(fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
		}

		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	ch := make(chan string, 10)

	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			chunk, err := parseStreamChunk(line)
			if err != nil {
				select {
				case ch <- "ERROR: " + err.Error():
				case <-ctx.Done():
				}
				return
			}

			if chunk != nil && len(chunk.Choices) > 0 && chunk.Choices[0].Delta != nil {
				select {
				case ch <- chunk.Choices[0].Delta.Content:
				case <-ctx.Done():
					return
				}
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case ch <- "ERROR: " + err.Error():
			case <-ctx.Done():
			}
		}
	}()

	return ch, nil
}

// doComplete performs a completion request
func (c *Client) doComplete(ctx context.Context, prompt string, stream bool) (*ChatCompletionResponse, error) {
	reqBody := ChatCompletionRequest{
		Model: c.base.Config().Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: c.base.Config().Temperature,
		MaxTokens:   c.base.Config().MaxTokens,
		Stream:      stream,
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

		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, core.NewAPIError(fmt.Sprintf("%s: %s", errResp.Error.Type, errResp.Error.Message), nil)
		}

		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, core.NewNetworkError("failed to read response", err)
	}

	var respData ChatCompletionResponse
	if err := json.Unmarshal(body, &respData); err != nil {
		return nil, core.NewValidationError("failed to unmarshal response", err)
	}

	return &respData, nil
}

// parseStreamChunk parses a stream chunk
func parseStreamChunk(line string) (*StreamChunk, error) {
	if !strings.HasPrefix(line, "data: ") {
		return nil, nil
	}

	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return nil, nil
	}

	var chunk StreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, core.NewValidationError("failed to unmarshal chunk", err)
	}

	return &chunk, nil
}
