package optimizer

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/chai2010/webp"
)

type ConversionResult struct {
	URL                 string  `json:"url"`
	Filename            string  `json:"filename"`
	OriginalBytes       int64   `json:"originalBytes"`
	OriginalFormatted   string  `json:"originalFormatted"`
	OptimizedBytes      int64   `json:"optimizedBytes"`
	OptimizedFormatted  string  `json:"optimizedFormatted"`
	SavingsBytes        int64   `json:"savingsBytes"`
	SavingsFormatted    string  `json:"savingsFormatted"`
	SavingsPercent      float64 `json:"savingsPercent"`
	OriginalDataBase64  string  `json:"originalDataBase64"`
	OptimizedWebPBase64 string  `json:"optimizedWebPBase64"`
	Error               string  `json:"error"`
}

func FormatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const k = 1024
	sizes := []string{"B", "KB", "MB", "GB"}
	i := 0
	val := float64(bytes)
	for val >= k && i < len(sizes)-1 {
		val /= k
		i++
	}
	return fmt.Sprintf("%.1f %s", val, sizes[i])
}

// ConvertImageURLToWebP downloads the image at rawURL and encodes it to WebP at specified quality (1-100)
func ConvertImageURLToWebP(rawURL string, quality float32) (*ConversionResult, error) {
	if quality <= 0 || quality > 100 {
		quality = 80
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "SpeedMap-Optimizer/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d fetching image", resp.StatusCode)
	}

	origBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image body: %w", err)
	}

	img, formatName, err := image.Decode(bytes.NewReader(origBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format %s): %w", formatName, err)
	}

	var webpBuf bytes.Buffer
	options := &webp.Options{
		Lossless: false,
		Quality:  quality,
	}

	if err := webp.Encode(&webpBuf, img, options); err != nil {
		return nil, fmt.Errorf("failed to encode to WebP: %w", err)
	}

	webpData := webpBuf.Bytes()
	origSize := int64(len(origBytes))
	webpSize := int64(len(webpData))
	savings := origSize - webpSize
	if savings < 0 {
		savings = 0
	}
	var savingsPct float64
	if origSize > 0 {
		savingsPct = float64(savings) / float64(origSize) * 100
	}

	filename := ExtractFilenameFromURL(rawURL)

	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	origBase64 := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(origBytes))
	webpBase64 := fmt.Sprintf("data:image/webp;base64,%s", base64.StdEncoding.EncodeToString(webpData))

	return &ConversionResult{
		URL:                 rawURL,
		Filename:            filename,
		OriginalBytes:       origSize,
		OriginalFormatted:   FormatBytes(origSize),
		OptimizedBytes:      webpSize,
		OptimizedFormatted:  FormatBytes(webpSize),
		SavingsBytes:        savings,
		SavingsFormatted:    FormatBytes(savings),
		SavingsPercent:      savingsPct,
		OriginalDataBase64:  origBase64,
		OptimizedWebPBase64: webpBase64,
	}, nil
}

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
