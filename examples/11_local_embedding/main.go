package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/embedding"
)

type CustomModel struct {
	dimension int
}

func NewCustomModel(dimension int, modelPath string) (*CustomModel, error) {
	return &CustomModel{
		dimension: dimension,
	}, nil
}

func (m *CustomModel) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	inputIDs, ok := inputs["input_ids"].([][]int64)
	if !ok {
		return nil, fmt.Errorf("invalid input_ids type")
	}

	batchSize := len(inputIDs)

	embeddings := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		embeddings[i] = make([]float32, m.dimension)
		for j := 0; j < m.dimension; j++ {
			embeddings[i][j] = float32(i*100+j) / 100.0
		}
	}

	return map[string]interface{}{
		"last_hidden_state": embeddings,
	}, nil
}

func (m *CustomModel) Close() error {
	return nil
}

func main() {
	ctx := context.Background()

	model, err := NewCustomModel(768, "path/to/your/model.onnx")
	if err != nil {
		log.Fatalf("Failed to create custom model: %v", err)
	}
	defer model.Close()

	provider, err := embedding.New(embedding.Config{
		Model:        model,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	if err != nil {
		log.Fatalf("Failed to create local provider: %v", err)
	}

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
