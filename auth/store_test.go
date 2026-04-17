package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileTokenStore_Save_Load(t *testing.T) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "token-test")
	require.NoError(t, err)
	tempFilename := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempFilename)

	// 创建 FileTokenStore
	store := NewFileTokenStore(tempFilename)

	// 测试保存令牌
	token := &OAuthToken{
		Access:      "test-access-token",
		Refresh:     "test-refresh-token",
		Expires:     1234567890,
		ResourceUrl: strPtr("https://api.example.com"),
	}

	err = store.Save(token)
	require.NoError(t, err)

	// 测试加载令牌
	loadedToken, err := store.Load()
	require.NoError(t, err)

	assert.Equal(t, token.Access, loadedToken.Access)
	assert.Equal(t, token.Refresh, loadedToken.Refresh)
	assert.Equal(t, token.Expires, loadedToken.Expires)
	assert.Equal(t, token.ResourceUrl, loadedToken.ResourceUrl)
}

func TestFileTokenStore_Load_Error(t *testing.T) {
	// 测试加载不存在的文件
	store := NewFileTokenStore("non-existent-file.json")
	_, err := store.Load()
	assert.Error(t, err)
}

// strPtr returns a pointer to the string value
func strPtr(s string) *string {
	return &s
}
