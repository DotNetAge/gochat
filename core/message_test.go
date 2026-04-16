package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageCreation(t *testing.T) {
	// Test NewUserMessage
	userMsg := NewUserMessage("Hello, world!")
	assert.Equal(t, RoleUser, userMsg.Role)
	require.Len(t, userMsg.Content, 1)
	assert.Equal(t, ContentTypeText, userMsg.Content[0].Type)
	assert.Equal(t, "Hello, world!", userMsg.Content[0].Text)

	// Test NewSystemMessage
	systemMsg := NewSystemMessage("You are a helpful assistant")
	assert.Equal(t, RoleSystem, systemMsg.Role)
	require.Len(t, systemMsg.Content, 1)
	assert.Equal(t, ContentTypeText, systemMsg.Content[0].Type)
	assert.Equal(t, "You are a helpful assistant", systemMsg.Content[0].Text)

	// Test NewTextMessage for assistant role
	assistantMsg := NewTextMessage(RoleAssistant, "I'm here to help!")
	assert.Equal(t, RoleAssistant, assistantMsg.Role)
	require.Len(t, assistantMsg.Content, 1)
	assert.Equal(t, ContentTypeText, assistantMsg.Content[0].Type)
	assert.Equal(t, "I'm here to help!", assistantMsg.Content[0].Text)

	// Test tool message creation
	toolMsg := Message{
		Role:       RoleTool,
		ToolCallID: "tool-123",
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: "Tool result"},
		},
	}
	assert.Equal(t, RoleTool, toolMsg.Role)
	assert.Equal(t, "tool-123", toolMsg.ToolCallID)
	require.Len(t, toolMsg.Content, 1)
	assert.Equal(t, ContentTypeText, toolMsg.Content[0].Type)
	assert.Equal(t, "Tool result", toolMsg.Content[0].Text)
}

func TestMessageWithImage(t *testing.T) {
	// Test message with image content
	msg := NewUserMessage("What's in this image?")
	msg.Content = append(msg.Content, ContentBlock{
		Type:      ContentTypeImage,
		MediaType: "image/png",
		Data:      "base64-image-data",
	})

	assert.Equal(t, RoleUser, msg.Role)
	require.Len(t, msg.Content, 2)
	assert.Equal(t, ContentTypeText, msg.Content[0].Type)
	assert.Equal(t, ContentTypeImage, msg.Content[1].Type)
	assert.Equal(t, "base64-image-data", msg.Content[1].Data)
}

func TestMessageValidation(t *testing.T) {
	// Test empty message
	msg := Message{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: ""},
		},
	}
	// Should be valid even with empty text
	assert.Equal(t, RoleUser, msg.Role)
	assert.Equal(t, "", msg.Content[0].Text)

	// Test message with thinking content
	msg = Message{
		Role: RoleAssistant,
		Content: []ContentBlock{
			{Type: ContentTypeThinking, Text: "Let me think about this..."},
		},
	}
	assert.Equal(t, ContentTypeThinking, msg.Content[0].Type)
	assert.Equal(t, "Let me think about this...", msg.Content[0].Text)
}

func TestContentBlockTypes(t *testing.T) {
	// Test all content type constants
	assert.Equal(t, ContentType("text"), ContentTypeText)
	assert.Equal(t, ContentType("image"), ContentTypeImage)
	assert.Equal(t, ContentType("file"), ContentTypeFile)
	assert.Equal(t, ContentType("thinking"), ContentTypeThinking)
}

func TestMessageRoleConstants(t *testing.T) {
	// Test all role constants
	assert.Equal(t, "system", RoleSystem)
	assert.Equal(t, "user", RoleUser)
	assert.Equal(t, "assistant", RoleAssistant)
	assert.Equal(t, "tool", RoleTool)
}

func TestMessageWithMultipleContentBlocks(t *testing.T) {
	// Test message with multiple content types
	msg := Message{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: "Look at this image:"},
			{Type: ContentTypeImage, MediaType: "image/jpeg", Data: "img1"},
			{Type: ContentTypeText, Text: "And this file:"},
			{Type: ContentTypeFile, MediaType: "application/pdf", FileName: "doc.pdf", Data: "file-data"},
		},
	}

	assert.Equal(t, RoleUser, msg.Role)
	require.Len(t, msg.Content, 4)
	assert.Equal(t, ContentTypeText, msg.Content[0].Type)
	assert.Equal(t, ContentTypeImage, msg.Content[1].Type)
	assert.Equal(t, ContentTypeText, msg.Content[2].Type)
	assert.Equal(t, ContentTypeFile, msg.Content[3].Type)
	assert.Equal(t, "img1", msg.Content[1].Data)
	assert.Equal(t, "doc.pdf", msg.Content[3].FileName)
}

func TestTextContentMethod(t *testing.T) {
	// Test TextContent method
	msg := Message{
		Role: RoleUser,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: "Hello "},
			{Type: ContentTypeImage, MediaType: "image/png", Data: "img"},
			{Type: ContentTypeText, Text: "world!"},
		},
	}

	text := msg.TextContent()
	assert.Equal(t, "Hello world!", text)
}
