package optimizer

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertImageURLToWebP(t *testing.T) {
	// Create a dummy 100x100 PNG image in memory
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode dummy png: %v", err)
	}

	// Serve the dummy PNG image via mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	testURL := ts.URL + "/hero-8.png"
	res, err := ConvertImageURLToWebP(testURL, 80)
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}

	if res.Filename != "hero-8.webp" {
		t.Errorf("expected filename hero-8.webp, got %s", res.Filename)
	}
	if res.OriginalBytes == 0 {
		t.Errorf("expected original bytes > 0")
	}
	if res.OptimizedBytes == 0 {
		t.Errorf("expected optimized bytes > 0")
	}
	if res.OptimizedWebPBase64 == "" {
		t.Errorf("expected webp base64 non-empty")
	}

	// Test ZIP creation
	zipBytes, err := CreateZIPArchive([]*ConversionResult{res})
	if err != nil {
		t.Fatalf("CreateZIPArchive failed: %v", err)
	}
	if len(zipBytes) == 0 {
		t.Errorf("expected non-empty zip bytes")
	}
}

func TestConvertImageURLToWebPAuth(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "uat" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	if _, err := ConvertImageURLToWebP(ts.URL+"/a.png", 80); err == nil {
		t.Fatal("expected error without auth")
	}
	res, err := ConvertImageURLToWebPAuth(ts.URL+"/a.png", 80, "uat", "secret")
	if err != nil {
		t.Fatalf("with auth: %v", err)
	}
	if res.OptimizedBytes == 0 {
		t.Fatal("expected webp bytes")
	}
}

func TestConvertTransparentHeavyImageAdaptive(t *testing.T) {
	// Create a large 800x600 PNG with gradient and transparency (simulating a banner/avatar)
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			a := uint8(255)
			if x < 100 {
				a = uint8((x * 255) / 100)
			}
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / 800),
				G: uint8((y * 255) / 600),
				B: uint8(((x + y) * 255) / 1400),
				A: a,
			})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Safe budget is 85KB (85% of 100KB). Lossless WebP on complex gradients easily exceeds 150KB.
	// The optimizer must correctly recognize that lossless exceeds safeBudget and fall back to Lossy WebP with alpha preservation.
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuth(ts.URL+"/gradient.png", 80, 100*1024, true, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebPAdaptiveBudgetAuth failed: %v", err)
	}

	if res.OptimizedBytes > 100*1024 {
		t.Errorf("expected optimized size <= 100KB budget, got %d bytes (%s)", res.OptimizedBytes, res.OptimizedFormatted)
	}
	if res.SavingsPercent < 50.0 {
		t.Errorf("expected savings >= 50%%, got %.1f%%", res.SavingsPercent)
	}
	if res.OptimizedWebPBase64 == "" {
		t.Errorf("expected non-empty webp base64")
	}
}

func TestConvertImageResizeProportional(t *testing.T) {
	// Create 4000x2000 image
	img := image.NewRGBA(image.Rect(0, 0, 4000, 2000))
	for y := 0; y < 2000; y += 100 {
		for x := 0; x < 4000; x += 100 {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Request resize to maxW=1000, maxH=500
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResize(ts.URL+"/oversized.png", 80, 100*1024, true, 1000, 500, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebPAdaptiveBudgetAuthResize failed: %v", err)
	}

	if res.OriginalWidth != 4000 || res.OriginalHeight != 2000 {
		t.Errorf("expected original 4000x2000, got %dx%d", res.OriginalWidth, res.OriginalHeight)
	}
	if res.OptimizedWidth != 1000 || res.OptimizedHeight != 500 {
		t.Errorf("expected optimized 1000x500, got %dx%d", res.OptimizedWidth, res.OptimizedHeight)
	}
}

func TestCaseStudyImageOnline(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2025/02/Case-Study_Slider-image-1770x700-1-6.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 90, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v, Skipped=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless, res.IsSkipped)
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("OptimizedBytes (%d) >= OriginalBytes (%d)", res.OptimizedBytes, res.OriginalBytes)
	}
}

