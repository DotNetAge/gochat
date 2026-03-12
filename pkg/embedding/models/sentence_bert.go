package models

import (
	"context"
	"fmt"

	"github.com/DotNetAge/gochat/pkg/embedding/local"
)

// SentenceBERTProvider creates a provider for Sentence-BERT models
type SentenceBERTProvider struct {
	model *local.Provider
}

// NewSentenceBERTProvider creates a new Sentence-BERT provider
func NewSentenceBERTProvider(modelPath string) (*SentenceBERTProvider, error) {
	// Create model info
	info, err := NewModelInfo(modelPath)
	if err != nil {
		return nil, err
	}

	// Validate model type
	if info.Type != ModelTypeSentenceBERT {
		return nil, fmt.Errorf("expected Sentence-BERT model, got %s", info.Type)
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

	return &SentenceBERTProvider{
		model: localProvider,
	}, nil
}

// Embed generates embeddings for the given texts
func (p *SentenceBERTProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return p.model.Embed(ctx, texts)
}

// Dimension returns the embedding dimension
func (p *SentenceBERTProvider) Dimension() int {
	return p.model.Dimension()
}
