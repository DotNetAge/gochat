# I got tired of rewriting Go code for every new LLM API, so I built a unified client (with native Reasoning & OAuth support)

If you are building AI features in Go right now, you probably know the pain:

Yesterday, you integrated OpenAI. Today, the team wants to try Claude 3.7. Tomorrow, you need to support `DeepSeek-R1` because it's cheaper. Next week, the security team tells you to move everything to local models using `Ollama`. And if you are dealing with enterprise gateways (like Azure or GCP), you can't even use simple API keys—you have to deal with OAuth2 and token refreshes.

Your codebase quickly becomes a mess of `if-else` statements, different message payloads, and hardcoded API keys.

I got frustrated with this fragmentation, so I built [**GoChat**](https://github.com/DotNetAge/gochat) — a modern, enterprise-ready LLM SDK for Go. 

Here is how it solves the mess:

## 1. One Unified Interface for Everything
Whether you are calling GPT-4o, Claude, DeepSeek, Qwen, or local Ollama, the business logic remains exactly the same. You just change one line in the initialization.

```go
// Switch between OpenAI, DeepSeek, or Ollama seamlessly
client, _ := deepseek.New(deepseek.Config{...})

resp, _ := client.Chat(ctx, []core.Message{
    core.NewUserMessage("Explain quantum computing in 1 sentence."),
})
```

## 2. Native Support for "Deep Thinking" (Reasoning Chains)
2026 is the year of reasoning models (o1, o3, DeepSeek-R1). Most SDKs mix the "thinking process" and the "final answer" together. GoChat has built-in interceptors that parse the reasoning chain out of the box via the Streaming API.

```go
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))

for stream.Next() {
    ev := stream.Event()
    if ev.Type == core.EventThinking { 
        fmt.Print("💭 Model is thinking: ", ev.Content) 
    } else if ev.Type == core.EventContent {
        fmt.Print("💡 Final Answer: ", ev.Content)
    }
}
```

## 3. Smart Secret Hosting (Zero Hardcoded Keys)
Stop leaking API keys on GitHub. GoChat uses smart environment variable extraction. Just pass an alias to the config:

```go
client, _ := openai.New(openai.Config{
    Config: base.Config{
        APIKey: "OPENAI_API_KEY", // The engine sniffs the env var automatically! Extremely secure.
    },
})
```

## 4. Enterprise-grade AuthManager (OAuth2 & Device Code)
If you are calling enterprise portals that require OAuth2 (like GCP Gemini or Alibaba Portals), you can't use static keys. GoChat comes with an `AuthManager` that handles the entire Device Code/OAuth flow, persists the tokens (to local files, Redis, or DB), and **silently auto-refreshes** them in the background when they expire.

```go
p := provider.NewGeminiProvider(clientID, secret, callback, port)
authMgr := core.NewAuthManager(p, "token.json")

// If expired, it auto-refreshes. If missing, it triggers the OAuth flow.
token, _ := authMgr.GetToken() 
```

## 5. Extensible via Functional Options
Need web search? `core.WithEnableSearch(true)`. Need function calling? `core.WithTools(tools...)`. Everything is handled elegantly without polluting the main `Client.Chat()` interface.

---

I built this because I wanted a clean, type-safe, and secure way to interact with the chaotic world of LLM APIs in Go. 

If you're building AI agents, CLI tools, or enterprise chat apps in Go, I'd love for you to try it out!

**GitHub repo**: [https://github.com/DotNetAge/gochat](https://github.com/DotNetAge/gochat)

Feedback, issues, and PRs are highly appreciated! Let me know what you think.
