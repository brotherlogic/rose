package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"

	"github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/draw"
)

const (
	// MaxDimensionLimit defends against decompression exhaustion attacks.
	MaxDimensionLimit = 10000

	// TargetMaxDimension is the maximum width or height bounding box for thumbnails.
	TargetMaxDimension = 600
)

// GenerateThumbnail decodes an image payload (JPEG/PNG), enforces dimension safety guards,
// downscales the image to fit within a 600x600 bounding box while strictly preserving aspect ratio
// without upscaling, and encodes the output to WebP format.
func GenerateThumbnail(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty image payload")
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", cfg.Width, cfg.Height)
	}

	if cfg.Width > MaxDimensionLimit || cfg.Height > MaxDimensionLimit {
		return nil, fmt.Errorf("image dimension exceeds maximum limit %d: %dx%d", MaxDimensionLimit, cfg.Width, cfg.Height)
	}

	srcImg, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var targetImg image.Image = srcImg
	if cfg.Width > TargetMaxDimension || cfg.Height > TargetMaxDimension {
		scale := math.Min(float64(TargetMaxDimension)/float64(cfg.Width), float64(TargetMaxDimension)/float64(cfg.Height))
		targetW := int(math.Round(float64(cfg.Width) * scale))
		targetH := int(math.Round(float64(cfg.Height) * scale))
		if targetW < 1 {
			targetW = 1
		}
		if targetH < 1 {
			targetH = 1
		}

		dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
		draw.BiLinear.Scale(dst, dst.Bounds(), srcImg, srcImg.Bounds(), draw.Over, nil)
		targetImg = dst
	}

	var out bytes.Buffer
	if err := nativewebp.Encode(&out, targetImg, nil); err != nil {
		return nil, fmt.Errorf("failed to encode webp thumbnail: %w", err)
	}

	return out.Bytes(), nil
}
