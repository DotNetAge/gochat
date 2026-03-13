package embedding

import (
	"context"
	"sync"
	"sync/atomic"
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
	provider   Provider
	options    BatchOptions
	cache      map[string][]float32
	cacheMutex sync.RWMutex
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
		provider: provider,
		options:  options,
		cache:    make(map[string][]float32),
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

	results := make([][]float32, total)
	var (
		processedCount int32
		once           sync.Once
		processErr     error
		wg             sync.WaitGroup
	)

	// Create a derived context that we can cancel on first error
	innerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, bp.options.MaxConcurrent)

	for i := 0; i < total; i += bp.options.MaxBatchSize {
		end := i + bp.options.MaxBatchSize
		if end > total {
			end = total
		}

		wg.Add(1)
		go func(startIdx, endIdx int) {
			defer wg.Done()

			// Check if already cancelled
			select {
			case <-innerCtx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			batchTexts := texts[startIdx:endIdx]
			batchSize := endIdx - startIdx
			batchResults := make([][]float32, batchSize)

			// 1. Check cache (Read Lock)
			uncachedTexts := make([]string, 0, batchSize)
			uncachedIndices := make([]int, 0, batchSize)

			bp.cacheMutex.RLock()
			for j, text := range batchTexts {
				if embedding, ok := bp.cache[text]; ok {
					batchResults[j] = embedding
				} else {
					uncachedTexts = append(uncachedTexts, text)
					uncachedIndices = append(uncachedIndices, j)
				}
			}
			bp.cacheMutex.RUnlock()

			// 2. Process uncached texts
			if len(uncachedTexts) > 0 {
				embeddings, err := bp.provider.Embed(innerCtx, uncachedTexts)
				if err != nil {
					once.Do(func() {
						processErr = err
						cancel()
					})
					if callback != nil {
						callback(int(atomic.LoadInt32(&processedCount)), total, err)
					}
					return
				}

				// Update cache and results (Write Lock)
				bp.cacheMutex.Lock()
				for j, text := range uncachedTexts {
					bp.cache[text] = embeddings[j]
					batchResults[uncachedIndices[j]] = embeddings[j]
				}
				bp.cacheMutex.Unlock()
			}

			// 3. Fill into final result slice (Direct access as slices are pre-allocated and indices are unique)
			for j := 0; j < batchSize; j++ {
				results[startIdx+j] = batchResults[j]
			}

			// 4. Update progress
			newProcessed := atomic.AddInt32(&processedCount, int32(batchSize))
			if callback != nil {
				if !callback(int(newProcessed), total, nil) {
					// Signal other goroutines to stop via context
					cancel()
					return
				}
			}
		}(i, end)
	}

	wg.Wait()

	if processErr != nil {
		return nil, processErr
	}
	// Final check for external cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return results, nil
}
