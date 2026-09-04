package wpexport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
)

func TestBuildApplyPHP(t *testing.T) {
	cfg := config.ScanConfig{WebPQuality: 80}
	images := []analytics.AggregatedImage{
		{
			URL:             "https://example.com/wp-content/uploads/2024/03/hero.png",
			Format:          "png",
			IsHeavy:         true,
			MaxTransferSize: 500000,
			Pages:           []string{"https://example.com/", "https://example.com/about"},
		},
		{
			URL:             "https://example.com/wp-content/uploads/2024/03/hero-1024x768.png",
			Format:          "png",
			IsHeavy:         false,
			MaxTransferSize: 100000,
			Pages:           []string{"https://example.com/blog"},
		},
		{
			URL:             "https://example.com/wp-content/uploads/2024/03/light.png",
			Format:          "png",
			IsHeavy:         false,
			MaxTransferSize: 10000,
			Pages:           []string{"https://example.com/"},
		},
		{URL: "https://example.com/wp-content/uploads/logo.svg", Format: "svg", IsHeavy: false},
		{URL: "https://example.com/wp-content/uploads/ok.webp", Format: "webp", IsHeavy: false},
	}

	php, err := BuildApplyPHP("https://example.com", cfg, images, "/var/www/site")
	if err != nil {
		t.Fatalf("BuildApplyPHP: %v", err)
	}
	if strings.Contains(php, `"sourceUrl": "https://example.com/wp-content/uploads/2024/03/hero-1024x768.png"`) {
		t.Fatalf("resized URL must not be sourceUrl after dedupe/prefer-original")
	}
	if !strings.Contains(php, `"sourceUrl": "https://example.com/wp-content/uploads/2024/03/hero.png"`) {
		t.Fatalf("expected original sourceUrl")
	}
	if strings.Contains(php, "light.png") {
		t.Fatalf("non-heavy light.png must be excluded")
	}
	if strings.Count(php, `"basename": "hero.png"`) != 1 {
		t.Fatalf("expected single hero.png entry after dedupe")
	}
	if !strings.Contains(php, "https://example.com/blog") {
		t.Fatalf("expected pages merged from sized variant")
	}
	if !strings.Contains(php, "2024/03/hero.webp") {
		t.Fatalf("expected webpRel 2024/03/hero.webp")
	}
	if !strings.Contains(php, "wp eval-file") {
		t.Fatalf("expected runbook comment")
	}
	if !strings.Contains(php, "speedmap-webp-backup") {
		t.Fatalf("expected backup logic in apply PHP")
	}
	if strings.Contains(php, "download_url") || strings.Contains(php, "speedmap_convert_to_webp") {
		t.Fatalf("PHP must be DB-only (no download/convert)")
	}
	rb := BuildRollbackPHP("/var/www/site")
	if !strings.Contains(rb, "speedmap-webp-backup") || !strings.Contains(rb, "--path=/var/www/site") {
		t.Fatalf("rollback template incomplete")
	}
}

func TestCollectHeavyImagesWebpRelDedupe(t *testing.T) {
	images := []analytics.AggregatedImage{
		{URL: "https://ex.com/wp-content/uploads/2024/12/Hero-BG-Mobile.png", Format: "png", IsHeavy: true, MaxTransferSize: 200000},
		{URL: "https://ex.com/wp-content/uploads/2024/12/Hero-BG-Mobile.jpg", Format: "jpg", IsHeavy: true, MaxTransferSize: 100000},
		{URL: "https://ex.com/wp-content/uploads/a.png", Format: "png", IsHeavy: true, MaxTransferSize: 150000},
		{URL: "https://ex.com/wp-content/uploads/b.png", Format: "png", IsHeavy: false, MaxTransferSize: 50000},
	}
	got := CollectHeavyImages(images)
	if len(got) != 2 {
		t.Fatalf("expected 2 heavy unique webpRel, got %d %+v", len(got), got)
	}
	var bg *ManifestImage
	for i := range got {
		if strings.Contains(got[i].WebpRel, "Hero-BG-Mobile") {
			bg = &got[i]
		}
	}
	if bg == nil {
		t.Fatal("missing Hero-BG-Mobile")
	}
	if !strings.HasSuffix(bg.SourceURL, ".png") {
		t.Fatalf("prefer larger png, got %s", bg.SourceURL)
	}
	if bg.Bytes != 200000 {
		t.Fatalf("bytes=%d", bg.Bytes)
	}
}

