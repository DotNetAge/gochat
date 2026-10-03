package auth

import (
	"encoding/json"
	"errors"
)

// ErrTokenNotFound 凭据库中不存在该键的令牌（未授权或已被清除）
var ErrTokenNotFound = errors.New("token not found in credential backend")

// CredentialsBackend 凭据库后端的最小键值视图——由宿主环境注入实现
// （mindx 的 CredentialStore：darwin = Keychain，其他 = 机器指纹加密文件）。
// gochat 不感知具体后端，保持依赖方向宿主 → gochat。
type CredentialsBackend interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

// KeyringTokenStore TokenStore 的凭据库后端实现——OAuth 令牌整体 JSON
// 序列化后存入 CredentialsBackend。存储键由构造参数携带
// （技能凭据约定 <skill_name>:<env_name>），Save/Load 的"单令牌"语义
// 由键唯一确定，技能间不共享凭据。
type KeyringTokenStore struct {
	backend CredentialsBackend
	key     string
}

// NewKeyringTokenStore 创建凭据库令牌存储。backend 为宿主凭据库，
// key 为存储键（如 <skill_name>:<env_name>）。
func NewKeyringTokenStore(backend CredentialsBackend, key string) *KeyringTokenStore {
	return &KeyringTokenStore{backend: backend, key: key}
}

// Save 将令牌整体序列化为 JSON 写入凭据库
func (s *KeyringTokenStore) Save(token *OAuthToken) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return s.backend.Set(s.key, string(data))
}

// Load 从凭据库读取令牌；键不存在或值为空时返回错误
// （对齐 FileTokenStore 语义：LoadToken 需区分"未授权"与"授权后读回"）。
func (s *KeyringTokenStore) Load() (*OAuthToken, error) {
	value, err := s.backend.Get(s.key)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, ErrTokenNotFound
	}

	var token OAuthToken
	if err := json.Unmarshal([]byte(value), &token); err != nil {
		return nil, err
	}
	return &token, nil
}
