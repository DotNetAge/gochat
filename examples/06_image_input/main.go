// Multimodal input example - sending images to the model.
//
// This example demonstrates how to send images along with text to models
// that support vision capabilities (GPT-4 Vision, Claude 3, etc.).
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/06_image_input/main.go path/to/image.jpg
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/DotNetAge/gochat/client/openai"
	"github.com/DotNetAge/gochat/core"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <image-path>")
	}
	imagePath := os.Args[1]

	// Read the image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	// Encode to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Detect image type from file extension
	mediaType := "image/jpeg"
	if len(imagePath) > 4 {
		ext := imagePath[len(imagePath)-4:]
		switch ext {
		case ".png":
			mediaType = "image/png"
		case ".jpg", "jpeg":
			mediaType = "image/jpeg"
		case "webp":
			mediaType = "image/webp"
		case ".gif":
			mediaType = "image/gif"
		}
	}

	// Create a client with a vision-capable model
	client, err := openai.NewOpenAI(core.Config{
		APIKey: "OPENAI_API_KEY",
		Model:  "gpt-4-vision-preview", // or "gpt-4o"
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a message with both text and image
	messages := []core.Message{
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type: core.ContentTypeText,
					Text: "What's in this image? Please describe it in detail.",
				},
				{
					Type:      core.ContentTypeImage,
					MediaType: mediaType,
					Data:      base64Image,
				},
			},
		},
	}

	fmt.Println("Analyzing image...")

	// Send the request
	response, err := client.Chat(context.Background(), messages,
		core.WithMaxTokens(500),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nModel's response:")
	fmt.Println(response.Content)
	fmt.Printf("\nTokens used: %d\n", response.Usage.TotalTokens)
}
