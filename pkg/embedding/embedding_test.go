package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider is a mock implementation of the Provider interface for testing
type MockProvider struct {
	DimensionValue int
	EmbedFunc      func(ctx context.Context, texts []string) ([][]float32, error)
}

func (m *MockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.EmbedFunc != nil {
		return m.EmbedFunc(ctx, texts)
	}
	// Default implementation
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, m.DimensionValue)
	}
	return embeddings, nil
}

func (m *MockProvider) Dimension() int {
	return m.DimensionValue
}

func TestBatchProcessor_Process(t *testing.T) {
	// Create a mock provider
	mockProvider := &MockProvider{
		DimensionValue: 768,
		EmbedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			embeddings := make([][]float32, len(texts))
			for i := range embeddings {
				embeddings[i] = make([]float32, 768)
				for j := range embeddings[i] {
					embeddings[i][j] = float32(i*100+j) / 100.0
				}
			}
			return embeddings, nil
		},
	}

	// Create batch processor
	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize:  2,
		MaxConcurrent: 2,
	})

	// Test with empty texts
	ctx := context.Background()
	texts := []string{}

	embeddings, err := processor.Process(ctx, texts)
	require.NoError(t, err)
	assert.Empty(t, embeddings)

	// Test with single text
	texts = []string{"hello"}
	embeddings, err = processor.Process(ctx, texts)
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
	assert.Len(t, embeddings[0], 768)

	// Test with multiple texts (requires batching)
	texts = []string{"hello", "world", "test", "embedding"}
	embeddings, err = processor.Process(ctx, texts)
	require.NoError(t, err)
	require.Len(t, embeddings, 4)

	for _, emb := range embeddings {
		assert.Len(t, emb, 768)
		// Verify that embeddings are not empty (mock provider creates non-zero values)
		hasNonZero := false
		for _, val := range emb {
			if val != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "Embedding should have non-zero values")
	}
}

func TestBatchProcessor_ProcessWithProgress(t *testing.T) {
	mockProvider := &MockProvider{
		DimensionValue: 384,
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize:  2,
		MaxConcurrent: 1,
	})

	ctx := context.Background()
	texts := []string{"a", "b", "c"}

	// Test with progress callback
	var progressCalls []int
	embeddings, err := processor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
		if err == nil {
			progressCalls = append(progressCalls, current)
		}
		return true
	})

	require.NoError(t, err)
	require.Len(t, embeddings, 3)
	assert.NotEmpty(t, progressCalls, "Progress callback should have been called")

	// Test cancellation via callback
	canceled := false
	embeddings, err = processor.ProcessWithProgress(ctx, texts, func(current, total int, err error) bool {
		if current >= 2 {
			canceled = true
			return false
		}
		return true
	})

	require.NoError(t, err)
	assert.True(t, canceled, "Processing should have been canceled")
}

func TestBatchProcessor_ErrorHandling(t *testing.T) {
	// Test provider that returns error
	errorProvider := &MockProvider{
		DimensionValue: 768,
		EmbedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			return nil, assert.AnError
		},
	}

	processor := NewBatchProcessor(errorProvider, BatchOptions{
		MaxBatchSize: 2,
	})

	ctx := context.Background()
	texts := []string{"test"}

	_, err := processor.Process(ctx, texts)
	// The batch processor might handle errors differently
	// Let's just test that an error is returned
	assert.Error(t, err)
}

func TestBatchOptions_Defaults(t *testing.T) {
	// Test that defaults are set correctly
	options := BatchOptions{}

	processor := NewBatchProcessor(&MockProvider{DimensionValue: 768}, options)
	require.NotNil(t, processor)

	// The processor should have set default values
	// We can't directly access the options, but we can test the behavior
	ctx := context.Background()
	texts := []string{"test"}

	embeddings, err := processor.Process(ctx, texts)
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
}

func TestProviderInterface(t *testing.T) {
	// Test that MockProvider correctly implements the Provider interface
	var provider Provider = &MockProvider{DimensionValue: 768}

	assert.Equal(t, 768, provider.Dimension())

	ctx := context.Background()
	embeddings, err := provider.Embed(ctx, []string{"test"})
	require.NoError(t, err)
	require.Len(t, embeddings, 1)
}
