package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/embedding"
)

func main() {
	ctx := context.Background()

	bgeProvider, err := embedding.NewProvider("path/to/bge-small-zh-v1.5.onnx")
	if err != nil {
		log.Printf("Failed to create BGE provider: %v", err)
	} else {
		fmt.Println("Created BGE provider successfully")
		fmt.Printf("BGE model dimension: %d\n", bgeProvider.Dimension())
	}

	sbertProvider, err := embedding.NewProvider("path/to/all-MiniLM-L6-v2.onnx")
	if err != nil {
		log.Printf("Failed to create Sentence-BERT provider: %v", err)
	} else {
		fmt.Println("Created Sentence-BERT provider successfully")
		fmt.Printf("Sentence-BERT model dimension: %d\n", sbertProvider.Dimension())
	}

	genericProvider, err := embedding.NewProvider("path/to/bert-base-uncased.onnx")
	if err != nil {
		log.Printf("Failed to create generic provider: %v", err)
	} else {
		fmt.Println("Created generic provider successfully")
		fmt.Printf("Generic model dimension: %d\n", genericProvider.Dimension())
	}

	var testProvider embedding.Provider
	if bgeProvider != nil {
		testProvider = bgeProvider
	} else if sbertProvider != nil {
		testProvider = sbertProvider
	} else if genericProvider != nil {
		testProvider = genericProvider
	} else {
		log.Fatal("No provider created successfully")
	}

	batchProcessor := embedding.NewBatchProcessor(testProvider, embedding.BatchOptions{
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
	fmt.Printf("Embedding dimension: %d\n", testProvider.Dimension())

	for i, emb := range embeddings {
		fmt.Printf("Text %d: %s\n", i+1, texts[i])
		fmt.Printf("Embedding length: %d\n", len(emb))
		fmt.Printf("First 5 values: %v\n", emb[:5])
		fmt.Println()
	}
}
