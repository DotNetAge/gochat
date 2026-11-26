package core

// Message roles define who is speaking in a conversation.
const (
	// RoleSystem is for system instructions that guide the model's behavior.
	// System messages set the context, personality, or constraints.
	// Example: "You are a helpful coding assistant."
	RoleSystem = "system"

	// RoleDeveloper is for developer instructions (OpenAI o1/o3 specific).
	// It's a modern replacement for RoleSystem in newer OpenAI models.
	RoleDeveloper = "developer"

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
	ContentTypeImage    ContentType = "image"
	ContentTypeImageURL ContentType = "image_url"

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
	URL  string `json:"url,omitempty"`

	// FileName is the original filename (only for ContentTypeFile).
	FileName string `json:"file_name,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role             string         `json:"role"`
	Content          []ContentBlock `json:"content,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`    // assistant requesting tool use
	ToolCallID       string         `json:"tool_call_id,omitempty"`  // for role=tool responses
	ReasoningContent string         `json:"reasoning_content,omitempty"` // thinking/reasoning stream (DeepSeek-R1 etc.)

	// Origin 标记这条消息的真实来源：空值表示用户或模型的正常发言，
	// "tool_vision" 表示工具读取的图片以视觉附件形式注入，"task_board_nudge"
	// 表示宿主机追加的任务板提醒。
	//
	// 这类消息在协议上只能以 user 角色出现在对话中间（多模态内容必须挂 user
	// 消息，且 OpenAI 兼容层只允许 messages[0] 为 system）。来源标记让宿主、
	// 会话存储与前端能区分「谁在说话」，而不必依赖正文里的固定前缀。
	// json:"-" 使其永不进入线上请求体：各 provider 转换层按字段白名单投影。
	Origin string `json:"-"`
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

// NewDeveloperMessage creates a developer message (OpenAI specific)
func NewDeveloperMessage(text string) Message {
	return NewTextMessage(RoleDeveloper, text)
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
