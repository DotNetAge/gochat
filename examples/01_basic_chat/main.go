// Basic example demonstrating simple chat completion with OpenAI.
//
// This example shows the most basic usage: sending a single message
// and getting a response.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/01_basic_chat/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

func main() {
	// Create a client
	client, err := openai.NewOpenAI(core.Config{
		APIKey: "OPENAI_API_KEY", // Will read from environment variable
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a simple message
	messages := []core.Message{
		core.NewUserMessage("What is the capital of France?"),
	}

	// Send the message and get a response
	response, err := client.Chat(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response
	fmt.Println("Response:", response.Content)
	fmt.Printf("Tokens used: %d\n", response.Usage.TotalTokens)
}
