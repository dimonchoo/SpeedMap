package optimizer

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

func ExtractFilenameFromURL(rawURL string) string {
	parts := strings.Split(rawURL, "?")
	cleanPath := parts[0]
	base := filepath.Base(cleanPath)
	if base == "" || base == "." || base == "/" {
		base = "image"
	}
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	if nameWithoutExt == "" {
		nameWithoutExt = "image"
	}
	return nameWithoutExt + ".webp"
}

// ExtractOriginalFilename preserves the original file extension (e.g. .svg, .png, .jpg)
func ExtractOriginalFilename(rawURL string) string {
	parts := strings.Split(rawURL, "?")
	cleanPath := parts[0]
	base := filepath.Base(cleanPath)
	if base == "" || base == "." || base == "/" {
		base = "image"
	}
	return base
}

// CreateZIPArchive compresses multiple WebP conversion results into a single .zip archive byte slice
func CreateZIPArchive(results []*ConversionResult) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	usedNames := make(map[string]int)

	for _, res := range results {
		if res == nil || res.OptimizedWebPBase64 == "" {
			continue
		}

		idx := strings.Index(res.OptimizedWebPBase64, ",")
		if idx == -1 {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(res.OptimizedWebPBase64[idx+1:])
		if err != nil {
			continue
		}

		filename := res.Filename
		if count, exists := usedNames[filename]; exists {
			usedNames[filename] = count + 1
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filename, ext)
			filename = fmt.Sprintf("%s_%d%s", base, count+1, ext)
		} else {
			usedNames[filename] = 1
		}

		writer, err := zipWriter.Create(filename)
		if err != nil {
			continue
		}
		_, _ = writer.Write(data)
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	return buf.Bytes(), nil
}

// ImageTuneOptions specifies granular optimization parameters for live per-image tuning.
type ImageTuneOptions struct {
	Quality    float32 `json:"quality"`    // 1-100 (for lossy)
	Lossless   bool    `json:"lossless"`   // true = encode in WebP Lossless
	Exact      bool    `json:"exact"`      // true = preserve RGB in transparent areas
	MaxW       int     `json:"maxW"`       // downscale max width (0 = maintain orig)
	MaxH       int     `json:"maxH"`       // downscale max height (0 = maintain orig)
	Dither     bool    `json:"dither"`     // apply anti-banding Bayer dither on lossy
}

// ConvertImageBytesTuned converts in-memory raw image bytes directly to WebP using exact tuned options.
