package embedding

import (
	"fmt"
	"os"
	"path/filepath"
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

// WithBEG 创建 BGE Embedding Provider
// modelName: 模型名称，例如 "bge-small-zh-v1.5"
// modelPath: 模型路径，如果为空则自动下载
func WithBEG(modelName, modelPath string) (Provider, error) {
	// 如果未指定路径，使用默认下载目录
	if modelPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		modelPath = filepath.Join(homeDir, ".embedding", modelName)
	}

	// 检查模型是否存在
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		// 自动下载模型
		fmt.Printf("Model not found, downloading %s to %s...\n", modelName, modelPath)
		downloader := NewDownloader("")
		_, downloadErr := downloader.DownloadModel(modelName, nil)
		if downloadErr != nil {
			return nil, fmt.Errorf("failed to download model: %w", downloadErr)
		}
		fmt.Println("Model downloaded successfully")
	}

	// 创建 Provider
	return NewProvider(modelPath)
}

// WithBERT 创建 BERT Embedding Provider
// modelName: 模型名称，例如 "all-mpnet-base-v2"
// modelPath: 模型路径，如果为空则自动下载
func WithBERT(modelName, modelPath string) (Provider, error) {
	// 如果未指定路径，使用默认下载目录
	if modelPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		modelPath = filepath.Join(homeDir, ".embedding", modelName)
	}

	// 检查模型是否存在
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		// 自动下载模型
		fmt.Printf("Model not found, downloading %s to %s...\n", modelName, modelPath)
		downloader := NewDownloader("")
		_, downloadErr := downloader.DownloadModel(modelName, nil)
		if downloadErr != nil {
			return nil, fmt.Errorf("failed to download model: %w", downloadErr)
		}
		fmt.Println("Model downloaded successfully")
	}

	// 创建 Provider
	return NewProvider(modelPath)
}
