package azureopenai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Chat_Full(t *testing.T) {
	deployment := "my-gpt4"
	apiVersion := "2024-03-01-preview"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/openai/deployments/"+deployment+"/chat/completions")
		assert.Equal(t, apiVersion, r.URL.Query().Get("api-version"))
		assert.Equal(t, "sk-test", r.Header.Get("api-key"))

		resp := openai.ChatCompletionResponse{
			ID: "test-id",
			Choices: []openai.Choice{
				{
					Message: openai.Message{
						Role:    "assistant",
						Content: "azure hello",
					},
				},
			},
			Usage: openai.Usage{TotalTokens: 100},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewAzureOpenAI(Config{
		Config: core.Config{
			APIKey: "sk-test",
			Model:  deployment,
		},
		Endpoint:   server.URL,
		APIVersion: apiVersion,
	})
	require.NoError(t, err)

	usageCalled := false
	resp, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")}, core.WithUsageCallback(func(u core.Usage) {
		usageCalled = true
	}))
	require.NoError(t, err)
	assert.Equal(t, "azure hello", resp.Content)
	assert.True(t, usageCalled)
}

func TestClient_ChatStream(t *testing.T) {
	deployment := "my-gpt4"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, _ := NewAzureOpenAI(Config{
		Config:   core.Config{APIKey: "key", Model: deployment},
		Endpoint: server.URL,
	})

	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("hi")})
	require.NoError(t, err)
	defer stream.Close()

	var content string
	for stream.Next() {
		if stream.Event().Type == core.EventContent {
			content += stream.Event().Content
		}
	}
	assert.Equal(t, "hello", content)
}

func TestClient_ChatStream_Error_Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client, _ := NewAzureOpenAI(Config{
		Config:   core.Config{APIKey: "key", Model: "deploy"},
		Endpoint: server.URL,
	})

	stream, err := client.ChatStream(context.Background(), []core.Message{core.NewUserMessage("hi")})
	assert.Error(t, err)
	assert.Nil(t, stream)
}

func TestClient_Options(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openai.ChatCompletionRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, 0.5, req.Temperature)
		assert.Equal(t, 100, req.MaxTokens)
		assert.Equal(t, 0.9, req.TopP)

		resp := openai.ChatCompletionResponse{
			Choices: []openai.Choice{{Message: openai.Message{Content: "ok"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := NewAzureOpenAI(Config{
		Config:   core.Config{APIKey: "key", Model: "deploy"},
		Endpoint: server.URL,
	})

	temp := 0.5
	maxTokens := 100
	topP := 0.9
	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")},
		core.WithTemperature(temp),
		core.WithMaxTokens(maxTokens),
		core.WithTopP(topP),
	)
	assert.NoError(t, err)
}

func TestClient_ResolveModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/openai/deployments/override-model/chat/completions")
		resp := openai.ChatCompletionResponse{
			Choices: []openai.Choice{{Message: openai.Message{Content: "ok"}}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, _ := NewAzureOpenAI(Config{
		Config:   core.Config{APIKey: "key", Model: "default-model"},
		Endpoint: server.URL,
	})

	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")}, core.WithModel("override-model"))
	assert.NoError(t, err)
}

func TestClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"bad request","type":"invalid_request"}}`))
	}))
	defer server.Close()

	client, _ := NewAzureOpenAI(Config{
		Config:   core.Config{APIKey: "key", Model: "deploy"},
		Endpoint: server.URL,
	})

	_, err := client.Chat(context.Background(), []core.Message{core.NewUserMessage("hi")})
	assert.Error(t, err)
}

func TestNew_Validation(t *testing.T) {
	_, err := NewAzureOpenAI(Config{Config: core.Config{Model: "deploy"}, Endpoint: "http://test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api key")

	_, err = NewAzureOpenAI(Config{Config: core.Config{APIKey: "key"}, Endpoint: "http://test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "model")

	_, err = NewAzureOpenAI(Config{Config: core.Config{APIKey: "key", Model: "deploy"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}
