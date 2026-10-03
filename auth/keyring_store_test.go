package auth

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoryBackend 内存凭据后端（模拟宿主 CredentialStore）
type memoryBackend struct {
	values map[string]string
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{values: make(map[string]string)}
}

func (b *memoryBackend) Get(key string) (string, error) {
	v, ok := b.values[key]
	if !ok {
		// 对齐 mindx encryptedFileStore 语义：缺键返回空值而非错误
		return "", nil
	}
	return v, nil
}

func (b *memoryBackend) Set(key, value string) error {
	b.values[key] = value
	return nil
}

// TestKeyringTokenStore_SaveLoad 令牌整体 JSON 存取回路
func TestKeyringTokenStore_SaveLoad(t *testing.T) {
	backend := newMemoryBackend()
	store := NewKeyringTokenStore(backend, "my-skill:GEMINI_TOKEN")

	require.NoError(t, store.Save(&OAuthToken{
		Access:  "access-1",
		Refresh: "refresh-1",
		Expires: 1234567890,
	}))

	token, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "access-1", token.Access)
	assert.Equal(t, "refresh-1", token.Refresh)
	assert.EqualValues(t, 1234567890, token.Expires)

	// 存储值是完整 JSON（access 字段可见）
	assert.Contains(t, backend.values["my-skill:GEMINI_TOKEN"], `"access":"access-1"`)
}

// TestKeyringTokenStore_KeyIsolation 不同键互不可见（技能间不共享凭据）
func TestKeyringTokenStore_KeyIsolation(t *testing.T) {
	backend := newMemoryBackend()
	storeA := NewKeyringTokenStore(backend, "skill-a:TOKEN")
	storeB := NewKeyringTokenStore(backend, "skill-b:TOKEN")

	require.NoError(t, storeA.Save(&OAuthToken{Access: "a-token"}))

	_, err := storeB.Load()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTokenNotFound))
}

// TestKeyringTokenStore_LoadMissing 未授权时返回 ErrTokenNotFound
func TestKeyringTokenStore_LoadMissing(t *testing.T) {
	store := NewKeyringTokenStore(newMemoryBackend(), "skill:TOKEN")
	_, err := store.Load()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrTokenNotFound))
}

// TestKeyringTokenStore_WithAuthManager 与 AuthManager 组合（自动刷新 + 持久化）
func TestKeyringTokenStore_WithAuthManager(t *testing.T) {
	backend := newMemoryBackend()
	store := NewKeyringTokenStore(backend, "skill:TOKEN")

	token := &OAuthToken{
		Access:  "stored-access",
		Refresh: "stored-refresh",
		Expires: 4102444800000, // 远期
	}
	require.NoError(t, store.Save(token))

	manager := NewAuthManagerWithStore(nil, store)
	require.NoError(t, manager.LoadToken())

	got, err := manager.GetToken()
	require.NoError(t, err)
	assert.Equal(t, "stored-access", got.Access)
}
