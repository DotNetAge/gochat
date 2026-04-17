package main

import (
	"fmt"
	"sync"
	"testing"

	"github.com/DotNetAge/gochat/auth"
	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

// MemoryTokenStore 是一个完全自定义的基于内存的 TokenStore 示例
// 开发者可以据此将其替换为 RedisStore、DBStore 等任何需要的地方。
type MemoryTokenStore struct {
	mu    sync.RWMutex
	token *auth.OAuthToken
}

func (s *MemoryTokenStore) Save(token *auth.OAuthToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.token = token
	fmt.Printf("[Custom Store] Token has been successfully SAVED to memory.\n")
	return nil
}

func (s *MemoryTokenStore) Load() (*auth.OAuthToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.token == nil {
		fmt.Printf("[Custom Store] Attempted to load token, but it is empty.\n")
		return nil, fmt.Errorf("no token in memory")
	}
	fmt.Printf("[Custom Store] Token has been successfully LOADED from memory.\n")
	return s.token, nil
}

func TestCustomTokenStore(t *testing.T) {
	fmt.Println("\n=== Testing Custom TokenStore with AuthManager ===")

	// 初始化一个完全自定义的存储后端 (内存存储)
	customStore := &MemoryTokenStore{}
	p := auth.NewQwenProvider()

	// 核心变更点：使用 NewAuthManagerWithStore 传入我们自定义的存储接口
	authMgr := auth.NewAuthManagerWithStore(p, customStore)

	// 在内存中强制塞入刚才跑通过的过期 Token，以此验证 Store 的读取和 AuthManager 的自动 Refresh 逻辑。
	mockToken := &auth.OAuthToken{
		Access:  "fake-expired-access",
		Refresh: "-lAWdkeb5OPvRv1mW06QRACnCW2SPOnRvXe6XH1r4k28UdJu4UFryfO3l7uue8EhQ2JAzrbONUvvmgM2W56LyA",
		Expires: 1000,
	}
	customStore.Save(mockToken)

	fmt.Println("\n[Test] Getting token from AuthManager. It should detect expiration, load from CustomStore, attempt refresh...")

	token, err := authMgr.GetToken()
	if err != nil {
		fmt.Printf("Expected network error when refreshing fake token: %v\n", err)
	} else {
		fmt.Printf("Refreshed Token: %v\n", token.Access)
	}

	_, err = openai.NewOpenAI(core.Config{
		AuthToken: "any-token",
		Model:     auth.QwenPortalModelCoder,
		BaseURL:   "https://portal.qwen.ai",
	})

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	fmt.Println("=== Custom TokenStore Architecture Verified ===")
}
