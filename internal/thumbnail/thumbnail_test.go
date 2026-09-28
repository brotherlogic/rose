package thumbnail

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/HugoSmits86/nativewebp"
)

func createSyntheticJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("failed to create synthetic JPEG: %v", err)
	}
	return buf.Bytes()
}

func createSyntheticPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to create synthetic PNG: %v", err)
	}
	return buf.Bytes()
}

func verifyWebPHeader(t *testing.T, data []byte) {
	t.Helper()
	if len(data) < 12 {
		t.Fatalf("output data too short for WebP: %d bytes", len(data))
	}
	if string(data[0:4]) != "RIFF" {
		t.Errorf("expected WebP RIFF magic, got %q", string(data[0:4]))
	}
	if string(data[8:12]) != "WEBP" {
		t.Errorf("expected WebP format header, got %q", string(data[8:12]))
	}
}

func TestGenerateThumbnail_ValidJPEGAndPNG(t *testing.T) {
	jpegData := createSyntheticJPEG(t, 800, 600)
	thumbJPEG, err := GenerateThumbnail(jpegData)
	if err != nil {
		t.Fatalf("unexpected error generating thumbnail for JPEG: %v", err)
	}
	verifyWebPHeader(t, thumbJPEG)

	cfgJPEG, err := nativewebp.DecodeConfig(bytes.NewReader(thumbJPEG))
	if err != nil {
		t.Fatalf("failed to decode generated WebP config for JPEG: %v", err)
	}
	if cfgJPEG.Width != 600 || cfgJPEG.Height != 450 {
		t.Errorf("expected JPEG thumbnail dimensions 600x450, got %dx%d", cfgJPEG.Width, cfgJPEG.Height)
	}

	pngData := createSyntheticPNG(t, 800, 600)
	thumbPNG, err := GenerateThumbnail(pngData)
	if err != nil {
		t.Fatalf("unexpected error generating thumbnail for PNG: %v", err)
	}
	verifyWebPHeader(t, thumbPNG)

	cfgPNG, err := nativewebp.DecodeConfig(bytes.NewReader(thumbPNG))
	if err != nil {
		t.Fatalf("failed to decode generated WebP config for PNG: %v", err)
	}
	if cfgPNG.Width != 600 || cfgPNG.Height != 450 {
		t.Errorf("expected PNG thumbnail dimensions 600x450, got %dx%d", cfgPNG.Width, cfgPNG.Height)
	}
}

func TestGenerateThumbnail_AspectRatioMatrix(t *testing.T) {
	tests := []struct {
		name       string
		srcWidth   int
		srcHeight  int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "2400x120 banner scales to 600x30",
			srcWidth:   2400,
			srcHeight:  120,
			wantWidth:  600,
			wantHeight: 30,
		},
		{
			name:       "100x3000 panorama scales to 20x600",
			srcWidth:   100,
			srcHeight:  3000,
			wantWidth:  20,
			wantHeight: 600,
		},
		{
			name:       "1200x800 image scales to 600x400",
			srcWidth:   1200,
			srcHeight:  800,
			wantWidth:  600,
			wantHeight: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createSyntheticPNG(t, tt.srcWidth, tt.srcHeight)
			thumb, err := GenerateThumbnail(data)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			verifyWebPHeader(t, thumb)

			cfg, err := nativewebp.DecodeConfig(bytes.NewReader(thumb))
			if err != nil {
				t.Fatalf("failed to decode WebP config: %v", err)
			}
			if cfg.Width != tt.wantWidth || cfg.Height != tt.wantHeight {
				t.Errorf("expected %dx%d, got %dx%d", tt.wantWidth, tt.wantHeight, cfg.Width, cfg.Height)
			}
		})
	}
}

func TestGenerateThumbnail_Sub600Retention(t *testing.T) {
	// 300x200 image retains 300x200 resolution without upscaling
	data := createSyntheticPNG(t, 300, 200)
	thumb, err := GenerateThumbnail(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	verifyWebPHeader(t, thumb)

	cfg, err := nativewebp.DecodeConfig(bytes.NewReader(thumb))
	if err != nil {
		t.Fatalf("failed to decode WebP config: %v", err)
	}
	if cfg.Width != 300 || cfg.Height != 200 {
		t.Errorf("expected retained sub-600px dimensions 300x200, got %dx%d", cfg.Width, cfg.Height)
	}
}

func TestGenerateThumbnail_EdgeCasesAndErrors(t *testing.T) {
	t.Run("empty byte slice", func(t *testing.T) {
		_, err := GenerateThumbnail([]byte{})
		if err == nil {
			t.Fatal("expected error for empty byte slice, got nil")
		}
		if !strings.Contains(err.Error(), "empty image payload") {
			t.Errorf("expected 'empty image payload' error message, got %v", err)
		}
	})

	t.Run("corrupted or truncated bytes", func(t *testing.T) {
		_, err := GenerateThumbnail([]byte{0x01, 0x02, 0x03, 0x04})
		if err == nil {
			t.Fatal("expected error for corrupted bytes, got nil")
		}
	})

	t.Run("image dimensions exceeding MaxDimensionLimit width", func(t *testing.T) {
		data := createSyntheticPNG(t, MaxDimensionLimit+1, 10)
		_, err := GenerateThumbnail(data)
		if err == nil {
			t.Fatal("expected error for dimensions exceeding MaxDimensionLimit, got nil")
		}
	})

	t.Run("image dimensions exceeding MaxDimensionLimit height", func(t *testing.T) {
		data := createSyntheticPNG(t, 10, MaxDimensionLimit+1)
		_, err := GenerateThumbnail(data)
		if err == nil {
			t.Fatal("expected error for dimensions exceeding MaxDimensionLimit, got nil")
		}
	})
}
