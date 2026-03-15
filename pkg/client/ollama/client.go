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

// Config defines configuration for the Ollama client.
//
// Ollama is for running LLMs locally. It doesn't require an API key.
//
// Example:
//
//	config := ollama.Config{
//	    Config: base.Config{
//	        Model:   "llama2",
//	        BaseURL: "http://localhost:11434",
//	    },
//	}
//	client, err := ollama.New(config)
type Config struct {
	base.Config
}

// Client is an Ollama LLM client.
//
// Ollama allows you to run large language models locally on your machine.
// This client connects to a local Ollama server (default: localhost:11434)
// and provides the same interface as cloud-based providers.
//
// No API key is required. The client uses a longer default timeout (60s)
// since local models may take more time to generate responses.
type Client struct {
	base *base.Client
}

// Ollama-specific wire format
type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"` // max tokens
}

type ollamaResponse struct {
	Model           string         `json:"model"`
	CreatedAt       string         `json:"created_at"`
	Message         *ollamaMessage `json:"message,omitempty"`
	Done            bool           `json:"done"`
	DoneReason      string         `json:"done_reason,omitempty"`
	PromptEvalCount int            `json:"prompt_eval_count,omitempty"`
	EvalCount       int            `json:"eval_count,omitempty"`
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
		config.Timeout = 60 * time.Second // Ollama needs longer timeout
	}

	baseClient := base.New(config.Config)

	return &Client{
		base: baseClient,
	}, nil
}

// DefaultOllamaClient 创建默认的 Ollama 客户端
// 使用 qwen3.5:0.8b 模型，适用于本地智能分块和图提取
func DefaultOllamaClient() (*Client, error) {
	return New(Config{
		Config: base.Config{
			Model:   "qwen3.5:0.8b",
			BaseURL: "http://localhost:11434",
		},
	})
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

	if options.UsageCallback != nil && response.Usage != nil {
		options.UsageCallback(*response.Usage)
	}

	return response, nil
}

// ChatStream performs a streaming chat completion
func (c *Client) ChatStream(ctx context.Context, messages []core.Message, opts ...core.Option) (*core.Stream, error) {
	options := core.ApplyOptions(opts...)

	reqBody := c.buildRequest(messages, options, true)

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
		body, _ := io.ReadAll(resp.Body)
		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
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

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				continue
			}

			if chunk.Message != nil && chunk.Message.Content != "" {
				ch <- core.StreamEvent{
					Type:    core.EventContent,
					Content: chunk.Message.Content,
				}
			}

			if chunk.Done {
				usage := &core.Usage{
					PromptTokens:     chunk.PromptEvalCount,
					CompletionTokens: chunk.EvalCount,
					TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
				}
				ch <- core.StreamEvent{
					Type:  core.EventDone,
					Usage: usage,
				}
				return
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, core.NewAPIError(fmt.Sprintf("request failed with status %d: %s", resp.StatusCode, string(body)), nil)
	}

	// Ollama returns streaming format even for non-streaming requests
	// We need to accumulate all chunks
	var content string
	var usage core.Usage

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}

		if chunk.Message != nil {
			content += chunk.Message.Content
		}

		if chunk.Done {
			usage = core.Usage{
				PromptTokens:     chunk.PromptEvalCount,
				CompletionTokens: chunk.EvalCount,
				TotalTokens:      chunk.PromptEvalCount + chunk.EvalCount,
			}
		}
	}

	return &core.Response{
		Model:   c.resolveModel(options),
		Content: content,
		Message: core.Message{
			Role: core.RoleAssistant,
			Content: []core.ContentBlock{
				{Type: core.ContentTypeText, Text: content},
			},
		},
		Usage: &usage,
	}, nil
}

// buildRequest builds an Ollama API request
func (c *Client) buildRequest(messages []core.Message, options core.Options, stream bool) ollamaRequest {
	var ollamaMessages []ollamaMessage

	// Add system prompt if provided
	if options.SystemPrompt != "" {
		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    core.RoleSystem,
			Content: options.SystemPrompt,
		})
	}

	// Convert messages
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    msg.Role,
			Content: msg.TextContent(),
		})
	}

	req := ollamaRequest{
		Model:    c.resolveModel(options),
		Messages: ollamaMessages,
		Stream:   stream,
	}

	// Add options if any are set
	if options.Temperature != nil || options.MaxTokens != nil {
		req.Options = &ollamaOptions{}
		if options.Temperature != nil {
			req.Options.Temperature = *options.Temperature
		} else {
			req.Options.Temperature = c.base.Config().Temperature
		}
		if options.MaxTokens != nil {
			req.Options.NumPredict = *options.MaxTokens
		} else if c.base.Config().MaxTokens > 0 {
			req.Options.NumPredict = c.base.Config().MaxTokens
		}
	}

	return req
}

// Helper methods
func (c *Client) resolveModel(opts core.Options) string {
	if opts.Model != "" {
		return opts.Model
	}
	return c.base.Config().Model
}
