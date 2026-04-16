package main

import (
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/core"
	"github.com/DotNetAge/gochat/provider"
)

func main() {
	fmt.Println("=== Gemini OAuth2 Authorization Code Flow Test ===")

	// 假设我们在 GCP Console 申请了以下参数，并打算在本地的 8080 端口接收回调
	clientID := "YOUR_GOOGLE_CLIENT_ID"
	clientSecret := "YOUR_GOOGLE_CLIENT_SECRET"
	callbackURL := "http://localhost:8080/oauth2/callback" // 需要在 GCP Console 登记此 URI
	listenAddr := ":8080"

	// 初始化 Provider
	p := provider.NewGeminiProvider(clientID, clientSecret, callbackURL, listenAddr)

	// 初始化 AuthManager
	authMgr := core.NewAuthManager(p, "gemini_token.json")

	// GetToken 会触发 Authenticate() -> 启动本地 Web 服务器 -> 挂起等待浏览器回调
	fmt.Println("[Info] Attempting to get Gemini token (will start local server if no token exists)...")
	token, err := authMgr.GetToken()
	if err != nil {
		fmt.Printf("[Info] Local token not found or invalid. Triggering login...\n")
		if err := authMgr.Login(); err != nil {
			log.Fatalf("[Error] Failed to login via standard OAuth2 flow: %v", err)
		}

		token, err = authMgr.GetToken()
		if err != nil {
			log.Fatalf("[Error] Failed to get token after login: %v", err)
		}
	}

	fmt.Printf("\n[Success] OAuth Flow Complete! Token saved to gemini_token.json\n")
	fmt.Printf("Access Token: %s...\n", token.Access[:10])
	fmt.Printf("Refresh Token: %s...\n", token.Refresh[:10])
}
