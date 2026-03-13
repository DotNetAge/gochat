package embedding

import (
	"context"
	"fmt"
)

// BGEProvider creates a provider for BGE models
type BGEProvider struct {
	model Provider
}

// NewBGEProvider creates a new BGE provider
func NewBGEProvider(modelPath string) (*BGEProvider, error) {
	// Create model info
	info, err := NewModelInfo(modelPath)
	if err != nil {
		return nil, err
	}

	// Validate model type
	if info.Type != ModelTypeBGE {
		return nil, fmt.Errorf("expected BGE model, got %s", info.Type)
	}

	// Create model
	model, err := NewModel(info.Dimension, modelPath)
	if err != nil {
		return nil, err
	}

	// Create local provider
	localProvider, err := New(Config{
		Model:        model,
		Dimension:    info.Dimension,
		MaxBatchSize: 32,
	})
	if err != nil {
		return nil, err
	}

	return &BGEProvider{
		model: localProvider,
	}, nil
}

// Embed generates embeddings for the given texts
func (p *BGEProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.model.Embed(ctx, texts)
}

// Dimension returns the embedding dimension
func (p *BGEProvider) Dimension() int {
	return p.model.Dimension()
}
