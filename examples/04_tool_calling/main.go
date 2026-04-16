// Tool calling example.
//
// This example demonstrates how to use function/tool calling, allowing
// the model to invoke external functions and use their results.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/04_tool_calling/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

// Define a simple weather tool
func getWeather(location string, unit string) string {
	// In a real application, this would call a weather API
	return fmt.Sprintf("The weather in %s is 72°%s and sunny", location, unit)
}

func main() {
	client, err := openai.NewOpenAI(core.Config{
		APIKey: "OPENAI_API_KEY",
		Model:  "gpt-3.5-turbo",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Define the tool
	weatherTool := core.Tool{
		Name:        "get_weather",
		Description: "Get the current weather for a location",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"location": {
					"type": "string",
					"description": "The city name, e.g. San Francisco"
				},
				"unit": {
					"type": "string",
					"enum": ["F", "C"],
					"description": "Temperature unit"
				}
			},
			"required": ["location"]
		}`),
	}

	messages := []core.Message{
		core.NewUserMessage("What's the weather like in San Francisco?"),
	}

	// First call: model decides to use the tool
	response, err := client.Chat(context.Background(), messages,
		core.WithTools(weatherTool),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the model wants to call a tool
	if len(response.ToolCalls) > 0 {
		fmt.Println("Model wants to call tools:")

		// Add the assistant's message (with tool calls) to history
		messages = append(messages, response.Message)

		// Execute each tool call
		for _, tc := range response.ToolCalls {
			fmt.Printf("  - %s with args: %s\n", tc.Name, tc.Arguments)

			// Parse arguments
			var args struct {
				Location string `json:"location"`
				Unit     string `json:"unit"`
			}
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				log.Fatal(err)
			}

			// Execute the tool
			result := getWeather(args.Location, args.Unit)

			// Add the tool result to the conversation
			messages = append(messages, core.Message{
				Role:       core.RoleTool,
				ToolCallID: tc.ID,
				Content: []core.ContentBlock{
					{Type: core.ContentTypeText, Text: result},
				},
			})
		}

		// Second call: model uses the tool results to generate final response
		response, err = client.Chat(context.Background(), messages,
			core.WithTools(weatherTool),
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\nFinal response:", response.Content)
}
