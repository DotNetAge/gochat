package compatible

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openaicompat"
	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Config: base.Config{
					APIKey:  "test-key",
					Model:   "gpt-3.5-turbo",
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: Config{
				Config: base.Config{
					Model:   "gpt-3.5-turbo",
					BaseURL: "https://api.example.com",
				},
			},
			wantErr: true,
		},
		{
			name: "missing base url",
			config: Config{
				Config: base.Config{
					APIKey: "test-key",
					Model:  "gpt-3.5-turbo",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var reqBody openaicompat.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "gpt-3.5-turbo", reqBody.Model)

		response := openaicompat.ChatCompletionResponse{
			ID:      "chatcmpl-abc123",
			Object:  "chat.completion",
			Created: 1699000000,
			Model:   "gpt-3.5-turbo",
			Choices: []openaicompat.Choice{
				{
					Index: 0,
					Message: openaicompat.Message{
						Role:    "assistant",
						Content: "test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: openaicompat.Usage{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:  "test-key",
			Model:   "gpt-3.5-turbo",
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test prompt"),
	}
	response, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)
	assert.Equal(t, "test response", response.Content)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody openaicompat.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&reqBody)

		assert.True(t, reqBody.Stream)

		flusher, _ := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-abc123","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:  "test-key",
			Model:   "gpt-3.5-turbo",
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test prompt"),
	}
	stream, err := client.ChatStream(context.Background(), messages)
	require.NoError(t, err)
	defer stream.Close()

	var result string
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			t.Fatal(ev.Err)
		}
		if ev.Type == core.EventContent {
			result += ev.Content
		}
	}

	assert.Equal(t, "Hello world", result)
}

func TestClient_ChatWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody openaicompat.ChatCompletionRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Verify options were applied
		assert.Equal(t, 0.9, reqBody.Temperature)
		assert.Equal(t, 500, reqBody.MaxTokens)
		assert.Len(t, reqBody.Messages, 2) // system + user

		response := openaicompat.ChatCompletionResponse{
			ID:    "chatcmpl-abc123",
			Model: "gpt-4",
			Choices: []openaicompat.Choice{
				{
					Message: openaicompat.Message{
						Role:    "assistant",
						Content: "response",
					},
					FinishReason: "stop",
				},
			},
			Usage: openaicompat.Usage{TotalTokens: 50},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:  "test-key",
			Model:   "gpt-3.5-turbo",
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithModel("gpt-4"),
		core.WithTemperature(0.9),
		core.WithMaxTokens(500),
		core.WithSystemPrompt("You are a helpful assistant"),
	)
	require.NoError(t, err)
	assert.Equal(t, "response", response.Content)
}

func TestClient_Chat_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`))
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:     "test-key",
			BaseURL:    server.URL,
			MaxRetries: 0,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{core.NewUserMessage("test")}
	_, err = client.Chat(context.Background(), messages)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}