func TestVoiceOfTheBuyerImageOnline(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2025/12/Voice-of-the-Buyer-2026.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("Voice-of-the-Buyer Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	// Must be well under 100 KB in Lossy mode
	if res.OptimizedBytes > 100*1024 {
		t.Errorf("expected Voice-of-the-Buyer to use Lossy (<100KB), got %s", res.OptimizedFormatted)
	}
}

func TestGtsBg1GradientOnline(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2023/12/gts-bg1.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("gts-bg1 Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	if !res.IsLossless {
		t.Errorf("expected gts-bg1 gradient to route to Lossless to avoid color banding, got Lossless=%v", res.IsLossless)
	}
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("expected gts-bg1 lossless to save bytes vs original, got Opt=%s, Orig=%s", res.OptimizedFormatted, res.OriginalFormatted)
	}
}

func TestCtaBg8GradientOnline(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2024/05/CTA-BG-8.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("CTA-BG-8 Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	if !res.IsLossless {
		t.Errorf("expected CTA-BG-8 gradient to route to Lossless to avoid color banding, got Lossless=%v", res.IsLossless)
	}
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("expected CTA-BG-8 lossless to save bytes vs original, got Opt=%s, Orig=%s", res.OptimizedFormatted, res.OriginalFormatted)
	}
}

func TestDepositphotosStaysLossy(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2026/03/Depositphotos_166776156_xl-2015-1.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("Depositphotos Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	// Photos MUST stay in Lossy mode and must not bloat into 1.8MB Lossless WebP
	if res.IsLossless {
		t.Errorf("expected Depositphotos to stay Lossy, but got Lossless=%v (%s)", res.IsLossless, res.OptimizedFormatted)
	}
	if res.OptimizedBytes > 200*1024 {
		t.Errorf("expected Depositphotos to stay compact (<200KB), got %s", res.OptimizedFormatted)
	}
}

func TestHeroBgStaysLossy(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2026/04/Hero-bg.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("Hero-bg Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	// Hero illustrations MUST stay in Lossy mode and must not bloat into 1.4MB Lossless WebP
	if res.IsLossless {
		t.Errorf("expected Hero-bg to stay Lossy, but got Lossless=%v (%s)", res.IsLossless, res.OptimizedFormatted)
	}
	if res.OptimizedBytes > 200*1024 {
		t.Errorf("expected Hero-bg to stay compact (<200KB), got %s", res.OptimizedFormatted)
	}
}

func TestCswCover1GradientOnline(t *testing.T) {
	url := "https://infuse.com/wp-content/uploads/2022/07/csw-cover1.png"
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Skipf("network error skipping: %v", err)
	}
	t.Logf("csw-cover1 Result: Orig=%s, Opt=%s, Savings=%.1f%%, Q=%.1f, Lossless=%v",
		res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)

	// Must route to Lossless to completely eliminate concentric circular banding rings
	if !res.IsLossless {
		t.Errorf("expected csw-cover1 radial gradient to route to Lossless to eliminate circular banding, got Lossless=%v", res.IsLossless)
	}
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("expected csw-cover1 lossless to save bytes vs original, got Opt=%s, Orig=%s", res.OptimizedFormatted, res.OriginalFormatted)
	}
}

// === Offline Fixture Tests ===

func serveLocalFixture(t *testing.T, fixtureRelPath, contentType string) (*httptest.Server, string) {
	data, err := os.ReadFile(fixtureRelPath)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", fixtureRelPath, err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write(data)
	}))
	return ts, ts.URL + "/" + filepath.Base(fixtureRelPath)
}

