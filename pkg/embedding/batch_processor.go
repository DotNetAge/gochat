// Package embedding provides interfaces and implementations for text embedding generation.
// It supports both local and remote embedding models with batch processing and caching capabilities.
package embedding

import (
	"context"
	"sync"
)

// BatchOptions contains configuration options for batch processing.
type BatchOptions struct {
	MaxBatchSize  int // Maximum batch size for embedding generation
	MaxConcurrent int // Maximum number of concurrent batches
}

// ProgressCallback is a callback function for tracking batch processing progress.
//
// Parameters:
// - current: Number of texts processed so far
// - total: Total number of texts to process
// - error: Error if any occurred during processing
//
// Returns:
// - bool: True to continue processing, false to cancel the operation
type ProgressCallback func(current, total int, err error) bool

// BatchProcessor handles batch processing of embeddings with optimization features
// including caching, concurrent processing, and progress tracking.
type BatchProcessor struct {
	provider      Provider
	options       BatchOptions
	cache         map[string][]float32
	cacheMutex    sync.RWMutex
}

// NewBatchProcessor creates a new batch processor with the given provider and options.
//
// Parameters:
// - provider: The embedding provider to use for generating embeddings
// - options: Configuration options for batch processing
//
// Returns:
// - *BatchProcessor: A new batch processor instance
func NewBatchProcessor(provider Provider, options BatchOptions) *BatchProcessor {
	// Set default values if not provided
	if options.MaxBatchSize <= 0 {
		options.MaxBatchSize = 32
	}

	if options.MaxConcurrent <= 0 {
		options.MaxConcurrent = 4
	}

	return &BatchProcessor{
		provider:  provider,
		options:   options,
		cache:     make(map[string][]float32),
	}
}

// Process processes embeddings with batch processing optimization
func (bp *BatchProcessor) Process(ctx context.Context, texts []string) ([][]float32, error) {
	return bp.ProcessWithProgress(ctx, texts, nil)
}

// ProcessWithProgress processes embeddings with progress tracking
func (bp *BatchProcessor) ProcessWithProgress(
	ctx context.Context,
	texts []string,
	callback ProgressCallback,
) ([][]float32, error) {
	total := len(texts)
	if total == 0 {
		return [][]float32{}, nil
	}

	// Initialize result
	results := make([][]float32, total)
	var mu sync.Mutex
	var wg sync.WaitGroup
	var processErr error

	// Semaphore for concurrent batches
	sem := make(chan struct{}, bp.options.MaxConcurrent)

	// Process in batches
	for i := 0; i < total; i += bp.options.MaxBatchSize {
		start := i
		end := i + bp.options.MaxBatchSize
		if end > total {
			end = total
		}

		batch := texts[start:end]
		batchIndices := make([]int, end-start)
		for j := range batchIndices {
			batchIndices[j] = start + j
		}

		wg.Add(1)
		go func(batchTexts []string, indices []int) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				if processErr == nil {
					processErr = ctx.Err()
				}
				mu.Unlock()
				return
			}

			// Check cache first
			cachedIndices := make([]int, 0)
			cachedEmbeddings := make([][]float32, 0)
			uncachedTexts := make([]string, 0)
			uncachedIndices := make([]int, 0)

			bp.cacheMutex.RLock()
			for j, text := range batchTexts {
				if embedding, ok := bp.cache[text]; ok {
					cachedIndices = append(cachedIndices, j)
					cachedEmbeddings = append(cachedEmbeddings, embedding)
				} else {
					uncachedTexts = append(uncachedTexts, text)
					uncachedIndices = append(uncachedIndices, j)
				}
			}
			bp.cacheMutex.RUnlock()

			// Process uncached texts
			var uncachedEmbeddings [][]float32
			var err error

			if len(uncachedTexts) > 0 {
				uncachedEmbeddings, err = bp.provider.Embed(ctx, uncachedTexts)
				if err != nil {
					if callback != nil {
						callback(0, total, err)
					}
					mu.Lock()
					if processErr == nil {
						processErr = err
					}
					mu.Unlock()
					return
				}

				// Update cache
				bp.cacheMutex.Lock()
				for j, text := range uncachedTexts {
					bp.cache[text] = uncachedEmbeddings[j]
				}
				bp.cacheMutex.Unlock()
			}

			// Combine results
			batchResults := make([][]float32, len(batchTexts))

			// Fill cached results
			for j, idx := range cachedIndices {
				batchResults[idx] = cachedEmbeddings[j]
			}

			// Fill uncached results
			for j, idx := range uncachedIndices {
				batchResults[idx] = uncachedEmbeddings[j]
			}

			// Update final results
			mu.Lock()
			for j, idx := range indices {
				results[idx] = batchResults[j]
			}
			mu.Unlock()

			// Update progress
			if callback != nil {
				processed := end
				if processed > total {
					processed = total
				}
				if !callback(processed, total, nil) {
					return
				}
			}
		}(batch, batchIndices)
	}

	wg.Wait()
	return results, processErr
}
