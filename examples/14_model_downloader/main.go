package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/embedding"
)

func main() {
	ctx := context.Background()

	dl := embedding.NewDownloader("")

	fmt.Println("Available models:")
	for _, model := range dl.GetModelInfo() {
		fmt.Printf("- %s (%s): %s\n", model.Name, model.Type, model.Description)
		fmt.Printf("  Size: %s\n", model.Size)
		if len(model.URLs) > 0 {
			fmt.Printf("  URL: %s\n", model.URLs[0])
		}
		fmt.Println()
	}

	modelName := "bge-small-zh-v1.5"
	fmt.Printf("Downloading model: %s\n", modelName)

	modelPath, err := dl.DownloadModel(modelName, func(modelName, fileName string, downloaded, total int64) {
		if total > 0 {
			percentage := float64(downloaded) / float64(total) * 100
			fmt.Printf("[%s] %s: %.1f%% (%d/%d bytes)\n", modelName, fileName, percentage, downloaded, total)
		} else {
			fmt.Printf("[%s] %s: %d bytes downloaded\n", modelName, fileName, downloaded)
		}
	})

	if err != nil {
		log.Fatalf("Failed to download model: %v", err)
	}

	fmt.Printf("Model downloaded to: %s\n", modelPath)

	provider, err := embedding.NewProvider(modelPath)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	fmt.Printf("Created provider for model: %s\n", modelName)
	fmt.Printf("Embedding dimension: %d\n", provider.Dimension())

	batchProcessor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
		MaxBatchSize:  32,
		MaxConcurrent: 4,
	})

	texts := []string{
		"你好，这是一个测试",
		"Go 是一种编程语言",
		"嵌入是文本的向量表示",
	}

	embeddings, err := batchProcessor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return false
		}
		fmt.Printf("Processing: %d/%d\n", current, total)
		return true
	})

	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	fmt.Printf("Generated %d embeddings\n", len(embeddings))
	fmt.Printf("Embedding dimension: %d\n", provider.Dimension())

	for i, emb := range embeddings {
		fmt.Printf("Text %d: %s\n", i+1, texts[i])
		fmt.Printf("Embedding length: %d\n", len(emb))
		fmt.Printf("First 5 values: %v\n", emb[:5])
		fmt.Println()
	}
}
