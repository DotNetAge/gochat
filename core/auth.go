package core

import (
	"fmt"
	"sync"
	"time"
)

// defaultTokenExpiryBuffer is the default buffer time before token expiration
// to trigger a refresh. If a token expires within this duration, it will
// be refreshed proactively.
const defaultTokenExpiryBuffer = 60 * time.Second

// AuthManager handles authentication and token lifecycle management for
// AI provider clients. It supports automatic token refresh and persistent
// token storage.
//
// AuthManager is safe for concurrent use. Multiple goroutines can call
// GetToken simultaneously; proper locking ensures tokens are refreshed
// only when necessary.
//
// Example:
//
//	manager := core.NewAuthManager(provider, "token.json")
//	if err := manager.Login(); err != nil {
//	    log.Fatal(err)
//	}
//	token, err := manager.GetToken()
//	// use token for API requests
type AuthManager struct {
	provider interface {
		GetProviderName() string
		Authenticate() (*OAuthToken, error)
		RefreshToken(refreshToken string) (*OAuthToken, error)
	}
	token        *OAuthToken
	store        TokenStore
	mu           sync.RWMutex
	expiryBuffer time.Duration
}

// NewAuthManager creates a new AuthManager with file-based token storage.
// The filename specifies where tokens should be persisted. If the file
// exists, tokens will be loaded from it on first access.
//
// Parameters:
//   - provider: The authentication provider implementing GetProviderName,
//     Authenticate, and RefreshToken
//   - filename: Path to the file where tokens are stored
//
// Returns a configured AuthManager
func NewAuthManager(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, filename string) *AuthManager {
	return &AuthManager{
		provider:     provider,
		store:        NewFileTokenStore(filename),
		expiryBuffer: defaultTokenExpiryBuffer,
	}
}

// NewAuthManagerWithStore creates a new AuthManager with custom token storage.
// Use this when you need to store tokens in a different location or format.
//
// Parameters:
//   - provider: The authentication provider
//   - store: Custom implementation of TokenStore
//
// Returns a configured AuthManager
func NewAuthManagerWithStore(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, store TokenStore) *AuthManager {
	return &AuthManager{
		provider:     provider,
		store:        store,
		expiryBuffer: defaultTokenExpiryBuffer,
	}
}

// Login authenticates the user and stores the resulting token.
// This method should be called once when the user first logs in.
// The token is stored persistently and will be reused on subsequent
// program starts via LoadToken.
//
// Returns an error if authentication fails
func (m *AuthManager) Login() error {
	token, err := m.provider.Authenticate()
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.token = token
	m.mu.Unlock()

	if m.store != nil {
		return m.store.Save(token)
	}
	return nil
}

// LoadToken loads a previously stored token from the token store.
// This is used to restore session state without requiring re-authentication.
// If no token store is configured or the store is empty, an error is returned.
//
// Returns an error if loading fails (file not found, corrupted data, etc.)
func (m *AuthManager) LoadToken() error {
	if m.store == nil {
		return fmt.Errorf("no token store configured")
	}

	token, err := m.store.Load()
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.token = token
	m.mu.Unlock()

	return nil
}

// isTokenExpired checks if a token has expired or will expire soon.
// The expiryBuffer is subtracted from the actual expiration time to
// trigger proactive refresh before the token actually expires.
//
// Parameters:
//   - token: The token to check
//
// Returns true if the token is expired or will expire within the buffer time
func (m *AuthManager) isTokenExpired(token *OAuthToken) bool {
	return time.Now().UnixMilli() > (token.Expires - m.expiryBuffer.Milliseconds())
}

// GetToken returns a valid authentication token, refreshing it if necessary.
// This is the main method used by API clients to obtain valid credentials.
//
// The method first checks if a token exists. If not, it attempts to load
// from the store. If the token is expired or will expire soon, it attempts
// to refresh the token using the provider's RefreshToken method.
//
// Returns a valid OAuthToken and nil on success.
// Returns nil and an error if no token is available, loading fails,
// or refresh fails.
func (m *AuthManager) GetToken() (*OAuthToken, error) {
	m.mu.RLock()
	token := m.token
	m.mu.RUnlock()

	if token == nil {
		if err := m.LoadToken(); err != nil {
			return nil, err
		}
		m.mu.RLock()
		token = m.token
		m.mu.RUnlock()
	}

	if token == nil {
		return nil, fmt.Errorf("no token available, please login first")
	}

	if m.isTokenExpired(token) {
		m.mu.Lock()
		token = m.token
		if m.isTokenExpired(token) {
			newToken, err := m.provider.RefreshToken(token.Refresh)
			if err != nil {
				m.mu.Unlock()
				return nil, err
			}
			m.token = newToken
			token = newToken
			m.mu.Unlock()

			if m.store != nil {
				if err := m.store.Save(newToken); err != nil {
					return nil, err
				}
			}
			return token, nil
		}
		m.mu.Unlock()
	}

	return token, nil
}

// SetExpiryBuffer changes the buffer time used to determine when to
// proactively refresh tokens. The default is 60 seconds.
//
// A larger buffer gives more time for network delays but increases
// the frequency of token refreshes. A smaller buffer may cause
// requests to fail if refresh takes too long.
//
// Parameters:
//   - buffer: The duration before expiration to trigger refresh
func (m *AuthManager) SetExpiryBuffer(buffer time.Duration) {
	m.mu.Lock()
	m.expiryBuffer = buffer
	m.mu.Unlock()
}
