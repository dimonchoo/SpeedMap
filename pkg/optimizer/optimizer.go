package optimizer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/chai2010/webp"
)

var (
	defaultUserAgentMu      sync.RWMutex
	currentDefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 SpeedMap/1.0"
)

// SetDefaultUserAgent sets the default User-Agent used by FetchImageBytes when none is provided.
func SetDefaultUserAgent(ua string) {
	if strings.TrimSpace(ua) == "" {
		return
	}
	defaultUserAgentMu.Lock()
	defer defaultUserAgentMu.Unlock()
	currentDefaultUserAgent = strings.TrimSpace(ua)
}

// GetDefaultUserAgent returns the current default User-Agent for image downloading.
func GetDefaultUserAgent() string {
	defaultUserAgentMu.RLock()
	defer defaultUserAgentMu.RUnlock()
	return currentDefaultUserAgent
}

var sharedClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	},
}

func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	if bytes < 0 {
		return "-" + FormatBytes(-bytes)
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
	return ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(rawURL, quality, 80.0, true, thresholdBytes, adaptive, maxW, maxH, user, pass)
}

// FetchImageBytes downloads the raw image bytes at rawURL with optional HTTP Basic Auth and User-Agent.
func FetchImageBytes(rawURL, user, pass string, userAgent ...string) ([]byte, error) {
	client := sharedClient
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	ua := GetDefaultUserAgent()
	if len(userAgent) > 0 && strings.TrimSpace(userAgent[0]) != "" {
		ua = strings.TrimSpace(userAgent[0])
	}
	req.Header.Set("User-Agent", ua)
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

	return io.ReadAll(resp.Body)
}

// ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality encodes to WebP with a strict minimum quality floor (e.g. 80%)
// and option to skip images if WebP output exceeds original file size.
func ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(rawURL string, quality float32, minQuality float32, skipIfNoSavings bool, thresholdBytes int64, adaptive bool, maxW, maxH int, user, pass string) (*ConversionResult, error) {
	if minQuality <= 0 || minQuality > 100 {
		minQuality = 80.0
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	if quality < minQuality {
		quality = minQuality
	}
	if thresholdBytes <= 0 {
		thresholdBytes = 100 * 1024
	}
	// Safe budget is 85% of threshold to guarantee we stay comfortably under the heavy limit
	safeBudget := int64(float64(thresholdBytes) * 0.85)
	if safeBudget < 40*1024 {
		safeBudget = thresholdBytes
	}

	origBytes, err := FetchImageBytes(rawURL, user, pass)
	if err != nil {
		return nil, err
	}

	decodeBytes := origBytes
	isTrojan := false
	if IsSVGContent(rawURL, origBytes) {
		if trojan, ok := DetectTrojanSVG(origBytes); ok {
			decodeBytes = trojan.RasterBytes
			isTrojan = true
		} else {
			// Pure vector SVG: preserve as clean vector graphic and optimize vector XML
			origSize := int64(len(origBytes))
			svgW, svgH := parseSVGDimensions(origBytes)
			optBytes, _ := OptimizeSVG(origBytes)
			optSize := int64(len(optBytes))
			savingsBytes, savingsPercent := ComputeSVGSavings(origSize, optSize)
			isSkipped := false
			if savingsBytes <= 0 && skipIfNoSavings {
				isSkipped = true
			}
			origBase64 := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(EnsureSVGXMLNS(origBytes)))
			optBase64 := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(EnsureSVGXMLNS(optBytes)))
			filename := ExtractOriginalFilename(rawURL)
			return &ConversionResult{
				URL:                 rawURL,
				Filename:            filename,
				OriginalWidth:       svgW,
				OriginalHeight:      svgH,
				OptimizedWidth:      svgW,
				OptimizedHeight:     svgH,
				OriginalBytes:       origSize,
				OriginalFormatted:   FormatBytes(origSize),
				OptimizedBytes:      optSize,
				OptimizedFormatted:  FormatBytes(optSize),
				SavingsBytes:        savingsBytes,
				SavingsFormatted:    FormatBytes(savingsBytes),
				SavingsPercent:      savingsPercent,
				QualityUsed:         100,
				IsLossless:          true,
				IsSkipped:           isSkipped,
				AdaptiveApplied:     false,
				OriginalDataBase64:  origBase64,
				OptimizedWebPBase64: optBase64,
			}, nil
		}
	}

	img, formatName, err := image.Decode(bytes.NewReader(decodeBytes))
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

	// 1. Compute Base Lossy WebP with quality parameter
	var lossyBuf bytes.Buffer
	lossyOpts := &webp.Options{
		Lossless: false,
		Quality:  quality,
		Exact:    false,
	}
	if err := webp.Encode(&lossyBuf, rawImg, lossyOpts); err != nil {
		return nil, fmt.Errorf("failed to encode to WebP: %w", err)
	}
	lossyData := lossyBuf.Bytes()
	lossyLen := int64(len(lossyData))
	webpData = lossyData
	qualityUsed = quality

	// Target-Budget Quality Step-Up:
	// If base lossy image compressed well below safe budget (e.g. < 65% of safeBudget, which is ~55 KB for a 100 KB budget),
	// we have plenty of budget headroom! Rather than unnecessarily over-compressing clean graphics/photos to 5-10 KB,
	// iteratively test higher quality levels up to 95% to maximize crispness and eliminate compression noise,
	// while strictly guaranteeing that the final size remains within safeBudget (<= 85 KB) and preserves strong savings.
	if adaptive && origSize > 60*1024 && lossyLen < int64(float64(safeBudget)*0.65) && quality < 95 {
		testQualities := []float32{88.0, 92.0, 95.0}
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
				if highQLen <= safeBudget && highQLen < int64(float64(origSize)*0.65) {
					webpData = highQBytes
					lossyData = highQBytes
					lossyLen = highQLen
					qualityUsed = testQ
					adaptiveApplied = true
				}
			}
		}
	}

	// 2. Adaptive Lossless Evaluation for PNG / Transparent graphics:
	// Lossless WebP is selected if:
	// 2. Adaptive Lossless Evaluation for PNG / Transparent graphics:
	// Lossless WebP is selected if it fits comfortably within safe budget (<= 85KB)
	// and is smaller than original (icons, badges, UI assets, transparent logos).
	if (formatName == "png" || isTransparent) && adaptive {
		var losslessBuf bytes.Buffer
		losslessOpts := &webp.Options{
			Lossless: true,
			Exact:    true,
		}
		if err := webp.Encode(&losslessBuf, rawImg, losslessOpts); err == nil {
			losslessBytes := losslessBuf.Bytes()
			losslessLen := int64(len(losslessBytes))

			// Lossless WebP should only be chosen if:
			// 1) It is smaller than or equal to Lossy (flat icons, logos, simple graphics), OR
			// 2) It is a tiny UI asset (<= 25 KB) where the delta from lossy is minimal (<= 5 KB), OR
			// 3) It is a smooth gradient UI graphic (isSmoothGradientOrUI) where Lossy WebP
			//    suffers from visible color banding stripes or circular rings (due to YUV 4:2:0 chroma quantization).
			//    Lossless guarantees 100% silky smoothness without banding while still saving bytes vs original!
			isSmallerThanLossy := losslessLen <= int64(len(lossyData))
			isTinyAssetWithSmallDelta := losslessLen <= 25*1024 && (losslessLen-int64(len(lossyData))) <= 5*1024
			isSmoothGradient := isSmoothGradientOrUI(rawImg)

			shouldUseLossless := isSmallerThanLossy || (losslessLen <= safeBudget && isTinyAssetWithSmallDelta) || (isSmoothGradient && losslessLen <= 650*1024 && (losslessLen <= int64(float64(origSize)*0.80) || losslessLen <= safeBudget))

			if losslessLen < origSize && shouldUseLossless {
				webpData = losslessBytes
				isLossless = true
				adaptiveApplied = true
				qualityUsed = 100
			}
		}
	}

	// 3. Anti-Banding Dithered WebP for JPEG / non-lossless Gradient Banners:
	// If the image is a smooth gradient banner (e.g. pav-bg3.jpg or cmo-bg-new1.jpg)
	// where Lossless WebP would exceed original JPEG size, apply subtle anti-banding dithering
	// at high quality (Q=96). This dissolves quantization plateaus (concentric rings) from 150px to 2-3px
	// while STRICTLY guaranteeing that the output fits within safeBudget (<= 85 KB)!
	if adaptive && isSmoothGradientOrUI(rawImg) && !isLossless {
		ditheredImg := applyAntiBandingDither(rawImg)
		var ditherBuf bytes.Buffer
		ditherOpts := &webp.Options{
			Lossless: false,
			Quality:  96.0,
			Exact:    false,
		}
		if err := webp.Encode(&ditherBuf, ditheredImg, ditherOpts); err == nil {
			ditherBytes := ditherBuf.Bytes()
			ditherLen := int64(len(ditherBytes))
			if ditherLen < origSize && ditherLen <= safeBudget {
				webpData = ditherBytes
				lossyData = ditherBytes
				lossyLen = ditherLen
				qualityUsed = 96.0
				adaptiveApplied = true
			}
		}
	}

		// Safety Guarantee: WebP output must NEVER be larger than the original asset.
		// If initial quality results in WebP >= origSize, step down quality towards minQuality (e.g. 80%).
		if int64(len(webpData)) >= origSize {
			fallbackQualities := []float32{90.0, 85.0, 80.0, 75.0, 70.0, 65.0, 60.0}
			for _, fq := range fallbackQualities {
				if fq >= qualityUsed || fq < minQuality {
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

	webpSize := int64(len(webpData))
	isSkipped := false
	if webpSize >= origSize {
		if skipIfNoSavings {
			isSkipped = true
		}
	}

	savings := origSize - webpSize
	if savings < 0 || isSkipped {
		savings = 0
	}
	var savingsPct float64
	if origSize > 0 && !isSkipped {
		savingsPct = float64(savings) / float64(origSize) * 100
	}

	mimeType := "image/jpeg"
	var origSourceBytes []byte = origBytes
	if isTrojan {
		formatLower := strings.ToLower(formatName)
		if formatLower == "jpg" {
			formatLower = "jpeg"
		}
		mimeType = "image/" + formatLower
		origSourceBytes = decodeBytes
	} else {
		switch strings.ToLower(formatName) {
		case "png":
			mimeType = "image/png"
		case "gif":
			mimeType = "image/gif"
		case "webp":
			mimeType = "image/webp"
		case "bmp":
			mimeType = "image/bmp"
		}
	}

	filename := ExtractFilenameFromURL(rawURL)

	origBase64 := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(origSourceBytes))
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
		IsSkipped:           isSkipped,
		AdaptiveApplied:     adaptiveApplied,
		OriginalDataBase64:  origBase64,
		OptimizedWebPBase64: webpBase64,
	}, nil
}

