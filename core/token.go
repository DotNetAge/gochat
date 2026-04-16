package core

// OAuthToken represents an OAuth token received from an AI provider.
// It contains the credentials needed to authenticate API requests,
// along with expiration information for token lifecycle management.
type OAuthToken struct {
	// Access is the bearer token used to authenticate API requests.
	// Include this as a Bearer token in the Authorization header.
	Access string `json:"access"`

	// Refresh is the token used to obtain a new access token when it expires.
	// Store this securely for refreshing expired tokens.
	Refresh string `json:"refresh"`

	// Expires is the Unix timestamp in milliseconds when this token expires.
	// Use this to determine when to refresh the token proactively.
	Expires int64 `json:"expires"`

	// ResourceUrl is an optional endpoint for accessing the resource.
	// This may be provided by some OAuth providers.
	ResourceUrl *string `json:"resourceUrl,omitempty"`

	// NotificationMessage may contain a message from the OAuth provider.
	// This could include warnings or important notices.
	NotificationMessage *string `json:"notification_message,omitempty"`
}
