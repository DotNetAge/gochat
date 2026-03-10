package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/core"
)

// Config configures the Ollama client
type Config struct {
	base.Config
}

// Client implements an LLM client for Ollama
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
	Model              string  `json:"model"`
	CreatedAt          string  `json:"created_at"`
	Message            Message `json:"message"`
	Done               bool    `json:"done"`
	DoneReason         string  `json:"done_reason"`
	TotalDuration      int64   `json:"total_duration"`
	LoadDuration       int64   `json:"load_duration"`
	PromptEvalCount    int     `json:"prompt_eval_count"`
	PromptEvalDuration int64   `json:"prompt_eval_duration"`
	EvalCount          int     `json:"eval_count"`
	EvalDuration       int64   `json:"eval_duration"`
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
	Model              string   `json:"model"`
	CreatedAt          string   `json:"created_at"`
	Message            *Message `json:"message,omitempty"`
	Delta              *Delta   `json:"delta,omitempty"`
	Done               bool     `json:"done"`
	DoneReason         string   `json:"done_reason,omitempty"`
	TotalDuration      int64    `json:"total_duration,omitempty"`
	LoadDuration       int64    `json:"load_duration,omitempty"`
	PromptEvalCount    int      `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64    `json:"prompt_eval_duration,omitempty"`
	EvalCount          int      `json:"eval_count,omitempty"`
	EvalDuration       int64    `json:"eval_duration,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// New creates a new Ollama client
func New(config Config) (*Client, error) {
	if config.Model == "" {
		config.Model = "llama2"
	}

	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}

	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	baseClient := base.New(config.Config)

	return &Client{
		base: baseClient,
	}, nil
}

// Complete performs a non-streaming completion
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

	return response.Message.Content, nil
}

// CompleteStream performs a streaming completion
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

	url := fmt.Sprintf("%s/api/chat", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
			return nil, core.NewAPIError(fmt.Sprintf("error: %s", errResp.Error), nil)
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

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				select {
				case ch <- "ERROR: " + err.Error():
				case <-ctx.Done():
				}
				return
			}

			if chunk.Delta != nil && chunk.Delta.Content != "" {
				select {
				case ch <- chunk.Delta.Content:
				case <-ctx.Done():
					return
				}
			} else if chunk.Message != nil && chunk.Message.Content != "" {
				select {
				case ch <- chunk.Message.Content:
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

	url := fmt.Sprintf("%s/api/chat", c.base.Config().BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, core.NewNetworkError("failed to create request", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
			return nil, core.NewAPIError(fmt.Sprintf("error: %s", errResp.Error), nil)
		}

		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	defer resp.Body.Close()

	// Read response line by line
	scanner := bufio.NewScanner(resp.Body)
	var fullContent string
	var lastChunk StreamChunk

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return nil, core.NewValidationError("failed to unmarshal chunk", err)
		}

		lastChunk = chunk

		if chunk.Delta != nil && chunk.Delta.Content != "" {
			fullContent += chunk.Delta.Content
		} else if chunk.Message != nil && chunk.Message.Content != "" {
			fullContent += chunk.Message.Content
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, core.NewNetworkError("failed to read response", err)
	}

	// Create final response
	response := &ChatCompletionResponse{
		Model:     lastChunk.Model,
		CreatedAt: lastChunk.CreatedAt,
		Message: Message{
			Role:    "assistant",
			Content: fullContent,
		},
		Done:               lastChunk.Done,
		DoneReason:         lastChunk.DoneReason,
		TotalDuration:      lastChunk.TotalDuration,
		LoadDuration:       lastChunk.LoadDuration,
		PromptEvalCount:    lastChunk.PromptEvalCount,
		PromptEvalDuration: lastChunk.PromptEvalDuration,
		EvalCount:          lastChunk.EvalCount,
		EvalDuration:       lastChunk.EvalDuration,
	}

	return response, nil
}