func ConvertImageBytesTuned(rawURL string, origBytes []byte, opts ImageTuneOptions) (*ConversionResult, error) {
	if len(origBytes) == 0 {
		return nil, fmt.Errorf("empty image bytes provided")
	}

	decodeBytes := origBytes
	isTrojan := false
	if IsSVGContent(rawURL, origBytes) {
		if trojan, ok := DetectTrojanSVG(origBytes); ok {
			decodeBytes = trojan.RasterBytes
			isTrojan = true
		} else {
			// Pure vector SVG: preserve as clean vector graphic and optimize vector XML
			origSize := int64(len(origBytes))
			svgW, svgH := parseSVGDimensions(origBytes)
			optBytes, _ := OptimizeSVG(origBytes)
			optSize := int64(len(optBytes))
			savingsBytes, savingsPercent := ComputeSVGSavings(origSize, optSize)
			isSkipped := false
			if savingsBytes <= 0 {
				isSkipped = true
			}
			origBase64 := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(EnsureSVGXMLNS(origBytes)))
			optBase64 := fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(EnsureSVGXMLNS(optBytes)))
			filename := ExtractOriginalFilename(rawURL)
			return &ConversionResult{
				URL:                 rawURL,
				Filename:            filename,
				OriginalWidth:       svgW,
				OriginalHeight:      svgH,
				OptimizedWidth:      svgW,
				OptimizedHeight:     svgH,
				OriginalBytes:       origSize,
				OriginalFormatted:   FormatBytes(origSize),
				OptimizedBytes:      optSize,
				OptimizedFormatted:  FormatBytes(optSize),
				SavingsBytes:        savingsBytes,
				SavingsFormatted:    FormatBytes(savingsBytes),
				SavingsPercent:      savingsPercent,
				QualityUsed:         100,
				IsLossless:          true,
				IsSkipped:           isSkipped,
				AdaptiveApplied:     false,
				OriginalDataBase64:  origBase64,
				OptimizedWebPBase64: optBase64,
			}, nil
		}
	}

	img, formatName, err := image.Decode(bytes.NewReader(decodeBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (format %s): %w", formatName, err)
	}

	origBounds := img.Bounds()
	origW := origBounds.Dx()
	origH := origBounds.Dy()
	origSize := int64(len(origBytes))

	rawImg := toStraightRGBA(img)

	// Downscale if requested
	if opts.MaxW > 0 || opts.MaxH > 0 {
		rawImg = resizeProportional(rawImg, opts.MaxW, opts.MaxH)
	}

	optBounds := rawImg.Bounds()
	optW := optBounds.Dx()
	optH := optBounds.Dy()

	if opts.Dither && !opts.Lossless {
		rawImg = applyAntiBandingDither(rawImg)
	}

	var webpBuf bytes.Buffer
	quality := opts.Quality
	if quality <= 0 || quality > 100 {
		quality = 80.0
	}

	webpOpts := &webp.Options{
		Lossless: opts.Lossless,
		Quality:  quality,
		Exact:    opts.Exact,
	}
	if opts.Lossless {
		webpOpts.Quality = 100.0
		quality = 100.0
	}

	if err := webp.Encode(&webpBuf, rawImg, webpOpts); err != nil {
		return nil, fmt.Errorf("failed to encode to WebP: %w", err)
	}

	webpData := webpBuf.Bytes()
	webpSize := int64(len(webpData))

	savings := origSize - webpSize
	var savingsPct float64
	if origSize > 0 {
		savingsPct = float64(savings) / float64(origSize) * 100
	}

	filename := ExtractFilenameFromURL(rawURL)

	mimeType := "image/jpeg"
	var origSourceBytes []byte = origBytes
	if isTrojan {
		formatLower := strings.ToLower(formatName)
		if formatLower == "jpg" {
			formatLower = "jpeg"
		}
		mimeType = "image/" + formatLower
		origSourceBytes = decodeBytes
	} else {
		switch strings.ToLower(formatName) {
		case "png":
			mimeType = "image/png"
		case "gif":
			mimeType = "image/gif"
		case "webp":
			mimeType = "image/webp"
		case "bmp":
			mimeType = "image/bmp"
		}
	}

	origBase64 := fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(origSourceBytes))
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
		QualityUsed:         quality,
		IsLossless:          opts.Lossless,
		IsSkipped:           false,
		AdaptiveApplied:     false,
		OriginalDataBase64:  origBase64,
		OptimizedWebPBase64: webpBase64,
	}, nil
}

