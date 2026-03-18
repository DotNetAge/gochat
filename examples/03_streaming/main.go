// Streaming response example.
//
// This example demonstrates how to use streaming to get responses
// token-by-token as they're generated, providing a better user experience
// for long responses.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/03_streaming/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/gochat/pkg/core"
)

func main() {
	client, err := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-3.5-turbo",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	messages := []core.Message{
		core.NewUserMessage("Write a short poem about Go programming."),
	}

	// Use ChatStream instead of Chat
	stream, err := client.ChatStream(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Close()

	fmt.Print("Response: ")

	// Iterate through the stream events
	for stream.Next() {
		event := stream.Event()

		// Check for errors
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		// Print content as it arrives
		if event.Type == core.EventContent {
			fmt.Print(event.Content)
		}
	}

	fmt.Println() // New line after streaming completes

	// Get usage information after the stream finishes
	if usage := stream.Usage(); usage != nil {
		fmt.Printf("\nTokens used: %d\n", usage.TotalTokens)
	}
}
