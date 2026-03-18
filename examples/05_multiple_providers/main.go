// Multiple providers example.
//
// This example shows how to use different LLM providers with the same code.
// The core.Client interface is implemented by all providers, making it easy
// to switch between them.
//
// To run:
//
//	export OPENAI_API_KEY="your-openai-key"
//	export ANTHROPIC_API_KEY="your-anthropic-key"
//	go run examples/05_multiple_providers/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/client/anthropic"
	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/ollama"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/gochat/pkg/core"
)

// askQuestion demonstrates using any provider through the core.Client interface
func askQuestion(client core.Client, providerName string) {
	messages := []core.Message{
		core.NewUserMessage("What is 2+2? Answer in one word."),
	}

	response, err := client.Chat(context.Background(), messages)
	if err != nil {
		log.Printf("%s error: %v", providerName, err)
		return
	}

	fmt.Printf("%s: %s (model: %s, tokens: %d)\n",
		providerName,
		response.Content,
		response.Model,
		response.Usage.TotalTokens,
	)
}

func main() {
	// OpenAI
	openaiClient, err := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-3.5-turbo",
		},
	})
	if err == nil {
		askQuestion(openaiClient, "OpenAI")
	}

	// Anthropic (Claude)
	anthropicClient, err := anthropic.New(anthropic.Config{
		Config: base.Config{
			APIKey: "ANTHROPIC_API_KEY",
			Model:  "claude-3-haiku-20240307",
		},
	})
	if err == nil {
		askQuestion(anthropicClient, "Anthropic")
	}

	// Ollama (local)
	// Note: Requires Ollama to be running locally
	ollamaClient, err := ollama.New(ollama.Config{
		Config: base.Config{
			Model:   "llama2",
			BaseURL: "http://localhost:11434",
		},
	})
	if err == nil {
		askQuestion(ollamaClient, "Ollama")
	}
}
