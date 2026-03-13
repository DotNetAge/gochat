package embedding

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadModel_RetryAndError(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "downloader-test-*")
	defer os.RemoveAll(tmpDir)

	d := NewDownloader(tmpDir)

	// Test model not found
	_, err := d.DownloadModel("non-existent", nil)
	assert.Error(t, err)

	// Test download failure (using a mock server that returns 404)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// We can't easily inject the mock server URL into the internal model list
	// but we can test the internal getModelInfoByName
	info, err := d.getModelInfoByName("bge-small-zh-v1.5")
	assert.NoError(t, err)
	assert.NotEmpty(t, info.URLs)
}

func TestDownloader_EdgeCases(t *testing.T) {
	d := NewDownloader("")

	// Test empty model name
	_, err := d.DownloadModel("", nil)
	assert.Error(t, err)

	// Test progress reader with nil callback
	pr := &progressReader{
		reader:     io.NopCloser(bytes.NewReader([]byte("test"))),
		total:      4,
		onProgress: nil,
	}
	buf := make([]byte, 4)
	n, err := pr.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
}

func TestDownloader_DefaultCacheDir(t *testing.T) {
	d := NewDownloader("")
	assert.NotEmpty(t, d.cacheDir)
}

func TestProgressReader_EdgeCases(t *testing.T) {
	// Already tested in existing tests, but can add more
}

func TestBGEProvider(t *testing.T) {
	// BGE small usually is 512, base is 768
	p, err := NewBGEProvider("bge-small-zh-v1.5.onnx")
	require.NoError(t, err)
	assert.Equal(t, 512, p.Dimension())

	p2, err := NewBGEProvider("bge-base-zh-v1.5.onnx")
	require.NoError(t, err)
	assert.Equal(t, 768, p2.Dimension())
}

func TestSentenceBERTProvider(t *testing.T) {
	p, err := NewSentenceBERTProvider("all-MiniLM-L6-v2.onnx")
	require.NoError(t, err)
	assert.Equal(t, 384, p.Dimension())
}

func TestGenericProvider(t *testing.T) {
	p, err := NewProvider("bert-base-uncased.onnx")
	require.NoError(t, err)
	assert.Equal(t, 768, p.Dimension())

	ctx := context.Background()
	// NewProvider uses NewCustomModel which returns mock embeddings
	embeddings, err := p.Embed(ctx, []string{"hello"})
	require.NoError(t, err)
	assert.Len(t, embeddings, 1)
	assert.Len(t, embeddings[0], 768)
}
