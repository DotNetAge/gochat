// Package models provides pre-configured providers for popular embedding models.
// It includes automatic model type detection and factory functions for easy model loading.
package models

import (
	"fmt"
	"path/filepath"
)

// ModelType defines the type of embedding model.
type ModelType string

const (
	// ModelTypeBERT represents BERT models
	ModelTypeBERT ModelType = "bert"
	// ModelTypeSentenceBERT represents Sentence-BERT models
	ModelTypeSentenceBERT ModelType = "sentence-bert"
	// ModelTypeBGE represents BGE models
	ModelTypeBGE ModelType = "bge"
	// ModelTypeGPT represents GPT models
	ModelTypeGPT ModelType = "gpt"
	// ModelTypeFastText represents FastText models
	ModelTypeFastText ModelType = "fasttext"
	// ModelTypeGloVe represents GloVe models
	ModelTypeGloVe ModelType = "glove"
)

// ModelInfo contains information about a model including its type, name, and dimensions.
type ModelInfo struct {
	Type      ModelType
	Name      string
	Dimension int
	ModelPath string
}

// NewModelInfo creates a new ModelInfo instance by analyzing the model file path.
//
// Parameters:
// - modelPath: Path to the model file
//
// Returns:
// - *ModelInfo: Model information including type and dimension
// - error: Error if the model type cannot be determined
//
// Note: This is a simplified implementation that uses filename patterns for detection.
// In a production environment, more sophisticated detection methods should be used.
func NewModelInfo(modelPath string) (*ModelInfo, error) {
	// Extract model name from path
	modelName := filepath.Base(modelPath)

	// Determine model type and dimension based on filename
	var modelType ModelType
	var dimension int

	switch {
	case contains(modelName, "bert"):
		modelType = ModelTypeBERT
		dimension = 768 // Default for BERT-base
	case contains(modelName, "sentence-bert") || contains(modelName, "all-MiniLM"):
		modelType = ModelTypeSentenceBERT
		dimension = 384 // Default for MiniLM
	case contains(modelName, "bge"):
		modelType = ModelTypeBGE
		dimension = 768 // Default for BGE-small
	case contains(modelName, "gpt") || contains(modelName, "ada"):
		modelType = ModelTypeGPT
		dimension = 1536 // Default for text-embedding-ada-002
	case contains(modelName, "fasttext"):
		modelType = ModelTypeFastText
		dimension = 300 // Default for FastText
	case contains(modelName, "glove"):
		modelType = ModelTypeGloVe
		dimension = 300 // Default for GloVe
	default:
		return nil, fmt.Errorf("unknown model type for file: %s", modelName)
	}

	return &ModelInfo{
		Type:      modelType,
		Name:      modelName,
		Dimension: dimension,
		ModelPath: modelPath,
	}, nil
}

// contains checks if a string contains another string (case-insensitive)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalsIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalsIgnoreCase checks if two strings are equal (case-insensitive)
func equalsIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

// toLower converts a byte to lowercase
func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
