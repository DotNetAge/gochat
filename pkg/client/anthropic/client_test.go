package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DotNetAge/gochat/pkg/client/base"
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
					APIKey: "test-key",
					Model:  "claude-3-opus-20240229",
				},
			},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: Config{
				Config: base.Config{
					APIKey: "",
				},
			},
			wantErr: true,
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
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

		var reqBody anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "claude-3-opus-20240229", reqBody.Model)

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "test response"},
			},
			Model:      "claude-3-opus-20240229",
			StopReason: "end_turn",
			Usage: anthropicUsage{
				InputTokens:  10,
				OutputTokens: 20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:  "test-key",
			Model:   "claude-3-opus-20240229",
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
		var reqBody anthropicRequest
		json.NewDecoder(r.Body).Decode(&reqBody)

		assert.True(t, reqBody.Stream)

		flusher, _ := w.(http.Flusher)

		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"type":"content_block_start","index":0}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
			`data: {"type":"content_block_stop","index":0}`,
			`data: {"type":"message_delta","usage":{"output_tokens":20}}`,
			`data: {"type":"message_stop"}`,
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
			Model:   "claude-3-opus-20240229",
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

func TestClient_ChatWithSystemPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody anthropicRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Verify system prompt was set
		assert.Equal(t, "You are a helpful assistant", reqBody.System)

		response := anthropicResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []contentBlock{
				{Type: "text", Text: "response"},
			},
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 20},
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			APIKey:  "test-key",
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithSystemPrompt("You are a helpful assistant"),
	)
	require.NoError(t, err)
	assert.Equal(t, "response", response.Content)
}

func TestClient_Chat_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`))
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
	assert.Contains(t, err.Error(), "authentication_error")
}
