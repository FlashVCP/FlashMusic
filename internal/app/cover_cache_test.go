package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

func TestCoverDiskCache(t *testing.T) {
	testURL := "https://example.com/cover/test_album_123.jpg"

	// 1. Initial state: cache miss
	_, ok := readCoverDiskCache(testURL)
	if ok {
		t.Log("Note: cache file already existed from previous run")
	}

	// 2. Generate a valid 32x32 test PNG
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 34, G: 197, B: 94, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	testData := buf.Bytes()

	// 3. Write to disk cache
	writeCoverDiskCache(testURL, testData)

	// 4. Verify cache hit
	cached, ok := readCoverDiskCache(testURL)
	if !ok {
		t.Fatalf("expected cache hit for %s", testURL)
	}
	if !bytes.Equal(cached, testData) {
		t.Fatalf("cached data does not match original data: len %d vs %d", len(cached), len(testData))
	}

	// 5. Test decodeArt with cached image
	bmp, decodedImg, err := decodeArt(cached)
	if err != nil {
		t.Fatalf("decodeArt on cached data failed: %v", err)
	}
	if bmp == nil || decodedImg == nil {
		t.Fatal("expected non-nil bitmap and image")
	}
	if decodedImg.Bounds().Dx() != 32 || decodedImg.Bounds().Dy() != 32 {
		t.Fatalf("unexpected dimensions: %v", decodedImg.Bounds())
	}

	// Cleanup test file
	path := urlCachePath(testURL)
	_ = os.Remove(path)
}
