package openaicompat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/DotNetAge/gochat/pkg/core"
)

// MessagesToWire converts core.Message slice to OpenAI wire format
func MessagesToWire(messages []core.Message, systemPrompt string) []Message {
	var wireMessages []Message

	// Add system prompt if provided
	if systemPrompt != "" {
		wireMessages = append(wireMessages, Message{
			Role:    core.RoleSystem,
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		wireMsg := Message{
			Role:       msg.Role,
			ToolCallID: msg.ToolCallID,
		}

		// Convert content blocks
		if len(msg.Content) == 1 && msg.Content[0].Type == core.ContentTypeText {
			// Simple text content
			wireMsg.Content = msg.Content[0].Text
		} else if len(msg.Content) > 0 {
			// Multimodal content
			var parts []ContentPart
			for _, block := range msg.Content {
				switch block.Type {
				case core.ContentTypeText:
					parts = append(parts, ContentPart{
						Type: "text",
						Text: block.Text,
					})
				case core.ContentTypeImage:
				case core.ContentTypeImageURL:
					parts = append(parts, ContentPart{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: block.URL,
						},
					})
					// Convert to data URL format
					dataURL := fmt.Sprintf("data:%s;base64,%s", block.MediaType, block.Data)
					parts = append(parts, ContentPart{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: dataURL,
						},
					})
				}
			}
			wireMsg.Content = parts
		}

		// Convert tool calls
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				wireMsg.ToolCalls = append(wireMsg.ToolCalls, ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				})
			}
		}

		wireMessages = append(wireMessages, wireMsg)
	}

	return wireMessages
}

// ToolsToWire converts core.Tool slice to OpenAI wire format
func ToolsToWire(tools []core.Tool) []Tool {
	var wireTools []Tool
	for _, t := range tools {
		wireTools = append(wireTools, Tool{
			Type: "function",
			Function: Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return wireTools
}

// ResponseFromWire converts OpenAI wire response to core.Response
func ResponseFromWire(resp *ChatCompletionResponse) *core.Response {
	if resp == nil {
		return &core.Response{}
	}

	if len(resp.Choices) == 0 {
		return &core.Response{
			ID:    resp.ID,
			Model: resp.Model,
			Usage: &core.Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]
	var content string
	var contentBlocks []core.ContentBlock
	var toolCalls []core.ToolCall

	// Extract content
	switch v := choice.Message.Content.(type) {
	case string:
		content = v
		contentBlocks = []core.ContentBlock{{Type: core.ContentTypeText, Text: v}}
	case []interface{}:
		// Multimodal content (not common in responses, but handle it)
		for _, part := range v {
			if partMap, ok := part.(map[string]interface{}); ok {
				if partMap["type"] == "text" {
					if text, ok := partMap["text"].(string); ok {
						content += text
						contentBlocks = append(contentBlocks, core.ContentBlock{
							Type: core.ContentTypeText,
							Text: text,
						})
					}
				}
			}
		}
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, core.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return &core.Response{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		ReasoningContent: choice.Message.ReasoningContent,
		FinishReason: choice.FinishReason,
		Message: core.Message{
			Role:      choice.Message.Role,
			Content:   contentBlocks,
			ToolCalls: toolCalls,
		},
		ToolCalls: toolCalls,
		Usage: &core.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
}

// ParseSSEStream parses Server-Sent Events stream and returns a channel of StreamChunks
func ParseSSEStream(reader io.Reader) <-chan StreamChunk {
	ch := make(chan StreamChunk, 10)

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var chunk StreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				// Skip malformed chunks
				continue
			}

			ch <- chunk
		}
	}()

	return ch
}

// StreamChunkToEvent converts a StreamChunk to core.StreamEvent
func StreamChunkToEvent(chunk StreamChunk) core.StreamEvent {
	if len(chunk.Choices) == 0 {
		return core.StreamEvent{Type: core.EventContent}
	}

	choice := chunk.Choices[0]

	if choice.FinishReason != "" {
		return core.StreamEvent{
			Type: core.EventDone,
		}
	}

	if choice.Delta != nil {
		if choice.Delta.ReasoningContent != "" {
			return core.StreamEvent{
				Type:    core.EventThinking,
				Content: choice.Delta.ReasoningContent,
			}
		}
		if choice.Delta.Content != "" {
			return core.StreamEvent{
				Type:    core.EventContent,
				Content: choice.Delta.Content,
			}
		}
	}

	return core.StreamEvent{Type: core.EventContent}
}