func TestPreferOriginalURL(t *testing.T) {
	got := preferOriginalURL("https://ex.com/wp-content/uploads/2026/07/hero-8-465x203.png")
	want := "https://ex.com/wp-content/uploads/2026/07/hero-8.png"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestBuildApplyPHPRequiresPath(t *testing.T) {
	_, err := BuildApplyPHP("https://example.com", config.ScanConfig{}, []analytics.AggregatedImage{
		{URL: "https://example.com/a.png", Format: "png", IsHeavy: true},
	}, "  ")
	if err == nil {
		t.Fatal("expected error for empty wordpress path")
	}
}

func TestStripSizeSuffix(t *testing.T) {
	got := stripSizeSuffix("hero-1024x768.png")
	if got != "hero.png" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteWebPFilesAndReviewZIP(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	for x := 0; x < 20; x++ {
		for y := 0; y < 20; y++ {
			img.Set(x, y, color.RGBA{R: 0, G: 128, B: 255, A: 255})
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

	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "wp-content", "uploads"), 0755)

	images := []ManifestImage{{
		SourceURL: ts.URL + "/wp-content/uploads/2024/03/hero.png",
		PathHint:  "2024/03/hero.png",
		Basename:  "hero.png",
		WebpRel:   "2024/03/hero.webp",
		Format:    "png",
		IsHeavy:   true,
	}}

	written, n, err := WriteWebPFiles(root, images, 80, "uat", "secret")
	if err != nil {
		t.Fatalf("WriteWebPFiles: %v", err)
	}
	if n != 1 || len(written) != 1 {
		t.Fatalf("written=%d len=%d", n, len(written))
	}
	// Convert only — must NOT write into uploads anymore.
	uploadsWebP := filepath.Join(root, "wp-content", "uploads", "2024", "03", "hero.webp")
	if _, err := os.Stat(uploadsWebP); err == nil {
		t.Fatalf("must not write WebP into uploads during convert: %s", uploadsWebP)
	}
	if written[0].ID == "" || written[0].PackageWebP == "" {
		t.Fatalf("expected package id/path on written image")
	}

	pkg := filepath.Join(root, "package")
	out, err := WriteDeployPackage(pkg, "https://example.com", config.ScanConfig{WebPQuality: 80}, written)
	if err != nil {
		t.Fatalf("WriteDeployPackage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkg, "images", "001", "optimized.webp")); err != nil {
		t.Fatalf("expected package webp: %v", err)
	}
	if _, err := os.Stat(out.ApplyPHP); err != nil {
		t.Fatalf("expected apply.php: %v", err)
	}

	zipBytes, err := BuildReviewZIP("https://example.com", written)
	if err != nil {
		t.Fatalf("BuildReviewZIP: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	for _, want := range []string{"compare.html", "render-report.html", "manifest.json", "images/001/optimized.webp"} {
		if !names[want] {
			t.Fatalf("zip missing %s (have %v)", want, names)
		}
	}

	var compareHTML string
	for _, f := range zr.File {
		if f.Name != "compare.html" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		compareHTML = string(b)
		break
	}
	if !strings.Contains(compareHTML, "flex-direction:column") {
		t.Fatalf("compare.html should stack Before/After vertically")
	}
	if strings.Contains(compareHTML, "grid-template-columns:1fr 1fr") {
		t.Fatalf("compare.html still uses side-by-side grid")
	}
	if !strings.Contains(compareHTML, `class="ctx"`) || !strings.Contains(compareHTML, `target="_blank"`) {
		t.Fatalf("compare.html should include live page/file context links")
	}
}

func TestWebpRelFromHint(t *testing.T) {
	got := webpRelFromHint("2024/03/hero.png", "hero.png")
	if got != "2024/03/hero.webp" {
		t.Fatalf("got %q", got)
	}
}

func TestConcurrentHeavyExportSpeed(t *testing.T) {
	// Create mock HTTP server serving a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for x := 0; x < 100; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: 150, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Simulate 50 heavy images
	const testCount = 50
	images := make([]ManifestImage, testCount)
	for i := 0; i < testCount; i++ {
		images[i] = ManifestImage{
			SourceURL:              ts.URL + "/wp-content/uploads/2026/08/img-" + string(rune('a'+(i%26))) + ".png",
			PathHint:               "2026/08/img.png",
			Basename:               "img.png",
			WebpRel:                "2026/08/img.webp",
			Format:                 "png",
			IsHeavy:                true,
			MaxRenderedWidth:       350,
			MaxRenderedHeight:      350,
			RecommendedRetinaWidth: 700,
		}
	}

	start := time.Now()
	written, err := ConvertHeavyImagesWithThreshold(images, 80, 0, true, true, "", "")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ConvertHeavyImagesWithThreshold failed: %v", err)
	}
	if len(written) != testCount {
		t.Fatalf("expected %d written images, got %d", testCount, len(written))
	}

	// Verify IDs and deterministic ordering
	for i, w := range written {
		expectedID := string(rune('0'+(i+1)/100)) + string(rune('0'+((i+1)/10)%10)) + string(rune('0'+(i+1)%10))
		if w.ID != expectedID {
			t.Errorf("expected ID %s for item %d, got %s", expectedID, i, w.ID)
		}
	}

	t.Logf("=== EXPORT BENCHMARK: Converted %d images concurrently in %v (%.2f ms/image) ===",
		testCount, elapsed, float64(elapsed.Milliseconds())/float64(testCount))
}

func TestBenchmark137ImagesFromDump(t *testing.T) {
	dumpDir := "/Users/dmytrobuhaiov/Downloads/speedmap-webp-20260812-192446"
	if _, err := os.Stat(dumpDir); err != nil {
		t.Skip("Dump directory not found, skipping dump benchmark")
	}

	// Serve the dump directory via mock HTTP
	ts := httptest.NewServer(http.FileServer(http.Dir(dumpDir)))
	defer ts.Close()

	// Read manifest
	manifestData, err := os.ReadFile(filepath.Join(dumpDir, "manifest.json"))
	if err != nil {
		t.Skipf("skipping dump benchmark: %v", err)
	}

	type rawMani struct {
		Images []struct {
			ID        string   `json:"id"`
			SourceURL string   `json:"sourceUrl"`
			PathHint  string   `json:"pathHint"`
			WebpRel   string   `json:"webpRel"`
			Basename  string   `json:"basename"`
			Pages     []string `json:"pages"`
		} `json:"images"`
	}

	var m rawMani
	if err := json.Unmarshal(manifestData, &m); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	manifestImages := make([]ManifestImage, len(m.Images))
	for i, im := range m.Images {
		// Map source to local HTTP server
		localURL := fmt.Sprintf("%s/images/%s/original.png", ts.URL, im.ID)
		if _, err := os.Stat(filepath.Join(dumpDir, "images", im.ID, "original.jpg")); err == nil {
			localURL = fmt.Sprintf("%s/images/%s/original.jpg", ts.URL, im.ID)
		}
		manifestImages[i] = ManifestImage{
			SourceURL:              localURL,
			PathHint:               im.PathHint,
			WebpRel:                im.WebpRel,
			Basename:               im.Basename,
			Format:                 "png",
			IsHeavy:                true,
			Pages:                  im.Pages,
			MaxRenderedWidth:       350,
			MaxRenderedHeight:      350,
			RecommendedRetinaWidth: 700,
		}
	}

	start := time.Now()
	written, err := ConvertHeavyImagesWithThreshold(manifestImages, 80, 100*1024, true, true, "", "")
	convElapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ConvertHeavyImagesWithThreshold failed: %v", err)
	}
	if len(written) == 0 {
		t.Fatalf("no images converted")
	}

	outDir := t.TempDir()
	out, err := WriteDeployPackage(filepath.Join(outDir, "pkg"), "uat.infuse.com", config.ScanConfig{WebPQuality: 80}, written)
	totalElapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WriteDeployPackage failed: %v", err)
	}

	t.Logf("\n===========================================================")
	t.Logf("🚀 FULL 137-IMAGE EXPORT BENCHMARK COMPLETED:")
	t.Logf("  - Converted Images : %d / %d", len(written), len(manifestImages))
	t.Logf("  - Conversion Time  : %v (%.2f ms/image)", convElapsed, float64(convElapsed.Milliseconds())/float64(len(written)))
	t.Logf("  - Total Export Time: %v (including ZIP & Deploy package build)", totalElapsed)
	t.Logf("  - Package Dir      : %s", out.PackageDir)
	t.Logf("  - Review ZIP       : %s", out.ReviewZIP)
	t.Logf("===========================================================\n")
}

func TestConvertHeavyImagesWithOverrides(t *testing.T) {
	// Create mock HTTP server serving 3 images
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 2), B: 120, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("png encode failed: %v", err)
	}
	payload := pngBuf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(payload)
	}))
	defer ts.Close()

	images := []ManifestImage{
		{
			SourceURL: ts.URL + "/default.png",
			Basename:  "default.png",
			WebpRel:   "default.webp",
			Format:    "png",
			IsHeavy:   true,
			Bytes:     int64(len(payload)),
		},
		{
			SourceURL: ts.URL + "/tuned.png",
			Basename:  "tuned.png",
			WebpRel:   "tuned.webp",
			Format:    "png",
			IsHeavy:   true,
			Bytes:     int64(len(payload)),
		},
		{
			SourceURL: ts.URL + "/skipped.png",
			Basename:  "skipped.png",
			WebpRel:   "skipped.webp",
			Format:    "png",
			IsHeavy:   true,
			Bytes:     int64(len(payload)),
		},
	}

	overrides := map[string]ImageOverride{
		ts.URL + "/tuned.png": {
			Quality:  93,
			Lossless: false,
		},
		ts.URL + "/skipped.png": {
			Skip: true,
		},
	}

	written, err := ConvertHeavyImagesWithProgressAndOverrides(
		images,
		80, 70, false, 0, false, false,
		"", "",
		overrides,
		nil,
	)
	if err != nil {
		t.Fatalf("ConvertHeavyImagesWithProgressAndOverrides failed: %v", err)
	}

	// Skipped image must not be in written list
	if len(written) != 2 {
		t.Fatalf("expected 2 images (skipped excluded), got %d", len(written))
	}

	foundTuned := false
	for _, w := range written {
		if w.SourceURL == ts.URL+"/skipped.png" {
			t.Errorf("skipped image was included in output!")
		}
		if w.SourceURL == ts.URL+"/tuned.png" {
			foundTuned = true
			if !w.IsOverridden {
				t.Errorf("expected IsOverridden=true for tuned image")
			}
			if w.Quality != 93 {
				t.Errorf("expected tuned quality 93, got %f", w.Quality)
			}
		}
	}
	if !foundTuned {
		t.Errorf("tuned image not found in results")
	}
}

