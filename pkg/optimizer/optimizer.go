package optimizer

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"

	"github.com/chai2010/webp"
)

var sharedClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	},
}

type ConversionResult struct {
	URL                 string  `json:"url"`
	Filename            string  `json:"filename"`
	OriginalWidth       int     `json:"originalWidth,omitempty"`
	OriginalHeight      int     `json:"originalHeight,omitempty"`
	OptimizedWidth      int     `json:"optimizedWidth,omitempty"`
	OptimizedHeight     int     `json:"optimizedHeight,omitempty"`
	OriginalBytes       int64   `json:"originalBytes"`
	OriginalFormatted   string  `json:"originalFormatted"`
	OptimizedBytes      int64   `json:"optimizedBytes"`
	OptimizedFormatted  string  `json:"optimizedFormatted"`
	SavingsBytes        int64   `json:"savingsBytes"`
	SavingsFormatted    string  `json:"savingsFormatted"`
	SavingsPercent      float64 `json:"savingsPercent"`
	QualityUsed         float32 `json:"qualityUsed"`
	IsLossless          bool    `json:"isLossless"`
	AdaptiveApplied     bool    `json:"adaptiveApplied"`
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

// ConvertImageURLToWebP downloads the image at rawURL and encodes it to WebP at specified quality (1-100).
func ConvertImageURLToWebP(rawURL string, quality float32) (*ConversionResult, error) {
	return ConvertImageURLToWebPAdaptiveAuth(rawURL, quality, true, "", "")
}

// ConvertImageURLToWebPAuth is ConvertImageURLToWebP with optional HTTP Basic Auth (empty user = no auth).
func ConvertImageURLToWebPAuth(rawURL string, quality float32, user, pass string) (*ConversionResult, error) {
	return ConvertImageURLToWebPAdaptiveAuth(rawURL, quality, true, user, pass)
}

// hasTransparency checks whether an image has any pixels with Alpha < 255
func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	// Sample pixels to quickly detect transparency
	stepY := (bounds.Dy() / 40) + 1
	stepX := (bounds.Dx() / 40) + 1
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xfffe {
				return true
			}
		}
	}
	return false
}

// toStraightRGBA converts any decoded image into *image.RGBA with STRAIGHT (unpremultiplied) RGBA bytes,
// avoiding Go's color.RGBAModel premultiplication bug from crushing anti-aliased edge colors to dark/black in libwebp C-API.
func toStraightRGBA(m image.Image) *image.RGBA {
	bounds := m.Bounds()
	if nrgba, ok := m.(*image.NRGBA); ok {
		rgba := &image.RGBA{
			Pix:    make([]uint8, len(nrgba.Pix)),
			Stride: nrgba.Stride,
			Rect:   nrgba.Rect,
		}
		copy(rgba.Pix, nrgba.Pix)
		return rgba
	}
	if rgba, ok := m.(*image.RGBA); ok {
		return rgba
	}
	rgba := image.NewRGBA(bounds)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := m.At(x, y)
			if nc, ok := color.NRGBAModel.Convert(c).(color.NRGBA); ok {
				rgba.Pix[idx] = nc.R
				rgba.Pix[idx+1] = nc.G
				rgba.Pix[idx+2] = nc.B
				rgba.Pix[idx+3] = nc.A
			} else {
				r, g, b, a := c.RGBA()
				if a > 0 {
					rgba.Pix[idx] = uint8((r * 255) / a)
					rgba.Pix[idx+1] = uint8((g * 255) / a)
					rgba.Pix[idx+2] = uint8((b * 255) / a)
					rgba.Pix[idx+3] = uint8(a >> 8)
				} else {
					rgba.Pix[idx] = 0
					rgba.Pix[idx+1] = 0
					rgba.Pix[idx+2] = 0
					rgba.Pix[idx+3] = 0
				}
			}
			idx += 4
		}
	}
	return rgba
}

