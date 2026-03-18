// Multiple images example - analyzing multiple images in one request.
//
// This example shows how to send multiple images to the model for
// comparison, analysis, or batch processing.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/08_multiple_images/main.go image1.jpg image2.jpg image3.jpg
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/gochat/pkg/core"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run main.go <image1> <image2> [image3...]")
	}
	imagePaths := os.Args[1:]

	// Create a client
	client, err := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-4-vision-preview",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Build content blocks with text and multiple images
	contentBlocks := []core.ContentBlock{
		{
			Type: core.ContentTypeText,
			Text: fmt.Sprintf("I'm providing you with %d images. Please:\n1. Describe what you see in each image\n2. Compare and contrast them\n3. Identify any common themes or differences\n\nImages:", len(imagePaths)),
		},
	}

	// Add each image
	for i, imagePath := range imagePaths {
		// Read image
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			log.Fatalf("Failed to read %s: %v", imagePath, err)
		}

		// Encode to base64
		base64Image := base64.StdEncoding.EncodeToString(imageData)

		// Detect media type
		mediaType := "image/jpeg"
		ext := filepath.Ext(imagePath)
		switch ext {
		case ".png":
			mediaType = "image/png"
		case ".webp":
			mediaType = "image/webp"
		case ".gif":
			mediaType = "image/gif"
		}

		// Add image block
		contentBlocks = append(contentBlocks, core.ContentBlock{
			Type:      core.ContentTypeImage,
			MediaType: mediaType,
			Data:      base64Image,
			FileName:  filepath.Base(imagePath),
		})

		fmt.Printf("Loaded image %d: %s (%d bytes)\n", i+1, filepath.Base(imagePath), len(imageData))
	}

	// Create message with all images
	messages := []core.Message{
		{
			Role:    core.RoleUser,
			Content: contentBlocks,
		},
	}

	fmt.Println("\nAnalyzing images...")

	// Send request
	response, err := client.Chat(context.Background(), messages,
		core.WithMaxTokens(1500),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nAnalysis:")
	fmt.Println(response.Content)
	fmt.Printf("\nTokens used: %d\n", response.Usage.TotalTokens)
}
