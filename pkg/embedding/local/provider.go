// Package local provides implementations for local embedding model providers.
// It supports various model formats and includes a tokenizer for text preprocessing.
package local

import (
	"context"
	"fmt"
	"sync"
)

// Model defines the interface for local embedding models.
// Implementations should handle model loading, inference, and resource cleanup.
type Model interface {
	// Run performs inference on the given inputs and returns the model outputs.
	Run(inputs map[string]interface{}) (map[string]interface{}, error)
	
	// Close releases any resources associated with the model.
	Close() error
}

// Config contains configuration parameters for the local embedding provider.
type Config struct {
	Model        Model // Local embedding model implementation
	Dimension    int   // Embedding dimension
	MaxBatchSize int   // Maximum batch size for embedding generation
}

// Provider implements the embedding.Provider interface for local models.
// It handles tokenization, batch processing, and model inference.
type Provider struct {
	config     Config
	tokenizer  *Tokenizer
	mutex      sync.RWMutex
}

// New creates a new local embedding provider with the given configuration.
//
// Parameters:
// - config: Configuration parameters for the provider
//
// Returns:
// - *Provider: A new local embedding provider instance
// - error: Error if configuration is invalid or initialization fails
func New(config Config) (*Provider, error) {
	// Validate configuration
	if config.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}

	if config.MaxBatchSize <= 0 {
		config.MaxBatchSize = 32 // Default batch size
	}

	// Initialize tokenizer
	tokenizer, err := NewTokenizer()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tokenizer: %w", err)
	}

	return &Provider{
		config:    config,
		tokenizer: tokenizer,
	}, nil
}

// Embed generates embeddings for the given texts
func (p *Provider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// Process in batches
	var allEmbeddings [][]float32
	batchSize := p.config.MaxBatchSize

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := p.processBatch(ctx, batch)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// Dimension returns the embedding dimension
func (p *Provider) Dimension() int {
	return p.config.Dimension
}

// processBatch processes a batch of texts
func (p *Provider) processBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Tokenize texts
	inputIDs, attentionMask, err := p.tokenizer.TokenizeBatch(texts)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize texts: %w", err)
	}

	// Run inference
	inputs := map[string]interface{}{
		"input_ids":      inputIDs,
		"attention_mask": attentionMask,
	}

	outputs, err := p.config.Model.Run(inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to run inference: %w", err)
	}

	// Extract embeddings from output
	embeddings, ok := outputs["last_hidden_state"].([][]float32)
	if !ok {
		return nil, fmt.Errorf("unexpected output type")
	}

	return embeddings, nil
}

