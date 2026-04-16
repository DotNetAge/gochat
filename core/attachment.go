package core

// Attachment represents a file or media item that can be attached to a message.
// Attachments allow sending images, documents, or other files along with
// text prompts to models that support multimodal inputs.
//
// Example:
//
//	imageData, _ := os.ReadFile("photo.jpg")
//	attachment := core.NewImageAttachment("photo.jpg", "image/jpeg", imageData)
//	response, err := client.Chat(ctx, messages, core.WithAttachments(attachment))
type Attachment struct {
	// Name is the filename or identifier for this attachment.
	Name string

	// MediaType is the MIME type of the content.
	// Examples: "image/png", "image/jpeg", "application/pdf", "text/plain"
	MediaType string

	// Data is the raw content bytes.
	// For images, this should be the raw image data (not base64 encoded in the struct itself).
	Data []byte

	// URL is an alternative to Data for referring to remote resources.
	// If URL is set, Data should be empty.
	URL string

	// IsTextBased indicates whether this is a text format (true)
	// or binary format (false). Text-based attachments may be handled
	// differently by some providers.
	IsTextBased bool
}

// NewAttachment creates a new attachment with the specified properties.
//
// Parameters:
//   - name: The filename or identifier
//   - mediaType: The MIME type (e.g., "image/png")
//   - data: The raw content bytes
//   - isText: Whether this is a text-based format
//
// Returns a configured Attachment
func NewAttachment(name, mediaType string, data []byte, isText bool) Attachment {
	return Attachment{Name: name, MediaType: mediaType, Data: data, IsTextBased: isText}
}

// NewImageAttachment creates a new attachment for image content.
// This is a convenience function that sets IsTextBased to false.
//
// Parameters:
//   - name: The filename
//   - mediaType: The MIME type (e.g., "image/jpeg", "image/png")
//   - data: The raw image bytes
//
// Returns a configured Attachment for images
func NewImageAttachment(name, mediaType string, data []byte) Attachment {
	return Attachment{Name: name, MediaType: mediaType, Data: data, IsTextBased: false}
}
