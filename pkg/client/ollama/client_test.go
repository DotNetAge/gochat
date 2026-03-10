package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		client, err := New(Config{})
		require.NoError(t, err)
		assert.Equal(t, "llama2", client.base.Config().Model)
		assert.Equal(t, "http://localhost:11434", client.base.Config().BaseURL)
		assert.Equal(t, 60*time.Second, client.base.Config().Timeout)
		assert.Equal(t, 3, client.base.Config().MaxRetries)
		assert.Equal(t, 0.7, client.base.Config().Temperature)
	})

	t.Run("Custom configuration", func(t *testing.T) {
		client, err := New(Config{
			Config: base.Config{
				Model:       "mistral",
				BaseURL:     "http://localhost:11435",
				Timeout:     30 * time.Second,
				MaxRetries:  5,
				Temperature: 0.5,
				MaxTokens:   1000,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "mistral", client.base.Config().Model)
		assert.Equal(t, "http://localhost:11435", client.base.Config().BaseURL)
		assert.Equal(t, 30*time.Second, client.base.Config().Timeout)
		assert.Equal(t, 5, client.base.Config().MaxRetries)
		assert.Equal(t, 0.5, client.base.Config().Temperature)
		assert.Equal(t, 1000, client.base.Config().MaxTokens)
	})
}

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)

		var reqBody ollamaRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, "llama2", reqBody.Model)
		assert.False(t, reqBody.Stream)

		// Ollama returns streaming format even for non-streaming
		responses := []ollamaResponse{
			{
				Model:     "llama2",
				CreatedAt: "2024-01-01T00:00:00Z",
				Message:   &ollamaMessage{Role: "assistant", Content: "Hello! "},
				Done:      false,
			},
			{
				Model:     "llama2",
				CreatedAt: "2024-01-01T00:00:00Z",
				Message:   &ollamaMessage{Role: "assistant", Content: "I'm Ollama."},
				Done:      false,
			},
			{
				Model:           "llama2",
				CreatedAt:       "2024-01-01T00:00:00Z",
				Done:            true,
				PromptEvalCount: 10,
				EvalCount:       20,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		for _, resp := range responses {
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("Hello"),
	}
	response, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)
	assert.Equal(t, "Hello! I'm Ollama.", response.Content)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 30, response.Usage.TotalTokens)
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody ollamaRequest
		json.NewDecoder(r.Body).Decode(&reqBody)

		assert.True(t, reqBody.Stream)

		w.Header().Set("Content-Type", "application/json")

		responses := []ollamaResponse{
			{
				Model:     "llama2",
				CreatedAt: "2024-01-01T00:00:00Z",
				Message:   &ollamaMessage{Role: "assistant", Content: "Hello"},
				Done:      false,
			},
			{
				Model:     "llama2",
				CreatedAt: "2024-01-01T00:00:00Z",
				Message:   &ollamaMessage{Role: "assistant", Content: " world"},
				Done:      false,
			},
			{
				Model:           "llama2",
				CreatedAt:       "2024-01-01T00:00:00Z",
				Done:            true,
				PromptEvalCount: 5,
				EvalCount:       10,
			},
		}

		for _, resp := range responses {
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test"),
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
	assert.NotNil(t, stream.Usage())
	assert.Equal(t, 15, stream.Usage().TotalTokens)
}

func TestClient_ChatWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody ollamaRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		// Verify options were applied
		assert.NotNil(t, reqBody.Options)
		assert.Equal(t, 0.9, reqBody.Options.Temperature)
		assert.Equal(t, 500, reqBody.Options.NumPredict)

		// Verify system prompt
		assert.Len(t, reqBody.Messages, 2) // system + user
		assert.Equal(t, "system", reqBody.Messages[0].Role)

		response := ollamaResponse{
			Model:           "llama2",
			Message:         &ollamaMessage{Role: "assistant", Content: "response"},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       20,
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(Config{
		Config: base.Config{
			BaseURL: server.URL,
		},
	})
	require.NoError(t, err)

	messages := []core.Message{
		core.NewUserMessage("test"),
	}

	response, err := client.Chat(context.Background(), messages,
		core.WithTemperature(0.9),
		core.WithMaxTokens(500),
		core.WithSystemPrompt("You are a helpful assistant"),
	)
	require.NoError(t, err)
	assert.Equal(t, "response", response.Content)
}
