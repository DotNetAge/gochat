# GoChat

[English](README.md) | [**简体中文**](README_zh.md)

GoChat 是一个专为生产环境打造的**现代化 Go 语言大模型 (LLM) 客户端 SDK**。它通过极其优雅且统一的类型安全接口，帮您彻底抹平 OpenAI、Anthropic (Claude)、DeepSeek、Qwen (通义千问)、Ollama 等各大云厂商及本地模型之间繁杂的 API 差异。

无论是基础对话，还是**深度思考 (Reasoning)**、**内置联网搜索**、**多模态 (视觉/文档)**，乃至企业级的 **OAuth2.0/Portal 网关授权持久化**，您只需掌握一套接口，便可畅行无阻。

---

## ✨ 核心杀手锏

- **🔌 一键无缝切模型**：只需改一行初始化代码，您的应用即可在 GPT-4o、Claude 3.7、DeepSeek-R1、Qwen-Max 之间随意切换，连 `Messages` 结构体都不用改。
- **🧠 原生支持深度思考 (Deep Thinking)**：内置拦截器，完美兼容 DeepSeek-R1、Claude 3.7、OpenAI o1/o3 的思维链。通过流式 API，将大模型的“内心OS”与最终答案分层剥离呈现。
- **🌐 随心开启联网搜索**：原生支持 Qwen 等具备外网检索能力的模型，一行代码 `core.WithEnableSearch(true)` 开启时效性超能力。
- **🛡️ 智能密钥托管 (Smart Secret Hosting)**：告别环境变量硬编码泄露风险！传入键名别名（如 `DASHSCOPE_API_KEY`），引擎自动嗅探并提取系统环境变量进行安全鉴权。
- **🏢 企业级 OAuth2 持久化鉴权**：内置强大的 `AuthManager`，专为接入各大厂（如 Qwen Portal、GCP Gemini）的企业网关设计。全自动处理 Device Code Flow / OAuth2 授权、Token 持久化存储（支持内存/文件/数据库等自定义 Store）以及静默自动无感刷新 (Refresh)。
- **🛠️ 完整的 Agent 基础设施**：多轮对话上下文管理、结构化 Tool/Function Calling、文档阅读提取、多模态图片识别，应有尽有。

---

## 📦 安装

```bash
go get github.com/DotNetAge/gochat
```

---

## 🚀 极速入门教程 (只需看这里就能学会)

### 1. 最基础的对话 (自带智能密钥托管)

配置好您的环境变量，例如：`export OPENAI_API_KEY="sk-..."`。

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
	// 引擎检测到 "OPENAI_API_KEY" 为键名，会自动去环境里拿真实的 sk-xxx，极其安全！
	client, _ := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-4o",
		},
	})

	resp, _ := client.Chat(context.Background(), []core.Message{
		core.NewUserMessage("你好，介绍一下你自己"),
	})

	fmt.Println(resp.Content)
}
```

### 2. 想要切换为 DeepSeek 并在流式输出中捕获“思维链”？

只需改动几行初始化代码，并加上 `WithThinking` 选项：

```go
import "github.com/DotNetAge/gochat/pkg/client/deepseek"

client, _ := deepseek.New(deepseek.Config{
	Config: base.Config{
		APIKey: "DEEPSEEK_API_KEY", 
		// 无需显式指定 deepseek-reasoner，开启思考选项后内部会自动适配
	},
})

// 开启思考引擎并使用流式输出
stream, _ := client.ChatStream(ctx, messages, core.WithThinking(0))
defer stream.Close()

fmt.Println("[🤔 深度思考过程]:")
isAnswering := false

for stream.Next() {
	ev := stream.Event()
	
	if ev.Type == core.EventThinking { // 捕获内心的推导逻辑
		fmt.Print(ev.Content)
	} else if ev.Type == core.EventContent {
		if !isAnswering {
			fmt.Println("\n\n[💡 最终答案]:")
			isAnswering = true
		}
		fmt.Print(ev.Content)
	}
}
```

### 3. 调用阿里云通义千问 (Qwen) 并开启内置联网搜索？

```go
import "github.com/DotNetAge/gochat/pkg/client/compatible"

client, _ := compatible.New(compatible.Config{
	Config: base.Config{
		APIKey:  "DASHSCOPE_API_KEY",
		Model:   "qwen-max",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
	},
})

// 一行代码，让模型拥有时效性检索能力
resp, _ := client.Chat(ctx, []core.Message{
	core.NewUserMessage("今天最新的纳斯达克三大股指表现如何？"),
}, core.WithEnableSearch(true))

fmt.Println(resp.Content)
```

### 4. 高阶：在企业内网使用 OAuth2/Device Code 登录企业级网关？

如果您的提供商（如 Qwen Portal 或 Gemini）不能用简单的 API Key，而是要求用户授权并下发 `Access Token`，别慌，交给 `AuthManager`。

```go
import "github.com/DotNetAge/gochat/pkg/core"
import "github.com/DotNetAge/gochat/pkg/provider"

// 1. 初始化供应商的 OAuth 协议实现
p := provider.NewQwenProvider()

// 2. 挂载到认证管理器，指定本地持久化文件（或者注入你的 Redis Store）
authMgr := core.NewAuthManager(p, "token_store.json")

// 3. 智能获取 Token
// - 如果本地有且没过期：直接返回。
// - 如果过期了：静默向网关自动 Refresh，然后覆盖旧文件。
// - 如果本地没有：自动弹起浏览器授权流程，拦截 Callback，拿取新 Token 保存！
token, _ := authMgr.GetToken()

// 4. 将拿到的动态 Token 无缝传入客户端
client, _ := compatible.New(compatible.Config{
	Config: base.Config{
		AuthToken: token.Access, // <--- 使用动态令牌
		Model:     "coder-model",
		BaseURL:   "https://portal.qwen.ai",
	},
})

client.Chat(...)
```

---

## 🔌 目前已全面支持的提供商

*   **OpenAI** (`gpt-4o`, `o1`, `o3-mini` 等)
*   **Anthropic** (`claude-3-7-sonnet`, `opus` 等)
*   **DeepSeek** (`deepseek-chat`, `deepseek-reasoner` 等)
*   **Alibaba Qwen 通义千问** (云端大模型与 Portal 专属网关)
*   **Ollama** (本地开源模型运行)
*   **Azure OpenAI** (微软云企业级部署)
*   **OpenAI-Compatible** (任何兼容 OpenAI 接口标准的代理、VLLM、LM Studio 节点等)

## 🎯 设计理念与规范

GoChat 秉承 Go 的极简大道：核心接口 `core.Client` 只有 `Chat` 和 `ChatStream` 两个方法。
所有不同厂商的个性化（如开启搜索、设置思考字数限制、注入工具集）全部通过 **Functional Options** (`core.Option`) 的方式优雅扩展，绝不污染主接口。

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。欢迎提交 PR 一起建设最强 Go 语言大模型基建！