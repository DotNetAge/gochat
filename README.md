# GoChat

[**English**](README.md) | [简体中文](README_zh.md)

GoChat is a **modern, enterprise-ready Go client SDK for Large Language Models (LLMs)**. It provides an exceptionally elegant and type-safe unified interface that completely smooths out the chaotic API differences between OpenAI, Anthropic (Claude), DeepSeek, Qwen, Ollama, and other major cloud providers or local models.

Whether you need basic chat, **Deep Thinking (Reasoning)**, **Built-in Web Search**, **Multimodal (Vision/Documents)**, or even enterprise-level **OAuth 2.0 / Portal Gateway Authentication Persistence**, you only need to master one set of interfaces to roam freely.

---

## ✨ Killer Features

- **🔌 Seamless Model Switching**: Change one line of initialization code to switch your application effortlessly between GPT-4o, Claude 3.7, DeepSeek-R1, and Qwen-Max, without even touching the `Messages` struct.
- **🧠 Native Deep Thinking Support**: Built-in interceptors perfectly compatible with the reasoning chains of DeepSeek-R1, Claude 3.7, and OpenAI o1/o3. The streaming API beautifully separates the model's "internal thought process" from its final answer.
- **🌐 Turn on Web Search at Will**: Native support for models with external web retrieval capabilities (like Qwen). A single line of code `core.WithEnableSearch(true)` unlocks real-time superpowers.
- **🛡️ Smart Secret Hosting**: Say goodbye to hardcoded secret leaks! Pass in an alias key name (e.g., `DASHSCOPE_API_KEY`), and the engine automatically sniffs and extracts system environment variables for secure authentication.
- **🏢 Enterprise OAuth2 Persistence**: The powerful built-in `AuthManager` is designed for accessing enterprise gateways (like Qwen Portal or GCP Gemini). It fully automates the Device Code Flow / OAuth2 authorization, persistent Token storage (supports custom Stores like Memory/File/Database), and silent, seamless auto-refreshing.
- **🛠️ Complete Agent Infrastructure**: Multi-turn conversation context management, structured Tool/Function Calling, document reading & extraction, and multimodal image recognition—all out of the box.

---

## 📦 Installation

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 Quick Start (Everything you need to know)

### 1. Basic Chat (with Smart Secret Hosting)

Set up your environment variables, e.g., `export OPENAI_API_KEY="sk-..."`.

```go
package main

import (
	"context"
	"fmt"
	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/gochat/pkg/core"
)

func main() {
	// The engine detects "OPENAI_API_KEY" as an env var name and safely retrieves the real key from the system!
	client, _ := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-4o",
		},
	})

	resp, _ := client.Chat(context.Background(), []core.Message{
		core.NewUserMessage("Hello, please introduce yourself."),
	})

	fmt.Println(resp.Content)
}
```

### 2. Switching to DeepSeek & Capturing the "Chain of Thought" in a Stream?

Just change a few lines of initialization code and add the `WithThinking` option:

```go
import "github.com/DotNetAge/gochat/pkg/client/deepseek"

client, _ := deepseek.New(deepseek.Config{
	Config: base.Config{
		APIKey: "DEEPSEEK_API_KEY", 
		// No need to explicitly specify deepseek-reasoner; it automatically adapts when thinking is enabled.
	},
})

// Enable the thinking engine and use streaming output
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))
defer stream.Close()

fmt.Println("[🤔 Deep Thinking Process]:")
isAnswering := false

for stream.Next() {
	ev := stream.Event()
	
	if ev.Type == core.EventThinking { // Capture the internal reasoning logic
		fmt.Print(ev.Content)
	} else if ev.Type == core.EventContent {
		if !isAnswering {
			fmt.Println("\n\n[💡 Final Answer]:")
			isAnswering = true
		}
		fmt.Print(ev.Content)
	}
}
```

### 3. Calling Alibaba Qwen and Enabling Built-in Web Search?

```go
import "github.com/DotNetAge/gochat/pkg/client/compatible"

client, _ := compatible.New(compatible.Config{
	Config: base.Config{
		APIKey:  "DASHSCOPE_API_KEY",
		Model:   "qwen-max",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
	},
})

// One line of code grants the model real-time web retrieval capabilities
resp, _ := client.Chat(ctx, []core.Message{
	core.NewUserMessage("How are the top three NASDAQ indices performing today?"),
}, core.WithEnableSearch(true))

fmt.Println(resp.Content)
```

### 4. Advanced: Logging into Enterprise Gateways via OAuth2/Device Code?

If your provider (like Qwen Portal or Gemini) doesn't allow simple API Keys but requires user authorization to issue an `Access Token`, don't panic. Hand it over to `AuthManager`.

```go
import "github.com/DotNetAge/gochat/pkg/core"
import "github.com/DotNetAge/gochat/pkg/provider"

// 1. Initialize the provider's OAuth protocol implementation
p := provider.NewQwenProvider()

// 2. Mount it to the AuthManager, specifying a local persistence file (or inject your Redis Store)
authMgr := core.NewAuthManager(p, "token_store.json")

// 3. Smartly fetch the Token
// - If it exists locally and isn't expired: Return directly.
// - If it's expired: Silently Auto-Refresh via the gateway and overwrite the old file.
// - If it doesn't exist: Automatically pop up the browser auth flow, intercept the Callback, and save the new Token!
token, _ := authMgr.GetToken()

// 4. Seamlessly pass the dynamically acquired Token to the client
client, _ := compatible.New(compatible.Config{
	Config: base.Config{
		AuthToken: token.Access, // <--- Use dynamic token
		Model:     "coder-model",
		BaseURL:   "https://portal.qwen.ai",
	},
})

client.Chat(...)
```

---

## 🔌 Fully Supported Providers

*   **OpenAI** (`gpt-4o`, `o1`, `o3-mini`, etc.)
*   **Anthropic** (`claude-3-7-sonnet`, `opus`, etc.)
*   **DeepSeek** (`deepseek-chat`, `deepseek-reasoner`, etc.)
*   **Alibaba Qwen** (Cloud Models & Portal Exclusive Gateways)
*   **Ollama** (Running Open-Source Models Locally)
*   **Azure OpenAI** (Microsoft Cloud Enterprise Deployment)
*   **OpenAI-Compatible** (Any VLLM, LM Studio nodes, or proxies adhering to the OpenAI interface standard)

## 🎯 Design Philosophy & Conventions

GoChat adheres to Go's philosophy of minimalism: The core interface `core.Client` has only two methods: `Chat` and `ChatStream`.
All provider-specific customizations (like enabling search, setting thinking word limits, or injecting toolsets) are elegantly extended via **Functional Options** (`core.Option`), ensuring the main interface is never polluted.

## 📄 License

This project is open-sourced under the [MIT License](LICENSE). PRs are welcome as we build the strongest LLM infrastructure for the Go ecosystem!