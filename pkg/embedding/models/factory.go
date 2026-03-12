package models

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/embedding"
	"github.com/DotNetAge/gochat/pkg/embedding/local"
)

// Provider is a generic interface for all model providers
type Provider interface {
	embedding.Provider
}

// CustomModel is a placeholder for model implementation
type CustomModel struct {
	dimension int
	modelPath string
}

// NewCustomModel creates a new custom model
func NewCustomModel(dimension int, modelPath string) (*CustomModel, error) {
	return &CustomModel{
		dimension: dimension,
		modelPath: modelPath,
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
func (m *CustomModel) Close() error {
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
	case ModelTypeBERT:
		// Create BERT provider
		model, err := NewCustomModel(info.Dimension, modelPath)
		if err != nil {
			return nil, err
		}

		localProvider, err := local.New(local.Config{
			Model:        model,
			Dimension:    info.Dimension,
			MaxBatchSize: 32,
		})
		if err != nil {
			return nil, err
		}

		return &GenericProvider{model: localProvider}, nil
	default:
		// Create generic provider for other model types
		model, err := NewCustomModel(info.Dimension, modelPath)
		if err != nil {
			return nil, err
		}

		localProvider, err := local.New(local.Config{
			Model:        model,
			Dimension:    info.Dimension,
			MaxBatchSize: 32,
		})
		if err != nil {
			return nil, err
		}

		return &GenericProvider{model: localProvider}, nil
	}
}

// GenericProvider is a generic provider for any model type
type GenericProvider struct {
	model *local.Provider
}

// Embed generates embeddings for the given texts
func (p *GenericProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.model.Embed(ctx, texts)
}

// Dimension returns the embedding dimension
func (p *GenericProvider) Dimension() int {
	return p.model.Dimension()
}
