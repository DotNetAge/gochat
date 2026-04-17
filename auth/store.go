package auth

import (
	"encoding/json"
	"os"
)

// TokenStore defines the interface for persisting and retrieving OAuth tokens.
// This allows developers to use custom storage like databases, Redis, or memory.
type TokenStore interface {
	Save(token *OAuthToken) error
	Load() (*OAuthToken, error)
}

// FileTokenStore is a simple file-based implementation of TokenStore.
type FileTokenStore struct {
	Filename string
}

// NewFileTokenStore creates a new FileTokenStore.
func NewFileTokenStore(filename string) *FileTokenStore {
	return &FileTokenStore{
		Filename: filename,
	}
}

// Save writes the token to a file as JSON.
func (s *FileTokenStore) Save(token *OAuthToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.Filename, data, 0600)
}

// Load reads the token from a file.
func (s *FileTokenStore) Load() (*OAuthToken, error) {
	data, err := os.ReadFile(s.Filename)
	if err != nil {
		return nil, err
	}

	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}