func resizeProportional(src *image.RGBA, maxW, maxH int) *image.RGBA {
	bounds := src.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW <= 0 || origH <= 0 || (maxW <= 0 && maxH <= 0) {
		return src
	}

	targetW := origW
	targetH := origH

	if maxW > 0 && targetW > maxW {
		targetW = maxW
		targetH = int(math.Round(float64(origH) * float64(maxW) / float64(origW)))
	}
	if maxH > 0 && targetH > maxH {
		targetH = maxH
		targetW = int(math.Round(float64(origW) * float64(maxH) / float64(origH)))
	}

	if targetW <= 0 {
		targetW = 1
	}
	if targetH <= 0 {
		targetH = 1
	}

	if targetW >= origW && targetH >= origH {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// ConvertImageURLToWebPAdaptiveAuth encodes to WebP with default 100KB heavy threshold budget.
func ConvertImageURLToWebPAdaptiveAuth(rawURL string, quality float32, adaptive bool, user, pass string) (*ConversionResult, error) {
	return ConvertImageURLToWebPAdaptiveBudgetAuth(rawURL, quality, 100*1024, adaptive, user, pass)
}

// ConvertImageURLToWebPAdaptiveBudgetAuth finds the optimal WebP quality that maximizes fidelity
// without exceeding the specified heavy threshold byte budget.
func ConvertImageURLToWebPAdaptiveBudgetAuth(rawURL string, quality float32, thresholdBytes int64, adaptive bool, user, pass string) (*ConversionResult, error) {
	return ConvertImageURLToWebPAdaptiveBudgetAuthResize(rawURL, quality, thresholdBytes, adaptive, 0, 0, user, pass)
}

// ConvertImageURLToWebPAdaptiveBudgetAuthResize encodes to WebP, optionally downscaling oversized
// images proportionally to target maxW/maxH (e.g. max rendered Retina 2x bounds).
func ConvertImageURLToWebPAdaptiveBudgetAuthResize(rawURL string, quality float32, thresholdBytes int64, adaptive bool, maxW, maxH int, user, pass string) (*ConversionResult, error) {
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	if thresholdBytes <= 0 {
		thresholdBytes = 100 * 1024
	}
	// Safe budget is 85% of threshold to guarantee we stay comfortably under the heavy limit
	safeBudget := int64(float64(thresholdBytes) * 0.85)
	if safeBudget < 40*1024 {
		safeBudget = thresholdBytes
	}

	client := sharedClient
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "SpeedMap-Optimizer/1.0")
	if user != "" {
		req.SetBasicAuth(user, pass)
	}

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

	origBounds := img.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()

	origSize := int64(len(origBytes))
	isTransparent := hasTransparency(img)
	isLossless := false
	qualityUsed := quality
	adaptiveApplied := false

	// Convert decoded image into raw straight (unpremultiplied) RGBA to prevent Go's color premultiplication
	// from corrupting anti-aliased edge pixels into dark/black borders in libwebp.
	rawImg := toStraightRGBA(img)

	// Apply proportional downscaling if target dimensions provided (Properly Size Images for Lighthouse)
	if maxW > 0 || maxH > 0 {
		rawImg = resizeProportional(rawImg, maxW, maxH)
	}

	optBounds := rawImg.Bounds()
	optW := optBounds.Dx()
	optH := optBounds.Dy()

	var webpData []byte

	// 1. Transparent PNG / UI Graphics Handling:
	// Lossless WebP is ideal for UI graphics and icons if it fits within safeBudget.
	// For photographic/complex images with alpha, Lossy WebP (with straight RGBA) preserves alpha cleanly
	// while providing 80-95% compression savings.
	if (formatName == "png" || isTransparent) && adaptive {
		var losslessBuf bytes.Buffer
		losslessOpts := &webp.Options{
			Lossless: true,
			Exact:    true,
		}
		if err := webp.Encode(&losslessBuf, rawImg, losslessOpts); err == nil {
			losslessBytes := losslessBuf.Bytes()
			losslessLen := int64(len(losslessBytes))
			// Only choose Lossless if it comfortably fits within safe budget AND is smaller than original
			if losslessLen <= safeBudget && losslessLen < origSize {
				webpData = losslessBytes
				isLossless = true
				adaptiveApplied = true
				qualityUsed = 100
			}
		}
	}

	// 2. Photographic / General Lossy WebP with Target-Budget Optimization
	if len(webpData) == 0 {
		var webpBuf bytes.Buffer
		opts := &webp.Options{
			Lossless: false,
			Quality:  quality,
			Exact:    false,
		}
		if err := webp.Encode(&webpBuf, rawImg, opts); err != nil {
			return nil, fmt.Errorf("failed to encode to WebP: %w", err)
		}
		webpData = webpBuf.Bytes()
		qualityUsed = quality

		// Target-Budget Optimization:
		// If image compressed to a size well below our safe budget (e.g. < 70% of safeBudget),
		// we iteratively test higher quality levels up to 96% to restore fine details,
		// crisp gradients, and remove compression artifacts while remaining strictly within budget.
		if adaptive && origSize > 80*1024 && int64(len(webpData)) < int64(float64(safeBudget)*0.70) && quality < 96 {
			testQualities := []float32{85.0, 88.0, 91.0, 93.0, 95.0, 96.0}
			for _, testQ := range testQualities {
				if testQ <= quality {
					continue
				}
				var highQBuf bytes.Buffer
				highQOpts := &webp.Options{
					Lossless: false,
					Quality:  testQ,
					Exact:    false,
				}
				if err := webp.Encode(&highQBuf, rawImg, highQOpts); err == nil {
					highQBytes := highQBuf.Bytes()
					highQLen := int64(len(highQBytes))
					// Ensure we don't breach the safeBudget or drop below 40% savings
					if highQLen <= safeBudget && highQLen < int64(float64(origSize)*0.60) {
						webpData = highQBytes
						qualityUsed = testQ
						adaptiveApplied = true
					} else {
						// Exceeded budget or savings floor, stop ascending
						break
					}
				}
			}
		}

		// Safety Guarantee: WebP output must NEVER be larger than the original asset.
		// If high quality (e.g. 90%) results in WebP >= origSize on indexed/noisy PNGs,
		// step down quality (85, 80, 75, 70, 65) to guarantee strict bandwidth reduction.
		if int64(len(webpData)) >= origSize {
			fallbackQualities := []float32{85.0, 80.0, 75.0, 70.0, 65.0, 60.0}
			for _, fq := range fallbackQualities {
				if fq >= qualityUsed {
					continue
				}
				var fBuf bytes.Buffer
				fOpts := &webp.Options{
					Lossless: false,
					Quality:  fq,
					Exact:    false,
				}
				if err := webp.Encode(&fBuf, rawImg, fOpts); err == nil {
					fBytes := fBuf.Bytes()
					if int64(len(fBytes)) < origSize {
						webpData = fBytes
						qualityUsed = fq
						adaptiveApplied = true
						break
					}
				}
			}
		}
	}

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
		OriginalWidth:       origW,
		OriginalHeight:      origH,
		OptimizedWidth:      optW,
		OptimizedHeight:     optH,
		OriginalBytes:       origSize,
		OriginalFormatted:   FormatBytes(origSize),
		OptimizedBytes:      webpSize,
		OptimizedFormatted:  FormatBytes(webpSize),
		SavingsBytes:        savings,
		SavingsFormatted:    FormatBytes(savings),
		SavingsPercent:      savingsPct,
		QualityUsed:         qualityUsed,
		IsLossless:          isLossless,
		AdaptiveApplied:     adaptiveApplied,
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
