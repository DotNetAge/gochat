package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// Qwen 创建 Qwen (通义千问) 的 AuthManager
// tokenFile 可选参数：当提供时启用文件存储，否则使用内存存储
func Qwen(tokenFile ...string) *AuthManager {
	provider := NewQwenProvider()
	return newAuthManagerWithOptionalStore(provider, "qwen", tokenFile...)
}

// GeminiWithConfig 使用完整配置创建 Gemini 的 AuthManager
func Gemini(clientID, clientSecret, callbackURL, listenAddr string, tokenFile ...string) *AuthManager {
	provider := NewGeminiProvider(clientID, clientSecret, callbackURL, listenAddr)
	return newAuthManagerWithOptionalStore(provider, "gemini", tokenFile...)
}

// MiniMax 创建 MiniMax 的 AuthManager
// region: "cn" 表示国内版，其他值表示国际版
// tokenFile 可选参数：当提供时启用文件存储，否则使用内存存储
func MiniMax(region string, tokenFile ...string) *AuthManager {
	provider := NewMiniMaxProvider(region)
	return newAuthManagerWithOptionalStore(provider, "minimax", tokenFile...)
}

// newAuthManagerWithOptionalStore 创建 AuthManager，根据 tokenFile 参数决定是否启用文件存储
func newAuthManagerWithOptionalStore(provider interface {
	GetProviderName() string
	Authenticate() (*OAuthToken, error)
	RefreshToken(refreshToken string) (*OAuthToken, error)
}, defaultTokenFileName string, tokenFile ...string) *AuthManager {
	// 当 tokenFile[0] 有值时启用 TokenStore
	if len(tokenFile) > 0 && tokenFile[0] != "" {
		return NewAuthManager(provider, tokenFile[0])
	}

	// 否则使用默认的 token 文件路径
	// 在用户主目录下创建 .gochat 目录存储 token
	defaultPath := getDefaultTokenPath(defaultTokenFileName)
	if err := ensureTokenDir(defaultPath); err == nil {
		return NewAuthManager(provider, defaultPath)
	}

	// 如果无法创建默认目录，则返回不带存储的 AuthManager
	return &AuthManager{
		provider: provider,
	}
}

// getDefaultTokenPath 获取默认的 token 文件路径
func getDefaultTokenPath(providerName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".gochat", providerName+"_token.json")
}

// ensureTokenDir 确保 token 文件所在的目录存在
func ensureTokenDir(tokenPath string) error {
	dir := filepath.Dir(tokenPath)
	return os.MkdirAll(dir, 0700)
}

// GetDefaultTokenPath 导出默认 token 路径供外部使用
func GetDefaultTokenPath(providerName string) (string, error) {
	path := getDefaultTokenPath(providerName)
	if err := ensureTokenDir(path); err != nil {
		return "", fmt.Errorf("failed to create token directory: %w", err)
	}
	return path, nil
}
