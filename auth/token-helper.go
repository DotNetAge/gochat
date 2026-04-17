package auth

import (
	"encoding/json"
	"os"
	"time"
)

// TokenHelper 令牌辅助工具
type TokenHelper struct{}

// SaveToken 保存令牌到文件
func (h *TokenHelper) SaveToken(token *OAuthToken, filename string) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0600)
}

// LoadToken 从文件加载令牌
func (h *TokenHelper) LoadToken(filename string) (*OAuthToken, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// IsTokenExpired 检查令牌是否过期
func (h *TokenHelper) IsTokenExpired(token *OAuthToken) bool {
	return time.Now().UnixMilli() > token.Expires
}
