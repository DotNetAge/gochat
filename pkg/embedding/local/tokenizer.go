// Package local provides implementations for local embedding model providers.
// It supports various model formats and includes a tokenizer for text preprocessing.
package local

import (
	"strings"
	"unicode/utf8"
)

// Tokenizer handles text tokenization for embedding models.
// It converts raw text into token IDs that can be processed by embedding models.
type Tokenizer struct {
	vocab       map[string]int
	reverseVocab map[int]string
	maxLength   int
}

// NewTokenizer creates a new tokenizer instance.
//
// Note: This is a simple whitespace tokenizer for demonstration purposes.
// In a production implementation, you would use a proper tokenizer like BPE or WordPiece.
//
// Returns:
// - *Tokenizer: A new tokenizer instance
// - error: Error if initialization fails
func NewTokenizer() (*Tokenizer, error) {
	return &Tokenizer{
		vocab:       make(map[string]int),
		reverseVocab: make(map[int]string),
		maxLength:   512,
	}, nil
}

// TokenizeBatch tokenizes a batch of texts
func (t *Tokenizer) TokenizeBatch(texts []string) ([][]int64, [][]int64, error) {
	batchSize := len(texts)
	maxLen := 0

	// Find maximum length in batch
	for _, text := range texts {
		length := utf8.RuneCountInString(text)
		if length > maxLen {
			maxLen = length
		}
	}

	// Cap at maxLength
	if maxLen > t.maxLength {
		maxLen = t.maxLength
	}

	// Initialize input IDs and attention mask
	inputIDs := make([][]int64, batchSize)
	attentionMask := make([][]int64, batchSize)

	for i, text := range texts {
		// Simple whitespace tokenization
		tokens := strings.Fields(text)
		ids := make([]int64, 0, len(tokens)+2) // +2 for [CLS] and [SEP]

		// Add [CLS] token
		ids = append(ids, 101)

		// Add tokens
		for j, token := range tokens {
			if j >= maxLen-2 { // Reserve space for [CLS] and [SEP]
				break
			}

			// Get or assign token ID
			id, ok := t.vocab[token]
			if !ok {
				// Assign new ID
				id = len(t.vocab) + 10000 // Start from 10000 to avoid reserved IDs
				t.vocab[token] = id
				t.reverseVocab[id] = token
			}

			ids = append(ids, int64(id))
		}

		// Add [SEP] token
		ids = append(ids, 102)

		// Pad to maxLen
		padding := maxLen - len(ids)
		if padding > 0 {
			paddingTokens := make([]int64, padding)
			for j := range paddingTokens {
				paddingTokens[j] = 0 // [PAD] token
			}
			ids = append(ids, paddingTokens...)
		}

		// Create attention mask
		mask := make([]int64, maxLen)
		for j := range mask {
			if j < len(ids) && ids[j] != 0 {
				mask[j] = 1
			}
		}

		inputIDs[i] = ids
		attentionMask[i] = mask
	}

	return inputIDs, attentionMask, nil
}
