package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/ollama"
	"github.com/DotNetAge/gochat/pkg/core"
)

func TestOllama(t *testing.T) {
	fmt.Println("=== Ollama Integration Test (Model: qwen3:1.7b) ===")

	client, err := ollama.New(ollama.Config{
		Config: base.Config{
			Model:   "qwen3:1.7b",
			BaseURL: "http://localhost:11434",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	messages := []core.Message{
		core.NewUserMessage("你好，请用一句话介绍一下你自己。"),
	}

	fmt.Println("\n--- Testing Streaming Chat ---")
	stream, err := client.ChatStream(ctx, messages)
	if err != nil {
		log.Fatalf("Stream request failed: %v", err)
	}
	defer stream.Close()

	fmt.Print("Response: ")
	for stream.Next() {
		ev := stream.Event()
		if ev.Err != nil {
			log.Fatalf("\nStream error: %v", ev.Err)
		}
		if ev.Type == core.EventContent {
			fmt.Print(ev.Content)
		}
	}
	fmt.Println()

	if usage := stream.Usage(); usage != nil {
		fmt.Printf("\nToken Usage: Prompt %d, Completion %d, Total %d\n",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	}

	fmt.Println("\n=== Ollama Test Completed Successfully ===")
}
