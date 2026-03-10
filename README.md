# GoChat

GoChat是一个Go语言库，用于与多种LLM（大型语言模型）提供商进行交互。它提供了统一的接口和丰富的功能，使开发者能够轻松集成不同的LLM服务。

## 功能特性

- 支持多种LLM提供商：OpenAI、Anthropic、Ollama、兼容OpenAI API的服务
- 支持API Key和Auth Token两种认证方式
- 支持流式和非流式请求
- 内置重试机制，带有指数退避策略
- 结构化错误处理
- 支持从环境变量获取API密钥，提高安全性

## 安装

```bash
go get github.com/DotNetAge/gochat
```

## 使用示例

### OpenAI客户端

```go
package main

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/client/openai"
)

func main() {
	// 使用API Key
	config := openai.Config{
		Config: base.Config{
			APIKey:  "your-api-key",
			Model:   "gpt-3.5-turbo",
			BaseURL: "https://api.openai.com",
		},
	}

	// 或者使用Auth Token
	// config := openai.Config{
	//     Config: base.Config{
	//         AuthToken: "your-auth-token",
	//         Model:     "gpt-3.5-turbo",
	//         BaseURL:   "https://api.openai.com",
	//     },
	// }

	client, err := openai.New(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	// 非流式请求
	response, err := client.Complete(context.Background(), "Hello, who are you?")
	if err != nil {
		fmt.Printf("Error getting completion: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response)

	// 流式请求
	stream, err := client.CompleteStream(context.Background(), "Write a short poem about AI")
	if err != nil {
		fmt.Printf("Error getting stream: %v\n", err)
		return
	}

	fmt.Println("Streaming response:")
	for chunk := range stream {
		fmt.Print(chunk)
	}
	fmt.Println()
}
```

### Anthropic客户端

```go
package main

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/client/anthropic"
)

func main() {
	config := anthropic.Config{
		Config: base.Config{
			APIKey:  "your-api-key",
			Model:   "claude-3-opus-20240229",
			BaseURL: "https://api.anthropic.com",
		},
	}

	client, err := anthropic.New(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	response, err := client.Complete(context.Background(), "Hello, who are you?")
	if err != nil {
		fmt.Printf("Error getting completion: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response)
}
```

### Ollama客户端

```go
package main

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/client/ollama"
)

func main() {
	config := ollama.Config{
		Config: base.Config{
			Model:   "llama2",
			BaseURL: "http://localhost:11434",
		},
	}

	client, err := ollama.New(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	response, err := client.Complete(context.Background(), "Hello, who are you?")
	if err != nil {
		fmt.Printf("Error getting completion: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response)
}
```

### 兼容OpenAI API的客户端

```go
package main

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/client/compatible"
)

func main() {
	config := compatible.Config{
		Config: base.Config{
			APIKey:  "your-api-key",
			Model:   "gpt-3.5-turbo",
			BaseURL: "https://api.example.com", // 替换为兼容OpenAI API的服务地址
		},
	}

	client, err := compatible.New(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}

	response, err := client.Complete(context.Background(), "Hello, who are you?")
	if err != nil {
		fmt.Printf("Error getting completion: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", response)
}
```

## 配置选项

所有客户端都支持以下配置选项：

| 选项 | 描述 | 默认值 |
|------|------|--------|
| APIKey | LLM提供商的API密钥 | 无（必需，除非提供AuthToken） |
| AuthToken | LLM提供商的认证令牌（替代APIKey） | 无 |
| Model | 使用的模型名称 | 各提供商的默认模型 |
| BaseURL | API请求的基础URL | 各提供商的默认URL |
| Timeout | 请求超时时间 | 30秒（Ollama为60秒） |
| MaxRetries | 最大重试次数 | 3 |
| Temperature | 生成温度（0.0-1.0） | 0.7 |
| MaxTokens | 最大生成令牌数 | 无限制 |

## 环境变量支持

库支持从环境变量获取API密钥，提高安全性：

- 对于APIKey，会尝试从大写的环境变量中获取（例如，OpenAI的APIKey会尝试从`OPENAI_API_KEY`环境变量获取）
- 对于AuthToken，会尝试从大写的环境变量中获取（例如，`OPENAI_AUTH_TOKEN`）

## 错误处理

库提供了结构化的错误处理，错误类型包括：

- `ErrorTypeAPI`：API错误
- `ErrorTypeNetwork`：网络错误
- `ErrorTypeTimeout`：超时错误
- `ErrorTypeValidation`：验证错误
- `ErrorTypeUnknown`：未知错误

## 重试机制

库内置了智能重试机制：

- 对于可重试的错误（如速率限制、超时、临时错误等）会自动重试
- 使用指数退避策略，避免请求雪崩
- 最大重试次数可通过配置设置

## 贡献

欢迎提交Issue和Pull Request来改进这个库。

## 许可证

MIT
