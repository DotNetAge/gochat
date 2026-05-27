package core

// Response is the complete result from a Chat call.
//
// It contains the model's generated content, metadata about the request,
// token usage information, and any tool calls the model wants to make.
//
// Example:
//
//	response, err := client.Chat(ctx, messages)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Simple text response
//	fmt.Println(response.Content)
//
//	// Check token usage
//	fmt.Printf("Used %d tokens\n", response.Usage.TotalTokens)
//
//	// Handle tool calls
//	if len(response.ToolCalls) > 0 {
//	    for _, tc := range response.ToolCalls {
//	        fmt.Printf("Model wants to call: %s\n", tc.Name)
//	    }
//	}
type Response struct {
	// ID is the unique identifier for this completion.
	// Provider-specific format (e.g., "chatcmpl-abc123" for OpenAI).
	ID string `json:"id,omitempty"`

	// Model is the name of the model that generated this response.
	// May differ from the requested model if the provider substituted it.
	Model string `json:"model,omitempty"`

	// Content is the generated text content.
	// This is a convenience field that concatenates all text blocks from Message.
	// For simple text responses, this is all you need.
	Content string `json:"content"`
	// ReasoningContent is the model's thinking/reasoning process output.
	ReasoningContent string `json:"reasoning_content,omitempty"`

	// Refusal is set when the model refuses to generate content
	// (finish_reason="content_filter"). Contains the refusal explanation.
	Refusal string `json:"refusal,omitempty"`

	// Message is the full structured message from the model.
	// Use this when you need access to multimodal content or tool calls.
	Message Message `json:"message"`

	// FinishReason indicates why the model stopped generating.
	// Common values: "stop" (natural end), "length" (max tokens reached),
	// "tool_calls" (model wants to call tools), "content_filter" (filtered).
	FinishReason string `json:"finish_reason"`

	// Usage contains token consumption information.
	// Use this to track costs and monitor usage.
	Usage *Usage `json:"usage,omitempty"`

	// ToolCalls is a convenience field extracted from Message.ToolCalls.
	// Non-empty when the model wants to invoke tools.
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// PromptTokensDetails contains breakdown of prompt token consumption.
type PromptTokensDetails struct {
	// CachedTokens is the number of tokens read from the prompt cache.
	CachedTokens int `json:"cached_tokens,omitempty"`

	// AudioTokens is the number of audio tokens in the prompt.
	AudioTokens int `json:"audio_tokens,omitempty"`
}

// CompletionTokensDetails contains breakdown of completion token consumption.
type CompletionTokensDetails struct {
	// ReasoningTokens is the number of tokens used for reasoning/thinking.
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`

	// AudioTokens is the number of audio tokens in the completion.
	AudioTokens int `json:"audio_tokens,omitempty"`

	// AcceptedPredictionTokens is the number of tokens accepted from speculation.
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`

	// RejectedPredictionTokens is the number of tokens rejected from speculation.
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// Usage tracks token consumption for a request.
//
// Tokens are the basic units that LLM providers use for billing.
// Different providers may count tokens differently, but the general
// principle is: more tokens = higher cost.
//
// Example:
//
//	response, _ := client.Chat(ctx, messages)
//	if response.Usage != nil {
//	    fmt.Printf("Prompt: %d tokens\n", response.Usage.PromptTokens)
//	    fmt.Printf("Completion: %d tokens\n", response.Usage.CompletionTokens)
//	    fmt.Printf("Total: %d tokens\n", response.Usage.TotalTokens)
//	}
type Usage struct {
	// PromptTokens is the number of tokens in the input (your messages).
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the output (model's response).
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of PromptTokens and CompletionTokens.
	// This is typically what you're billed for.
	TotalTokens int `json:"total_tokens"`

	// PromptTokensDetails contains a breakdown of prompt token usage.
	// Set when the provider returns detailed token information.
	PromptTokensDetails *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`

	// CompletionTokensDetails contains a breakdown of completion token usage.
	// Set when the provider returns detailed token information.
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}
