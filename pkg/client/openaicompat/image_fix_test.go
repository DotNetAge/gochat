package openaicompat

import (
	"testing"

	"github.com/DotNetAge/gochat/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestMessagesToWire_ImageURL(t *testing.T) {
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type:      core.ContentTypeImageURL,
					URL:       "https://example.com/image.png",
					MediaType: "image/png",
				},
			},
		},
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 1)

	parts, ok := wireMessages[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, parts, 1)
	assert.Equal(t, "image_url", parts[0].Type)
	assert.Equal(t, "https://example.com/image.png", parts[0].ImageURL.URL)
}

func TestMessagesToWire_ImageBase64(t *testing.T) {
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type:      core.ContentTypeImage,
					Data:      "base64imagedata",
					MediaType: "image/png",
				},
			},
		},
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 1)

	parts, ok := wireMessages[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, parts, 1)
	assert.Equal(t, "image_url", parts[0].Type)
	assert.Contains(t, parts[0].ImageURL.URL, "data:image/png;base64,")
	assert.Contains(t, parts[0].ImageURL.URL, "base64imagedata")
}

func TestMessagesToWire_MultipleImages_NoDuplicate(t *testing.T) {
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type:      core.ContentTypeImage,
					Data:      "base64data1",
					MediaType: "image/png",
				},
				{
					Type:      core.ContentTypeImage,
					Data:      "base64data2",
					MediaType: "image/jpeg",
				},
			},
		},
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 1)

	parts, ok := wireMessages[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, parts, 2)
}

func TestMessagesToWire_ImageWithURLAndData(t *testing.T) {
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type:      core.ContentTypeImage,
					URL:       "https://example.com/image.png",
					Data:      "base64data",
					MediaType: "image/png",
				},
			},
		},
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 1)

	parts, ok := wireMessages[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, parts, 1)
	assert.Equal(t, "https://example.com/image.png", parts[0].ImageURL.URL)
}

func TestMessagesToWire_TextAndImage(t *testing.T) {
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{Type: core.ContentTypeText, Text: "Look at this image"},
				{
					Type:      core.ContentTypeImage,
					Data:      "base64data",
					MediaType: "image/png",
				},
			},
		},
	}

	wireMessages := MessagesToWire(messages, "")
	assert.Len(t, wireMessages, 1)

	parts, ok := wireMessages[0].Content.([]ContentPart)
	assert.True(t, ok)
	assert.Len(t, parts, 2)
	assert.Equal(t, "text", parts[0].Type)
	assert.Equal(t, "Look at this image", parts[0].Text)
	assert.Equal(t, "image_url", parts[1].Type)
}
