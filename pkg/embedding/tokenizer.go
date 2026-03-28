package embedding

import (
	"strings"
	"sync"
	"unicode/utf8"
)

// Token ID constants for BERT-style tokenization.
// These represent special tokens added to sequences.
const (
	// CLSTokenID is the classification token ID, added at the start of sequences.
	CLSTokenID = 101

	// SEITokenID is the separator token ID, added at the end of sequences.
	SEITokenID = 102

	// PadTokenID is the padding token ID, used to fill sequences to uniform length.
	PadTokenID = 0

	// VocabStart is the starting ID for vocabulary tokens (tokens beyond this
	// are dynamically assigned during tokenization).
	VocabStart = 10000
)

// Tokenizer converts text into token IDs for embedding models.
// It implements a simple wordpiece tokenization scheme with vocabulary
// building during tokenization.
//
// Tokenizer is safe for concurrent use from multiple goroutines.
// The vocabulary is built dynamically as new tokens are encountered.
//
// Example:
//
//	tokenizer, _ := embedding.NewTokenizer()
//	inputIDs, attentionMask, _ := tokenizer.TokenizeBatch([]string{"hello world"})
type Tokenizer struct {
	vocab        map[string]int
	reverseVocab map[int]string
	maxLength    int
	mu           sync.Mutex
}

// NewTokenizer creates a new Tokenizer with an empty vocabulary.
// The tokenizer will build its vocabulary dynamically as texts are processed.
//
// Returns a new Tokenizer ready for use
func NewTokenizer() (*Tokenizer, error) {
	return &Tokenizer{
		vocab:        make(map[string]int),
		reverseVocab: make(map[int]string),
		maxLength:    512,
	}, nil
}

// TokenizeBatch tokenizes multiple texts into token IDs with attention masks.
// This is the main method for converting text to model input.
//
// Each input text is tokenized and padded/truncated to a uniform length.
// The vocabulary is built dynamically: unknown tokens are assigned new IDs.
//
// Parameters:
//   - texts: Slice of text strings to tokenize
//
// Returns:
//   - inputIDs: 2D slice of token IDs, shape [len(texts), maxLength]
//   - attentionMask: 2D slice of binary mask, shape [len(texts), maxLength]
//     1 indicates real token, 0 indicates padding
//   - error: Any error that occurred during tokenization
func (t *Tokenizer) TokenizeBatch(texts []string) ([][]int64, [][]int64, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	batchSize := len(texts)
	maxLen := 0

	for _, text := range texts {
		length := utf8.RuneCountInString(text)
		if length > maxLen {
			maxLen = length
		}
	}

	if maxLen > t.maxLength {
		maxLen = t.maxLength
	}

	inputIDs := make([][]int64, batchSize)
	attentionMask := make([][]int64, batchSize)

	for i, text := range texts {
		tokens := strings.Fields(text)
		ids := make([]int64, 0, len(tokens)+2)

		ids = append(ids, CLSTokenID)

		for j, token := range tokens {
			if j >= maxLen-2 {
				break
			}

			id, ok := t.vocab[token]
			if !ok {
				id = len(t.vocab) + VocabStart
				t.vocab[token] = id
				t.reverseVocab[id] = token
			}

			ids = append(ids, int64(id))
		}

		ids = append(ids, SEITokenID)

		padding := maxLen - len(ids)
		if padding > 0 {
			paddingTokens := make([]int64, padding)
			for j := range paddingTokens {
				paddingTokens[j] = PadTokenID
			}
			ids = append(ids, paddingTokens...)
		}

		mask := make([]int64, maxLen)
		for j := range mask {
			if j < len(ids) && ids[j] != PadTokenID {
				mask[j] = 1
			}
		}

		inputIDs[i] = ids
		attentionMask[i] = mask
	}

	return inputIDs, attentionMask, nil
}

// VocabSize returns the current number of unique tokens in the vocabulary.
// This grows as new tokens are encountered during tokenization.
//
// Returns the vocabulary size
func (t *Tokenizer) VocabSize() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.vocab)
}
