package main

import (
	"context"
	"fmt"
	"log"

	"github.com/DotNetAge/gochat/pkg/embedding"
	"github.com/DotNetAge/gochat/pkg/embedding/local"
)

// CustomModel is a custom implementation of the local.Model interface
// In a real application, you would implement this with a real model
// For example, using ONNX Runtime, TensorFlow Lite, or other ML frameworks
type CustomModel struct {
	dimension int
	// Add model-specific fields here
	// e.g., modelPath string
	// e.g., session *onnxruntime.InferenceSession
}

// NewCustomModel creates a new custom model
func NewCustomModel(dimension int, modelPath string) (*CustomModel, error) {
	// Initialize your model here
	// Example: load ONNX model
	// session, err := onnxruntime.NewInferenceSession(modelPath)
	// if err != nil {
	//     return nil, err
	// }

	return &CustomModel{
		dimension: dimension,
		// Add initialized fields here
	}, nil
}

// Run runs inference on the given inputs
func (m *CustomModel) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	// Extract input IDs to determine batch size
	inputIDs, ok := inputs["input_ids"].([][]int64)
	if !ok {
		return nil, fmt.Errorf("invalid input_ids type")
	}

	batchSize := len(inputIDs)

	// In a real implementation, you would:
	// 1. Preprocess the input (e.g., convert to tensor)
	// 2. Run the model inference
	// 3. Postprocess the output (e.g., extract embeddings)

	// Create mock embeddings for demonstration
	embeddings := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		embeddings[i] = make([]float32, m.dimension)
		// Fill with mock values
		for j := 0; j < m.dimension; j++ {
			embeddings[i][j] = float32(i*100 + j) / 100.0
		}
	}

	return map[string]interface{}{
		"last_hidden_state": embeddings,
	}, nil
}

// Close closes the model
func (m *CustomModel) Close() error {
	// Clean up resources here
	// Example: close ONNX session
	// if m.session != nil {
	//     m.session.Close()
	// }
	return nil
}

func main() {
	ctx := context.Background()

	// Create custom model (load your actual model file here)
	model, err := NewCustomModel(768, "path/to/your/model.onnx")
	if err != nil {
		log.Fatalf("Failed to create custom model: %v", err)
	}
	defer model.Close()

	// Create local embedding provider
	provider, err := local.New(local.Config{
		Model:        model,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	if err != nil {
		log.Fatalf("Failed to create local provider: %v", err)
	}

	// Create batch processor for better performance
	batchProcessor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
		MaxBatchSize:  32,
		MaxConcurrent: 4,
	})

	// Texts to embed
	texts := []string{
		"你好，这是一个测试",
		"Go 是一种编程语言",
		"嵌入是文本的向量表示",
	}

	// Generate embeddings with progress tracking
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

	// Print results
	fmt.Printf("Generated %d embeddings\n", len(embeddings))
	fmt.Printf("Embedding dimension: %d\n", provider.Dimension())

	for i, emb := range embeddings {
		fmt.Printf("Text %d: %s\n", i+1, texts[i])
		fmt.Printf("Embedding length: %d\n", len(emb))
		fmt.Printf("First 5 values: %v\n", emb[:5])
		fmt.Println()
	}
}

