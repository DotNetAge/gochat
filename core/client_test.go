package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements the Client interface for testing
type mockClient struct{}

func (m *mockClient) Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error) {
	return &Response{
		Content: "Mock completion",
		Usage: &Usage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
		},
	}, nil
}

func (m *mockClient) ChatStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error) {
	ch := make(chan StreamEvent, 3)
	ch <- StreamEvent{Type: EventContent, Content: "Mock"}
	ch <- StreamEvent{Type: EventContent, Content: " stream"}
	ch <- StreamEvent{Type: EventDone, Usage: &Usage{TotalTokens: 10}}
	close(ch)
	return NewStream(ch, nil), nil
}

func TestClient_Chat(t *testing.T) {
	client := &mockClient{}

	messages := []Message{
		NewUserMessage("test prompt"),
	}

	response, err := client.Chat(context.Background(), messages)
	require.NoError(t, err)
	assert.Equal(t, "Mock completion", response.Content)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 15, response.Usage.TotalTokens)
}

func TestClient_ChatStream(t *testing.T) {
	client := &mockClient{}

	messages := []Message{
		NewUserMessage("test prompt"),
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
		if ev.Type == EventContent {
			result += ev.Content
		}
	}

	assert.Equal(t, "Mock stream", result)
	assert.NotNil(t, stream.Usage())
	assert.Equal(t, 10, stream.Usage().TotalTokens)
}

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("Hello")
	assert.Equal(t, RoleUser, msg.Role)
	assert.Len(t, msg.Content, 1)
	assert.Equal(t, ContentTypeText, msg.Content[0].Type)
	assert.Equal(t, "Hello", msg.Content[0].Text)
}

func TestNewSystemMessage(t *testing.T) {
	msg := NewSystemMessage("You are helpful")
	assert.Equal(t, RoleSystem, msg.Role)
	assert.Equal(t, "You are helpful", msg.TextContent())
}

func TestMessage_TextContent(t *testing.T) {
	msg := Message{
		Role: RoleAssistant,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: "Hello "},
			{Type: ContentTypeText, Text: "world"},
		},
	}
	assert.Equal(t, "Hello world", msg.TextContent())
}
