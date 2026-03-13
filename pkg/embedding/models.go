// Package models provides pre-configured providers for popular embedding models.
// It includes automatic model type detection and factory functions for easy model loading.
package embedding

import (
	"fmt"
	"path/filepath"
	"strings"
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
func NewModelInfo(modelPath string) (*ModelInfo, error) {
	// Extract model name from path
	modelName := filepath.Base(modelPath)
	lowerName := strings.ToLower(modelName)

	// Determine model type and dimension based on filename patterns
	var modelType ModelType
	var dimension int

	switch {
	case strings.Contains(lowerName, "bge"):
		modelType = ModelTypeBGE
		if strings.Contains(lowerName, "small") {
			dimension = 512
		} else if strings.Contains(lowerName, "large") {
			dimension = 1024
		} else {
			dimension = 768 // Default for BGE-base
		}
	case strings.Contains(lowerName, "sentence-bert") || strings.Contains(lowerName, "all-minilm"):
		modelType = ModelTypeSentenceBERT
		dimension = 384 // Default for MiniLM
	case strings.Contains(lowerName, "bert"):
		modelType = ModelTypeBERT
		dimension = 768 // Default for BERT-base
	case strings.Contains(lowerName, "gpt") || strings.Contains(lowerName, "ada"):
		modelType = ModelTypeGPT
		dimension = 1536 // Default for text-embedding-ada-002
	case strings.Contains(lowerName, "fasttext"):
		modelType = ModelTypeFastText
		dimension = 300 // Default for FastText
	case strings.Contains(lowerName, "glove"):
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
