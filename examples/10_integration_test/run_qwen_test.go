package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

func TestQWen(t *testing.T) {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		t.Skip("DASHSCOPE_API_KEY environment variable is not set. Please set it to run this test.")
	}

	fmt.Println("=== Aliyun QWen Integration Test (Model: qwen-max) ===")

	client, err := openai.NewOpenAI(core.Config{
		APIKey:  apiKey,
		Model:   "qwen-max",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode",
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Println("\n--- Testing QWen with Web Search Enabled ---")
	messages := []core.Message{
		core.NewUserMessage("今天最新的纳斯达克三大股指表现如何？"),
	}

	stream, err := client.ChatStream(ctx, messages, core.WithEnableSearch(true))
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

	fmt.Println("\n=== QWen Test Completed Successfully ===")
}
