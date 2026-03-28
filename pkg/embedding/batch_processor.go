package embedding

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
)

// BatchOptions configures the behavior of a BatchProcessor.
// These options control batching strategy, concurrency, and caching.
type BatchOptions struct {
	// MaxBatchSize is the maximum number of texts to process in a single batch.
	// Larger batches are more efficient but use more memory.
	// Default: 32
	MaxBatchSize int

	// MaxConcurrent is the maximum number of concurrent API calls.
	// Higher values increase throughput but may hit rate limits.
	// Default: 4
	MaxConcurrent int

	// MaxCacheSize is the maximum number of entries in the LRU cache.
	// When exceeded, least recently used entries are evicted.
	// Default: 10000
	MaxCacheSize int
}

// ProgressCallback is called periodically during batch processing.
// It receives the current progress and can return false to cancel processing.
//
// Parameters:
//   - current: Number of texts processed so far
//   - total: Total number of texts to process
//   - err: Error if processing failed, nil otherwise
//
// Returns true to continue processing, false to cancel
type ProgressCallback func(current, total int, err error) bool

// BatchProcessor processes multiple texts through an embedding provider
// with support for batching, concurrent processing, and LRU caching.
// It improves throughput by batching requests and reduces API calls
// by caching embeddings for repeated texts.
//
// BatchProcessor is safe for concurrent use from multiple goroutines.
//
// Example:
//
//	processor := embedding.NewBatchProcessor(provider, embedding.BatchOptions{
//	    MaxBatchSize:  32,
//	    MaxConcurrent: 4,
//	    MaxCacheSize:  10000,
//	})
//	embeddings, err := processor.Process(ctx, texts)
type BatchProcessor struct {
	provider   Provider
	options    BatchOptions
	cache      map[string]*list.Element
	cacheList  *list.List
	cacheMutex sync.Mutex
}

// cachedValue holds a cached embedding with its key for LRU tracking.
type cachedValue struct {
	key   string
	value []float32
}

// NewBatchProcessor creates a new BatchProcessor with the specified provider and options.
// Default values are applied to any options that are zero:
//
//   - MaxBatchSize: 32
//   - MaxConcurrent: 4
//   - MaxCacheSize: 10000
//
// Parameters:
//   - provider: The embedding provider to use for generating embeddings
//   - options: Configuration options (nil uses defaults)
//
// Returns a configured BatchProcessor
func NewBatchProcessor(provider Provider, options BatchOptions) *BatchProcessor {
	if options.MaxBatchSize <= 0 {
		options.MaxBatchSize = 32
	}

	if options.MaxConcurrent <= 0 {
		options.MaxConcurrent = 4
	}

	if options.MaxCacheSize <= 0 {
		options.MaxCacheSize = 10000
	}

	return &BatchProcessor{
		provider:  provider,
		options:   options,
		cache:     make(map[string]*list.Element),
		cacheList: list.New(),
	}
}

// Process generates embeddings for a list of texts.
// This is a convenience method equivalent to calling ProcessWithProgress
// with no progress callback.
//
// Parameters:
//   - ctx: Context for cancellation
//   - texts: The texts to embed
//
// Returns a 2D slice of embeddings (one per text) and any error
func (bp *BatchProcessor) Process(ctx context.Context, texts []string) ([][]float32, error) {
	return bp.ProcessWithProgress(ctx, texts, nil)
}

// ProcessWithProgress generates embeddings for a list of texts with optional progress reporting.
// The method processes texts in batches, concurrent API calls, and caches results
// to avoid recomputing embeddings for the same text.
//
// The callback, if provided, is called after each batch completes with current progress.
// Returning false from the callback cancels processing.
//
// Parameters:
//   - ctx: Context for cancellation
//   - texts: The texts to embed
//   - callback: Optional progress reporter (nil for no reporting)
//
// Returns a 2D slice of embeddings (one per text) and any error
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

			select {
			case <-innerCtx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			batchTexts := texts[startIdx:endIdx]
			batchSize := endIdx - startIdx
			batchResults := make([][]float32, batchSize)

			uncachedTexts := make([]string, 0, batchSize)
			uncachedIndices := make([]int, 0, batchSize)

			bp.cacheMutex.Lock()
			for j, text := range batchTexts {
				if elem, ok := bp.cache[text]; ok {
					batchResults[j] = elem.Value.(*cachedValue).value
					bp.cacheList.MoveToFront(elem)
				} else {
					uncachedTexts = append(uncachedTexts, text)
					uncachedIndices = append(uncachedIndices, j)
				}
			}
			bp.cacheMutex.Unlock()

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

				bp.cacheMutex.Lock()
				for j, text := range uncachedTexts {
					bp.addToCache(text, embeddings[j])
					batchResults[uncachedIndices[j]] = embeddings[j]
				}
				bp.cacheMutex.Unlock()
			}

			for j := 0; j < batchSize; j++ {
				results[startIdx+j] = batchResults[j]
			}

			newProcessed := atomic.AddInt32(&processedCount, int32(batchSize))
			if callback != nil {
				if !callback(int(newProcessed), total, nil) {
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
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return results, nil
}

// addToCache adds or updates a cached embedding.
// If the key already exists, it moves the entry to the front (most recently used).
// If the cache is full, it evicts the least recently used entry.
func (bp *BatchProcessor) addToCache(key string, value []float32) {
	if elem, ok := bp.cache[key]; ok {
		bp.cacheList.MoveToFront(elem)
		elem.Value.(*cachedValue).value = value
		return
	}

	entry := bp.cacheList.PushFront(&cachedValue{key: key, value: value})
	bp.cache[key] = entry

	if bp.cacheList.Len() > bp.options.MaxCacheSize {
		oldest := bp.cacheList.Back()
		if oldest != nil {
			cv := oldest.Value.(*cachedValue)
			delete(bp.cache, cv.key)
			bp.cacheList.Remove(oldest)
		}
	}
}

// ClearCache removes all entries from the embedding cache.
// Call this to free memory or when the underlying model changes.
func (bp *BatchProcessor) ClearCache() {
	bp.cacheMutex.Lock()
	defer bp.cacheMutex.Unlock()
	bp.cache = make(map[string]*list.Element)
	bp.cacheList.Init()
}

// CacheSize returns the current number of entries in the cache.
func (bp *BatchProcessor) CacheSize() int {
	bp.cacheMutex.Lock()
	defer bp.cacheMutex.Unlock()
	return len(bp.cache)
}
