package embedding

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBatchProcessor_LRUCache(t *testing.T) {
	mockProvider := &MockProvider{
		DimensionValue: 128,
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize:  2,
		MaxConcurrent: 1,
		MaxCacheSize:  3,
	})

	ctx := context.Background()
	texts := []string{"text1", "text2", "text3", "text4"}

	embeddings, err := processor.Process(ctx, texts)
	assert.NoError(t, err)
	assert.Len(t, embeddings, 4)

	assert.Equal(t, 3, processor.CacheSize())
}

func TestBatchProcessor_ClearCache(t *testing.T) {
	mockProvider := &MockProvider{
		DimensionValue: 128,
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize: 2,
	})

	ctx := context.Background()
	texts := []string{"text1", "text2"}

	_, err := processor.Process(ctx, texts)
	assert.NoError(t, err)

	assert.Greater(t, processor.CacheSize(), 0)

	processor.ClearCache()
	assert.Equal(t, 0, processor.CacheSize())
}

func TestBatchProcessor_CacheConcurrency(t *testing.T) {
	mockProvider := &MockProvider{
		DimensionValue: 128,
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize:  10,
		MaxConcurrent: 4,
		MaxCacheSize:  100,
	})

	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			texts := []string{"concurrent1", "concurrent2", "concurrent3"}
			_, err := processor.Process(ctx, texts)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.Greater(t, processor.CacheSize(), 0)
}

func TestBatchProcessor_CacheHit(t *testing.T) {
	callCount := 0
	mockProvider := &MockProvider{
		DimensionValue: 128,
		EmbedFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
			callCount++
			embeddings := make([][]float32, len(texts))
			for i := range embeddings {
				embeddings[i] = make([]float32, 128)
			}
			return embeddings, nil
		},
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize:  10,
		MaxConcurrent:  1,
		MaxCacheSize:  100,
	})

	ctx := context.Background()
	sameText := "same text to cache"

	_, err := processor.Process(ctx, []string{sameText})
	assert.NoError(t, err)
	firstCallCount := callCount

	_, err = processor.Process(ctx, []string{sameText})
	assert.NoError(t, err)
	secondCallCount := callCount

	assert.Equal(t, firstCallCount, secondCallCount)
}

func TestBatchProcessor_MaxCacheSize(t *testing.T) {
	mockProvider := &MockProvider{
		DimensionValue: 64,
	}

	processor := NewBatchProcessor(mockProvider, BatchOptions{
		MaxBatchSize: 1,
		MaxCacheSize: 5,
	})

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		texts := []string{string(rune('a' + i))}
		_, err := processor.Process(ctx, texts)
		assert.NoError(t, err)
	}

	assert.LessOrEqual(t, processor.CacheSize(), 5)
}
