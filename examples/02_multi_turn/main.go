// Multi-turn conversation example.
//
// This example shows how to maintain conversation history across multiple
// turns, allowing the model to remember context from previous messages.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/02_multi_turn/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

func main() {
	client, err := openai.NewOpenAI(core.Config{
		APIKey: "OPENAI_API_KEY",
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start with a system message to set the assistant's behavior
	messages := []core.Message{
		core.NewSystemMessage("You are a helpful math tutor."),
		core.NewUserMessage("What is 15 * 7?"),
	}

	// First turn
	response, err := client.Chat(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Assistant:", response.Content)

	// Add the assistant's response to the conversation history
	messages = append(messages, core.Message{
		Role: core.RoleAssistant,
		Content: []core.ContentBlock{
			{Type: core.ContentTypeText, Text: response.Content},
		},
	})

	// Second turn - the model remembers the previous question
	messages = append(messages, core.NewUserMessage("Now multiply that by 2"))

	response, err = client.Chat(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Assistant:", response.Content)

	// Third turn
	messages = append(messages, core.Message{
		Role: core.RoleAssistant,
		Content: []core.ContentBlock{
			{Type: core.ContentTypeText, Text: response.Content},
		},
	})
	messages = append(messages, core.NewUserMessage("What was my first question?"))

	response, err = client.Chat(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Assistant:", response.Content)
}
