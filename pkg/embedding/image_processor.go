package embedding

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

// ImageProcessor handles image preprocessing for vision models like CLIP.
type ImageProcessor struct {
	TargetSize int
	Mean       [3]float32
	Std        [3]float32
}

// NewImageProcessor creates an image processor matching CLIP default preprocessing.
func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		TargetSize: 224, // Standard input size for CLIP ViT-B/32
		Mean:       [3]float32{0.48145466, 0.4578275, 0.40821073},
		Std:        [3]float32{0.26862954, 0.26130258, 0.27577711},
	}
}

// ProcessBatch decodes, resizes, crops, and normalizes a batch of images.
// Returns a 4D float32 slice matching ONNX `pixel_values` shape [Batch][Channel][Height][Width].
func (p *ImageProcessor) ProcessBatch(images [][]byte) ([][][][]float32, error) {
	batchSize := len(images)
	pixelValues := make([][][][]float32, batchSize)

	for i, imgData := range images {
		img, _, err := image.Decode(bytes.NewReader(imgData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image %d: %w", i, err)
		}

		pixelValues[i] = p.processSingle(img)
	}

	return pixelValues, nil
}

// processSingle resizes, center crops, and normalizes a single image.
func (p *ImageProcessor) processSingle(img image.Image) [][][]float32 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// 1. Resize shortest edge to TargetSize
	var newW, newH int
	if w < h {
		newW = p.TargetSize
		newH = int(float32(h) * (float32(p.TargetSize) / float32(w)))
	} else {
		newH = p.TargetSize
		newW = int(float32(w) * (float32(p.TargetSize) / float32(h)))
	}

	// Simple nearest-neighbor resize to avoid external dependencies
	resized := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := int(float32(x) * (float32(w) / float32(newW)))
			srcY := int(float32(y) * (float32(h) / float32(newH)))
			resized.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	// 2. Center crop to TargetSize x TargetSize
	cropX := (newW - p.TargetSize) / 2
	cropY := (newH - p.TargetSize) / 2

	// 3. Normalize and convert to C, H, W
	result := make([][][]float32, 3)
	for c := 0; c < 3; c++ {
		result[c] = make([][]float32, p.TargetSize)
		for y := 0; y < p.TargetSize; y++ {
			result[c][y] = make([]float32, p.TargetSize)
		}
	}

	for y := 0; y < p.TargetSize; y++ {
		for x := 0; x < p.TargetSize; x++ {
			r, g, b, _ := resized.At(bounds.Min.X+cropX+x, bounds.Min.Y+cropY+y).RGBA()
			// RGBA() returns 0-65535, divide by 255.0 to get roughly 0-255 scale
			rf := float32(r>>8) / 255.0
			gf := float32(g>>8) / 255.0
			bf := float32(b>>8) / 255.0

			result[0][y][x] = (rf - p.Mean[0]) / p.Std[0]
			result[1][y][x] = (gf - p.Mean[1]) / p.Std[1]
			result[2][y][x] = (bf - p.Mean[2]) / p.Std[2]
		}
	}

	return result
}
