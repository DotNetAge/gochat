package core

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// ProcessAttachments injects provided attachments into the message list.
func ProcessAttachments(messages []Message, attachments []Attachment) []Message {
	if len(attachments) == 0 || len(messages) == 0 {
		return messages
	}

	var blocks []ContentBlock
	for _, att := range attachments {
		if att.IsTextBased {
			text := fmt.Sprintf("\n--- Attachment: %s ---\n%s\n--- End ---\n", att.Name, string(att.Data))
			blocks = append(blocks, ContentBlock{Type: ContentTypeText, Text: text})
		} else if strings.HasPrefix(att.MediaType, "image/") {
			if att.URL != "" {
				blocks = append(blocks, ContentBlock{Type: ContentTypeImageURL, URL: att.URL})
			} else if len(att.Data) > 0 {
				b64 := base64.StdEncoding.EncodeToString(att.Data)
				blocks = append(blocks, ContentBlock{Type: ContentTypeImage, MediaType: att.MediaType, Data: b64})
			}
		}
	}

	if len(blocks) == 0 {
		return messages
	}

	lastUserIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == RoleUser {
			lastUserIdx = i
			break
		}
	}

	result := make([]Message, len(messages))
	copy(result, messages)

	if lastUserIdx >= 0 {
		result[lastUserIdx].Content = append(result[lastUserIdx].Content, blocks...)
	} else {
		result = append(result, Message{Role: RoleUser, Content: blocks})
	}

	return result
}
