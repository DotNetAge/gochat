package embedding

import (
	"fmt"
)

// Model is a generic model implementation for embedding generation
type Model struct {
	dimension int
	modelPath string
}

// NewModel creates a new model instance
func NewModel(dimension int, modelPath string) (*Model, error) {
	return &Model{
		dimension: dimension,
		modelPath: modelPath,
	}, nil
}

// Run runs inference on the given inputs
func (m *Model) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	// Extract input IDs to determine batch size
	inputIDs, ok := inputs["input_ids"].([][]int64)
	if !ok {
		return nil, fmt.Errorf("invalid input_ids type")
	}

	batchSize := len(inputIDs)

	// Create mock embeddings for demonstration
	embeddings := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		embeddings[i] = make([]float32, m.dimension)
		// Fill with mock values
		for j := 0; j < m.dimension; j++ {
			embeddings[i][j] = float32(i*100+j) / 100.0
		}
	}

	return map[string]interface{}{
		"last_hidden_state": embeddings,
	}, nil
}

// Close closes the model
func (m *Model) Close() error {
	return nil
}

// NewProvider creates a new provider based on the model path
func NewProvider(modelPath string) (Provider, error) {
	// Create model info
	info, err := NewModelInfo(modelPath)
	if err != nil {
		return nil, err
	}

	// Create provider based on model type
	switch info.Type {
	case ModelTypeBGE:
		return NewBGEProvider(modelPath)
	case ModelTypeSentenceBERT:
		return NewSentenceBERTProvider(modelPath)
	default:
		// Create generic provider for BERT and other model types
		model, err := NewModel(info.Dimension, modelPath)
		if err != nil {
			return nil, err
		}

		return New(Config{
			Model:        model,
			Dimension:    info.Dimension,
			MaxBatchSize: 32,
		})
	}
}
