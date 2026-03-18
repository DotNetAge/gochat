// Document analysis example - analyzing PDF, text files, or other documents.
//
// This example shows how to send document content to the model for analysis.
// For text-based documents (PDF, TXT, MD, code files), we extract the text
// and send it as a text message.
//
// To run:
//
//	export OPENAI_API_KEY="your-key-here"
//	go run examples/07_document_analysis/main.go path/to/document.txt
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/DotNetAge/gochat/pkg/client/base"
	"github.com/DotNetAge/gochat/pkg/client/openai"
	"github.com/DotNetAge/gochat/pkg/core"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <document-path>")
	}
	docPath := os.Args[1]

	// Read the document
	content, err := os.ReadFile(docPath)
	if err != nil {
		log.Fatalf("Failed to read document: %v", err)
	}

	// Get file info
	fileName := filepath.Base(docPath)
	fileExt := filepath.Ext(docPath)

	// Create a client
	client, err := openai.New(openai.Config{
		Config: base.Config{
			APIKey: "OPENAI_API_KEY",
			Model:  "gpt-4",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a message with the document content
	// For text files, we include the content directly
	// For binary files (PDF, DOCX), you'd need to extract text first
	messages := []core.Message{
		core.NewSystemMessage("You are a helpful document analysis assistant."),
		{
			Role: core.RoleUser,
			Content: []core.ContentBlock{
				{
					Type: core.ContentTypeText,
					Text: fmt.Sprintf("I have a %s file named '%s'. Here's its content:\n\n%s\n\nPlease analyze this document and provide:\n1. A brief summary\n2. Key points or main topics\n3. Any notable patterns or insights",
						fileExt, fileName, string(content)),
				},
			},
		},
	}

	fmt.Printf("Analyzing document: %s\n", fileName)
	fmt.Printf("File size: %d bytes\n\n", len(content))

	// Send the request
	response, err := client.Chat(context.Background(), messages,
		core.WithTemperature(0.3), // Lower temperature for more focused analysis
		core.WithMaxTokens(1000),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Analysis:")
	fmt.Println(response.Content)
	fmt.Printf("\nTokens used: %d (prompt: %d, completion: %d)\n",
		response.Usage.TotalTokens,
		response.Usage.PromptTokens,
		response.Usage.CompletionTokens,
	)
}
