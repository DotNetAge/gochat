package models

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModelInfo(t *testing.T) {
	tests := []struct {
		name        string
		modelPath   string
		expectedType ModelType
		expectedDim  int
		expectError  bool
	}{
		{
			name:        "BERT model",
			modelPath:   "/path/to/bert-base-uncased.onnx",
			expectedType: ModelTypeBERT,
			expectedDim:  768,
			expectError:  false,
		},
		{
			name:        "Sentence-BERT model",
			modelPath:   "/path/to/all-MiniLM-L6-v2.onnx",
			expectedType: ModelTypeSentenceBERT,
			expectedDim:  384,
			expectError:  false,
		},
		{
			name:        "BGE model",
			modelPath:   "/path/to/bge-small-zh-v1.5.onnx",
			expectedType: ModelTypeBGE,
			expectedDim:  512,
			expectError:  false,
		},

		{
			name:        "GPT model",
			modelPath:   "/path/to/gpt-embedding.onnx",
			expectedType: ModelTypeGPT,
			expectedDim:  1536,
			expectError:  false,
		},
		{
			name:        "FastText model",
			modelPath:   "/path/to/fasttext-model.onnx",
			expectedType: ModelTypeFastText,
			expectedDim:  300,
			expectError:  false,
		},
		{
			name:        "GloVe model",
			modelPath:   "/path/to/glove-model.onnx",
			expectedType: ModelTypeGloVe,
			expectedDim:  300,
			expectError:  false,
		},
		{
			name:        "Unknown model",
			modelPath:   "/path/to/unknown-model.onnx",
			expectedType: "",
			expectedDim:  0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelInfo, err := NewModelInfo(tt.modelPath)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, modelInfo)
			} else {
				require.NoError(t, err)
				require.NotNil(t, modelInfo)
				
				assert.Equal(t, tt.expectedType, modelInfo.Type)
				assert.Equal(t, tt.expectedDim, modelInfo.Dimension)
				assert.Equal(t, filepath.Base(tt.modelPath), modelInfo.Name)
				assert.Equal(t, tt.modelPath, modelInfo.ModelPath)
			}
		})
	}
}

func TestModelTypeConstants(t *testing.T) {
	// Test that all model type constants are defined correctly
	assert.Equal(t, ModelType("bert"), ModelTypeBERT)
	assert.Equal(t, ModelType("sentence-bert"), ModelTypeSentenceBERT)
	assert.Equal(t, ModelType("bge"), ModelTypeBGE)
	assert.Equal(t, ModelType("gpt"), ModelTypeGPT)
	assert.Equal(t, ModelType("fasttext"), ModelTypeFastText)
	assert.Equal(t, ModelType("glove"), ModelTypeGloVe)
}

func TestModelInfoStructure(t *testing.T) {
	// Test that ModelInfo structure can be created and used
	modelInfo := &ModelInfo{
		Type:      ModelTypeBERT,
		Name:      "test-model",
		Dimension: 768,
		ModelPath: "/path/to/model.onnx",
	}

	assert.Equal(t, ModelTypeBERT, modelInfo.Type)
	assert.Equal(t, "test-model", modelInfo.Name)
	assert.Equal(t, 768, modelInfo.Dimension)
	assert.Equal(t, "/path/to/model.onnx", modelInfo.ModelPath)
}

func TestEdgeCases(t *testing.T) {
	// Test edge cases for NewModelInfo
	tests := []struct {
		name       string
		modelPath  string
		expectError bool
	}{
		{"empty path", "", true},
		{"path with spaces", "/path with spaces/bert-model.onnx", false},
		{"path with special chars", "/path/with-special_chars/bge-model.onnx", false},
		{"very long path", "/very/long/path/that/goes/on/and/on/minilm-model.onnx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modelInfo, err := NewModelInfo(tt.modelPath)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, modelInfo)
			} else {
				// For unknown model types, we expect an error
				if err != nil {
					assert.Contains(t, err.Error(), "unknown model type")
				}
			}
		})
	}
}