func TestOfflineIconLossless(t *testing.T) {
	ts, url := serveLocalFixture(t, "testdata/icon_transparent.png", "image/png")
	defer ts.Close()

	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 80, 80, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}
	if !res.IsLossless {
		t.Errorf("expected small transparent icon to use Lossless WebP, got Lossy (Q=%.1f)", res.QualityUsed)
	}
	if res.OptimizedBytes == 0 || res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("expected optimized size < original size, got opt=%d, orig=%d", res.OptimizedBytes, res.OriginalBytes)
	}
}

func TestOfflineLargePhotoPngRoutesToLossy(t *testing.T) {
	ts, url := serveLocalFixture(t, "testdata/photo_large.png", "image/png")
	defer ts.Close()

	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 85, 75, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}
	if res.IsLossless {
		t.Errorf("expected large photographic PNG to route to Lossy WebP, but got Lossless (%s)", res.OptimizedFormatted)
	}
	if res.SavingsPercent < 50.0 {
		t.Errorf("expected at least 50%% savings on photo PNG, got %.1f%% (%s -> %s)", res.SavingsPercent, res.OriginalFormatted, res.OptimizedFormatted)
	}
}

func TestOfflineRetinaResizingProportional(t *testing.T) {
	ts, url := serveLocalFixture(t, "testdata/photo_sample.jpg", "image/jpeg")
	defer ts.Close()

	// Original is 1200x800. Request maxRenderedWidth=300, maxRenderedHeight=200 with 2x Retina (600x400)
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(url, 85, 75, true, 100*1024, true, 600, 400, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}
	if res.OptimizedWidth != 600 || res.OptimizedHeight != 400 {
		t.Errorf("expected resized dimensions 600x400, got %dx%d", res.OptimizedWidth, res.OptimizedHeight)
	}
}

func TestOfflineQualityStepDownFallback(t *testing.T) {
	// Create an already compact PNG where high quality (95%) would result in WebP >= Original
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Request initial quality 95 with minQuality 70
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(ts.URL+"/stepdown.png", 95, 70, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}

	// Must never be larger than original
	if res.OptimizedBytes >= res.OriginalBytes {
		t.Errorf("expected step-down quality to produce WebP < Original, got opt=%d, orig=%d", res.OptimizedBytes, res.OriginalBytes)
	}
	if !res.AdaptiveApplied {
		t.Errorf("expected AdaptiveApplied to be true on step-down")
	}
}

func TestOfflineSkipIfNoSavings(t *testing.T) {
	// Create a tiny 5x5 monochrome PNG (e.g. 80 bytes) where WebP header overhead makes it larger
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// With skipIfNoSavings = true and high minQuality floor (90%)
	res, err := ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(ts.URL+"/tiny.png", 95, 90, true, 100*1024, true, 0, 0, "", "")
	if err != nil {
		t.Fatalf("ConvertImageURLToWebP failed: %v", err)
	}

	// If WebP >= Original, it should be marked as skipped
	if res.OptimizedBytes >= res.OriginalBytes && !res.IsSkipped {
		t.Errorf("expected IsSkipped == true when WebP >= Original, got IsSkipped=false")
	}
}

func TestStraightRGBAAlphaIntegrity(t *testing.T) {
	// Test toStraightRGBA function with semi-transparent NRGBA pixels
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	// Bright cyan with 50% opacity
	nrgba.SetNRGBA(0, 0, color.NRGBA{R: 0, G: 200, B: 255, A: 128})

	straight := toStraightRGBA(nrgba)
	if straight == nil {
		t.Fatal("expected non-nil straight RGBA")
	}

	// RGB values must not be crushed/premultiplied to half (0, 100, 127)
	r := straight.Pix[0]
	g := straight.Pix[1]
	b := straight.Pix[2]
	a := straight.Pix[3]

	if r != 0 || g != 200 || b != 255 || a != 128 {
		t.Errorf("expected straight RGBA (0, 200, 255, 128), got (%d, %d, %d, %d)", r, g, b, a)
	}
}


