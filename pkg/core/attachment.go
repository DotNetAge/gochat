package core

type Attachment struct {
	Name        string
	MediaType   string
	Data        []byte
	URL         string
	IsTextBased bool
}

func NewAttachment(name, mediaType string, data []byte, isText bool) Attachment {
	return Attachment{Name: name, MediaType: mediaType, Data: data, IsTextBased: isText}
}
func NewImageAttachment(name, mediaType string, data []byte) Attachment {
	return Attachment{Name: name, MediaType: mediaType, Data: data, IsTextBased: false}
}
