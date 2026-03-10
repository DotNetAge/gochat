package core

import (
	"context"
)

// Client is the primary interface for LLM interactions.
// All providers implement this interface.
//
// Example usage:
//
//	messages := []core.Message{
//	    core.NewUserMessage("Hello, who are you?"),
//	}
//	response, err := client.Chat(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(response.Content)
//
// With options:
//
//	response, err := client.Chat(ctx, messages,
//	    core.WithTemperature(0.8),
//	    core.WithMaxTokens(1000),
//	)
type Client interface {
	// Chat sends messages and returns a complete response
	//
	// Parameters:
	// - ctx: Context for cancellation and timeout
	// - messages: Conversation messages
	// - opts: Optional parameters (temperature, max tokens, tools, etc.)
	//
	// Returns:
	// - *Response: Complete response with content, usage, and tool calls
	// - error: Error if the request fails
	Chat(ctx context.Context, messages []Message, opts ...Option) (*Response, error)

	// ChatStream sends messages and returns a stream of events
	//
	// Parameters:
	// - ctx: Context for cancellation and timeout
	// - messages: Conversation messages
	// - opts: Optional parameters (temperature, max tokens, tools, etc.)
	//
	// Returns:
	// - *Stream: Stream of events (content, usage, errors)
	// - error: Error if the request fails to start
	ChatStream(ctx context.Context, messages []Message, opts ...Option) (*Stream, error)
}
