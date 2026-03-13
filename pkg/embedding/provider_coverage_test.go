package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModelForProvider struct {
	dimension int
	runFunc   func(inputs map[string]interface{}) (map[string]interface{}, error)
}

func (m *mockModelForProvider) Run(inputs map[string]interface{}) (map[string]interface{}, error) {
	if m.runFunc != nil {
		return m.runFunc(inputs)
	}
	inputIDs, ok := inputs["input_ids"].([][]int64)
	if !ok {
		return nil, assert.AnError
	}
	batchSize := len(inputIDs)
	embeddings := make([][]float32, batchSize)
	for i := 0; i < batchSize; i++ {
		embeddings[i] = make([]float32, m.dimension)
		for j := 0; j < m.dimension; j++ {
			embeddings[i][j] = float32(i*100 + j)
		}
	}
	return map[string]interface{}{
		"last_hidden_state": embeddings,
	}, nil
}

func (m *mockModelForProvider) Close() error {
	return nil
}

func TestBGEProvider_Embed(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
	}

	localProvider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	bgeProvider := &BGEProvider{
		model: localProvider,
	}

	ctx := context.Background()
	texts := []string{"hello", "world"}

	embeddings, err := bgeProvider.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	assert.Len(t, embeddings[0], 768)
}

func TestBGEProvider_Dimension(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
	}

	localProvider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	bgeProvider := &BGEProvider{
		model: localProvider,
	}

	assert.Equal(t, 768, bgeProvider.Dimension())
}

func TestSentenceBERTProvider_Embed(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 384,
	}

	localProvider, err := New(Config{
		Model:        mock,
		Dimension:    384,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	sbertProvider := &SentenceBERTProvider{
		model: localProvider,
	}

	ctx := context.Background()
	texts := []string{"hello", "world"}

	embeddings, err := sbertProvider.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	assert.Len(t, embeddings[0], 384)
}

func TestSentenceBERTProvider_Dimension(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 384,
	}

	localProvider, err := New(Config{
		Model:        mock,
		Dimension:    384,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	sbertProvider := &SentenceBERTProvider{
		model: localProvider,
	}

	assert.Equal(t, 384, sbertProvider.Dimension())
}

func TestModel_Close(t *testing.T) {
	model, err := NewModel(768, "test/path")
	require.NoError(t, err)

	err = model.Close()
	assert.NoError(t, err)
}

func TestLocalProvider_processBatch_3DOutput(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			inputIDs, ok := inputs["input_ids"].([][]int64)
			if !ok {
				return nil, assert.AnError
			}
			batchSize := len(inputIDs)
			seqLen := 10
			hiddenSize := 768

			embeddings := make([][][]float32, batchSize)
			for i := 0; i < batchSize; i++ {
				embeddings[i] = make([][]float32, seqLen)
				for j := 0; j < seqLen; j++ {
					embeddings[i][j] = make([]float32, hiddenSize)
					for k := 0; k < hiddenSize; k++ {
						embeddings[i][j][k] = float32(i*10000 + j*100 + k)
					}
				}
			}
			return map[string]interface{}{
				"last_hidden_state": embeddings,
			}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"hello", "world"}

	embeddings, err := provider.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 2)
	assert.Len(t, embeddings[0], 768)
}

func TestLocalProvider_processBatch_AlternativeOutputKey(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			inputIDs, ok := inputs["input_ids"].([][]int64)
			if !ok {
				return nil, assert.AnError
			}
			batchSize := len(inputIDs)
			embeddings := make([][]float32, batchSize)
			for i := 0; i < batchSize; i++ {
				embeddings[i] = make([]float32, 768)
			}
			return map[string]interface{}{
				"embeddings": embeddings,
			}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test"}

	embeddings, err := provider.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 1)
}

func TestLocalProvider_processBatch_EmptySequence(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"last_hidden_state": [][][]float32{
					{},
				},
			}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test"}

	embeddings, err := provider.Embed(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 1)
	assert.Len(t, embeddings[0], 768)
}

func TestLocalProvider_processBatch_BatchSizeMismatch(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			embeddings := make([][]float32, 5)
			for i := 0; i < 5; i++ {
				embeddings[i] = make([]float32, 768)
			}
			return map[string]interface{}{
				"last_hidden_state": embeddings,
			}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test1", "test2"}

	_, err = provider.Embed(ctx, texts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch size mismatch")
}

func TestLocalProvider_processBatch_InvalidOutputType(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{
				"last_hidden_state": "invalid output",
			}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test"}

	_, err = provider.Embed(ctx, texts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected output type")
}

func TestLocalProvider_InvalidModelOutput(t *testing.T) {
	mock := &mockModelForProvider{
		dimension: 768,
		runFunc: func(inputs map[string]interface{}) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
	}

	provider, err := New(Config{
		Model:        mock,
		Dimension:    768,
		MaxBatchSize: 32,
	})
	require.NoError(t, err)

	ctx := context.Background()
	texts := []string{"test"}

	_, err = provider.Embed(ctx, texts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
