// Package downloader provides functionality for downloading embedding models from remote sources.
// It supports progress tracking, caching, and multi-file downloads.
package embedding

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DownloadModelInfo contains information about a model available for download.
type DownloadModelInfo struct {
	Name        string   // Model name
	Type        string   // Model type (e.g., "bge", "sentence-bert")
	URLs        []string // URLs of files to download
	Size        string   // Approximate total size
	Description string   // Model description
}

// DownloadProgressCallback is a callback function for tracking download progress.
//
// Parameters:
// - modelName: Name of the model being downloaded
// - fileName: Name of the file being downloaded
// - downloaded: Number of bytes downloaded so far
// - total: Total number of bytes to download (0 if unknown)
type DownloadProgressCallback func(modelName, fileName string, downloaded, total int64)

// Downloader handles the downloading and caching of embedding models.
type Downloader struct {
	cacheDir string
}

// NewDownloader creates a new downloader with the specified cache directory.
//
// Parameters:
// - cacheDir: Directory to cache downloaded models (empty for default)
//
// Returns:
// - *Downloader: A new downloader instance
func NewDownloader(cacheDir string) *Downloader {
	if cacheDir == "" {
		// Use default cache directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			cacheDir = "./models"
		} else {
			cacheDir = filepath.Join(homeDir, ".gochat", "models")
		}
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create cache directory: %v\n", err)
	}

	return &Downloader{
		cacheDir: cacheDir,
	}
}

// GetModelInfo returns information about available models
func (d *Downloader) GetModelInfo() []DownloadModelInfo {
	return []DownloadModelInfo{
		{
			Name: "bge-small-zh-v1.5",
			Type: "bge",
			URLs: []string{
				"https://huggingface.co/BAAI/bge-small-zh-v1.5/resolve/main/model.onnx",
				"https://huggingface.co/BAAI/bge-small-zh-v1.5/resolve/main/tokenizer.json",
			},
			Size:        "~100MB",
			Description: "Chinese BGE small model for embedding",
		},
		{
			Name: "all-MiniLM-L6-v2",
			Type: "sentence-bert",
			URLs: []string{
				"https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/model.onnx",
				"https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/tokenizer.json",
			},
			Size:        "~80MB",
			Description: "English Sentence-BERT model for embedding",
		},
		{
			Name: "bert-base-uncased",
			Type: "bert",
			URLs: []string{
				"https://huggingface.co/bert-base-uncased/resolve/main/model.onnx",
				"https://huggingface.co/bert-base-uncased/resolve/main/tokenizer.json",
			},
			Size:        "~400MB",
			Description: "English BERT base model for embedding",
		},
		{
			Name: "bge-base-zh-v1.5",
			Type: "bge",
			URLs: []string{
				"https://huggingface.co/BAAI/bge-base-zh-v1.5/resolve/main/model.onnx",
				"https://huggingface.co/BAAI/bge-base-zh-v1.5/resolve/main/tokenizer.json",
			},
			Size:        "~400MB",
			Description: "Chinese BGE base model for embedding",
		},
		{
			Name: "all-mpnet-base-v2",
			Type: "sentence-bert",
			URLs: []string{
				"https://huggingface.co/sentence-transformers/all-mpnet-base-v2/resolve/main/model.onnx",
				"https://huggingface.co/sentence-transformers/all-mpnet-base-v2/resolve/main/tokenizer.json",
			},
			Size:        "~400MB",
			Description: "English MPNet base model for embedding",
		},
	}
}

// DownloadModel downloads a model by name
func (d *Downloader) DownloadModel(modelName string, callback DownloadProgressCallback) (string, error) {
	// Get model info
	modelInfo, err := d.getModelInfoByName(modelName)
	if err != nil {
		return "", err
	}

	// Create model directory
	modelDir := filepath.Join(d.cacheDir, modelName)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create model directory: %w", err)
	}

	// Check if all files already exist
	allFilesExist := true
	for _, url := range modelInfo.URLs {
		fileName := filepath.Base(url)
		filePath := filepath.Join(modelDir, fileName)
		if _, err := os.Stat(filePath); err != nil {
			allFilesExist = false
			break
		}
	}

	if allFilesExist {
		return modelDir, nil
	}

	// Download model files
	fmt.Printf("Downloading model: %s (%s)\n", modelName, modelInfo.Size)
	fmt.Printf("Files to download: %d\n", len(modelInfo.URLs))

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	// Download each file
	for i, url := range modelInfo.URLs {
		fileName := filepath.Base(url)
		filePath := filepath.Join(modelDir, fileName)

		fmt.Printf("Downloading file %d/%d: %s\n", i+1, len(modelInfo.URLs), fileName)

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			fmt.Printf("File already exists, skipping: %s\n", fileName)
			continue
		}

		// Send request
		resp, err := client.Get(url)
		if err != nil {
			return "", fmt.Errorf("failed to download file %s: %w", fileName, err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to download file %s: status code %d", fileName, resp.StatusCode)
		}

		// Get content length for progress tracking
		contentLength := resp.ContentLength

		// Create temporary file for atomic download
		tmpPath := filePath + ".tmp"
		dst, err := os.Create(tmpPath)
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file: %w", err)
		}

		// Copy content with progress tracking
		var downloaded int64
		reader := &progressReader{
			reader: resp.Body,
			total:  contentLength,
			onProgress: func(n int64) {
				downloaded = n
				if callback != nil {
					callback(modelName, fileName, downloaded, contentLength)
				}
			},
		}

		_, err = io.Copy(dst, reader)
		dst.Close()

		if err != nil {
			// Clean up partial file
			os.Remove(tmpPath)
			return "", fmt.Errorf("failed to save file %s: %w", fileName, err)
		}

		// Atomically move temporary file to final destination
		if err := os.Rename(tmpPath, filePath); err != nil {
			return "", fmt.Errorf("failed to rename temporary file: %w", err)
		}

		fmt.Printf("Download completed: %s\n", fileName)
	}

	fmt.Printf("All files downloaded to: %s\n", modelDir)
	return modelDir, nil
}

// progressReader wraps a reader to track progress
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	onProgress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		if r.onProgress != nil {
			r.onProgress(r.downloaded)
		}
	}
	return n, err
}

// getModelInfoByName returns model info by name
func (d *Downloader) getModelInfoByName(modelName string) (DownloadModelInfo, error) {
	for _, info := range d.GetModelInfo() {
		if info.Name == modelName {
			return info, nil
		}
	}
	return DownloadModelInfo{}, fmt.Errorf("model not found: %s", modelName)
}
