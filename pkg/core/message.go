package core

// Message roles define who is speaking in a conversation.
const (
	// RoleSystem is for system instructions that guide the model's behavior.
	// System messages set the context, personality, or constraints.
	// Example: "You are a helpful coding assistant."
	RoleSystem = "system"

	// RoleUser is for messages from the human user.
	// These are the questions or prompts you send to the model.
	RoleUser = "user"

	// RoleAssistant is for messages from the AI model.
	// These are the model's responses.
	RoleAssistant = "assistant"

	// RoleTool is for tool execution results.
	// Use this when responding to a model's tool call request.
	RoleTool = "tool"
)

// ContentType identifies the type of content in a ContentBlock.
type ContentType string

const (
	// ContentTypeText is for plain text content.
	ContentTypeText ContentType = "text"

	// ContentTypeImage is for image content (base64-encoded).
	ContentTypeImage ContentType = "image"

	// ContentTypeFile is for file attachments.
	ContentTypeFile ContentType = "file"
	// ContentTypeThinking is for thinking/reasoning content from the model.
	ContentTypeThinking ContentType = "thinking"
)

// ContentBlock is one piece of content within a message.
//
// Messages can contain multiple content blocks of different types,
// enabling multimodal interactions (text + images + files).
//
// For simple text messages, use the NewUserMessage helper which creates
// a message with a single text content block.
//
// Example (text):
//
//	block := ContentBlock{
//	    Type: ContentTypeText,
//	    Text: "Hello, world!",
//	}
//
// Example (image):
//
//	imageData := base64.StdEncoding.EncodeToString(imageBytes)
//	block := ContentBlock{
//	    Type:      ContentTypeImage,
//	    MediaType: "image/png",
//	    Data:      imageData,
//	}
type ContentBlock struct {
	// Type identifies what kind of content this block contains.
	Type ContentType `json:"type"`

	// Text is the text content (only for ContentTypeText).
	Text string `json:"text,omitempty"`

	// MediaType is the MIME type (only for ContentTypeImage/ContentTypeFile).
	// Examples: "image/png", "image/jpeg", "application/pdf"
	MediaType string `json:"media_type,omitempty"`

	// Data is the base64-encoded content (only for ContentTypeImage/ContentTypeFile).
	Data string `json:"data,omitempty"`

	// FileName is the original filename (only for ContentTypeFile).
	FileName string `json:"file_name,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content,omitempty"`
	ToolCalls  []ToolCall     `json:"tool_calls,omitempty"`   // assistant requesting tool use
	ToolCallID string         `json:"tool_call_id,omitempty"` // for role=tool responses
}

// NewTextMessage creates a message with text content
func NewTextMessage(role, text string) Message {
	return Message{
		Role: role,
		Content: []ContentBlock{
			{Type: ContentTypeText, Text: text},
		},
	}
}

// NewUserMessage creates a user message with text content
func NewUserMessage(text string) Message {
	return NewTextMessage(RoleUser, text)
}

// NewSystemMessage creates a system message
func NewSystemMessage(text string) Message {
	return NewTextMessage(RoleSystem, text)
}

// TextContent returns the concatenated text of all text blocks.
// Convenience for the common single-text-block case.
func (m Message) TextContent() string {
	var s string
	for _, b := range m.Content {
		if b.Type == ContentTypeText {
			s += b.Text
		}
	}
	return s
}
