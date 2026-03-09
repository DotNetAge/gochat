package core

// OAuthToken 通用 OAuth 令牌
type OAuthToken struct {
	Access              string  `json:"access"`
	Refresh             string  `json:"refresh"`
	Expires             int64   `json:"expires"`
	ResourceUrl         *string `json:"resourceUrl,omitempty"`
	NotificationMessage *string `json:"notification_message,omitempty"`
}
