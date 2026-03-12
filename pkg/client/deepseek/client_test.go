package deepseek

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

func TestClient_Chat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		resp := openaicompat.ChatCompletionResponse{
			ID:    "test-id",
			Model: "deepseek-chat",
			Choices: []openaicompat.Choice{
				{
					Message: openaicompat.Message{
						Role:    "assistant",
						Content: "hello world",
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
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := New(Config{
		Config: base.Config{APIKey: "sk-test", BaseURL: server.URL},
	})

	resp, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	assert.Equal(t, "hello world", resp.Content)
	assert.Equal(t, 30, resp.Usage.TotalTokens)
}

func TestClient_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, _ := New(Config{
		Config: base.Config{APIKey: "sk-test", BaseURL: server.URL},
	})

	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	defer stream.Close()

	var result string
	for stream.Next() {
		if stream.Event().Type == core.EventContent {
			result += stream.Event().Content
		}
	}
	assert.Equal(t, "hello", result)
}

func TestClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid key","type":"auth_error"}}`))
	}))
	defer server.Close()

	client, _ := New(Config{
		Config: base.Config{APIKey: "key", BaseURL: server.URL},
	})

	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	assert.Error(t, err)
}

func TestClient_Options(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaicompat.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		
		assert.Equal(t, "deepseek-reasoner", req.Model)
		
		resp := openaicompat.ChatCompletionResponse{
			Choices: []openaicompat.Choice{{Message: openaicompat.Message{Content: "thinking"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := New(Config{
		Config: base.Config{APIKey: "key", BaseURL: server.URL},
	})

	// WithThinking(1000) should switch model to deepseek-reasoner
	client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")}, core.WithThinking(1000))
}
