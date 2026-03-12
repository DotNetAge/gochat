package models

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/embedding/local"
)

// BGEProvider creates a provider for BGE models
type BGEProvider struct {
	model *local.Provider
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

	// Create custom model
	model, err := NewCustomModel(info.Dimension, modelPath)
	if err != nil {
		return nil, err
	}

	// Create local provider
	localProvider, err := local.New(local.Config{
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
