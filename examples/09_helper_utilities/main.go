// Helper utilities for common use cases.
//
// This example demonstrates useful helper functions that make working
// with GoChat even easier for common scenarios.
package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DotNetAge/gochat/pkg/core"
)

// LoadImageAsContentBlock reads an image file and returns a ContentBlock.
// This is a convenience function for the common task of loading images.
func LoadImageAsContentBlock(imagePath string) (core.ContentBlock, error) {
	// Read the image
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return core.ContentBlock{}, fmt.Errorf("failed to read image: %w", err)
	}

	// Encode to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Detect media type from extension
	mediaType := "image/jpeg"
	ext := filepath.Ext(imagePath)
	switch ext {
	case ".png":
		mediaType = "image/png"
	case ".webp":
		mediaType = "image/webp"
	case ".gif":
		mediaType = "image/gif"
	}

	return core.ContentBlock{
		Type:      core.ContentTypeImage,
		MediaType: mediaType,
		Data:      base64Data,
		FileName:  filepath.Base(imagePath),
	}, nil
}

// LoadDocumentAsText reads a text document and returns its content.
// Useful for analyzing code, markdown, or other text files.
func LoadDocumentAsText(docPath string) (string, error) {
	data, err := os.ReadFile(docPath)
	if err != nil {
		return "", fmt.Errorf("failed to read document: %w", err)
	}
	return string(data), nil
}

// CreateMultimodalMessage creates a message with text and multiple images.
// This is a common pattern for vision tasks.
func CreateMultimodalMessage(text string, imagePaths ...string) (core.Message, error) {
	blocks := []core.ContentBlock{
		{Type: core.ContentTypeText, Text: text},
	}

	for _, path := range imagePaths {
		imageBlock, err := LoadImageAsContentBlock(path)
		if err != nil {
			return core.Message{}, err
		}
		blocks = append(blocks, imageBlock)
	}

	return core.Message{
		Role:    core.RoleUser,
		Content: blocks,
	}, nil
}

// AppendAssistantResponse is a helper to add the assistant's response to history.
// This is needed for multi-turn conversations.
func AppendAssistantResponse(messages []core.Message, response *core.Response) []core.Message {
	return append(messages, core.Message{
		Role:      core.RoleAssistant,
		Content:   []core.ContentBlock{{Type: core.ContentTypeText, Text: response.Content}},
		ToolCalls: response.ToolCalls,
	})
}

// AppendToolResult adds a tool execution result to the conversation.
func AppendToolResult(messages []core.Message, toolCallID, result string) []core.Message {
	return append(messages, core.Message{
		Role:       core.RoleTool,
		ToolCallID: toolCallID,
		Content:    []core.ContentBlock{{Type: core.ContentTypeText, Text: result}},
	})
}

// Example usage
func main() {
	// Example 1: Load an image easily
	imageBlock, err := LoadImageAsContentBlock("photo.jpg")
	if err != nil {
		fmt.Printf("Error loading image: %v\n", err)
		return
	}
	fmt.Printf("Loaded image: %s (%s)\n", imageBlock.FileName, imageBlock.MediaType)

	// Example 2: Create a multimodal message
	message, err := CreateMultimodalMessage(
		"Compare these images and tell me the differences",
		"image1.jpg",
		"image2.jpg",
	)
	if err != nil {
		fmt.Printf("Error creating message: %v\n", err)
		return
	}
	fmt.Printf("Created message with %d content blocks\n", len(message.Content))

	// Example 3: Build conversation history
	var messages []core.Message
	messages = append(messages, core.NewUserMessage("Hello"))

	// Simulate a response
	response := &core.Response{
		Content: "Hi! How can I help you?",
	}
	messages = AppendAssistantResponse(messages, response)

	fmt.Printf("Conversation has %d messages\n", len(messages))

	// Example 4: Add tool result
	messages = AppendToolResult(messages, "call_123", `{"temperature": 72, "condition": "sunny"}`)
	fmt.Printf("Added tool result, now %d messages\n", len(messages))
}
