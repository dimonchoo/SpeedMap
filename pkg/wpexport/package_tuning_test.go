package wpexport

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
)

func TestLoadPackageForStudio(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "speedmap_pkg_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	imagesDir := filepath.Join(tmpDir, "images", "001")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		t.Fatalf("failed to create images dir: %v", err)
	}
	testWebPFile := filepath.Join(imagesDir, "optimized.webp")
	if err := os.WriteFile(testWebPFile, []byte("RIFFtestWEBP"), 0644); err != nil {
		t.Fatalf("failed to write dummy webp: %v", err)
	}

	manifestContent := `{
		"domain": "https://example.com/sitemap.xml",
		"generated": "2026-09-08T12:00:00Z",
		"count": 1,
		"images": [
			{
				"id": "001",
				"sourceUrl": "https://example.com/wp-content/uploads/2026/01/banner.png",
				"pathHint": "2026/01/banner.png",
				"webpRel": "2026/01/banner.webp",
				"basename": "banner.png",
				"pages": ["https://example.com/page1"],
				"naturalWidth": 1200,
				"naturalHeight": 600,
				"optimizedWidth": 1200,
				"optimizedHeight": 600,
				"originalBytes": 500000,
				"optimizedBytes": 50000,
				"savingsPercent": 90.0,
				"originalFormatted": "488.3 KB",
				"optimizedFormatted": "48.8 KB",
				"originalPath": "https://example.com/wp-content/uploads/2026/01/banner.png",
				"optimizedPath": "images/001/optimized.webp"
			}
		]
	}`

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	ctx, err := LoadPackageForStudio(tmpDir)
	if err != nil {
		t.Fatalf("LoadPackageForStudio returned error: %v", err)
	}

	if ctx.Count != 1 {
		t.Errorf("expected 1 image, got %d", ctx.Count)
	}
	if ctx.Domain != "https://example.com/sitemap.xml" {
		t.Errorf("unexpected domain: %s", ctx.Domain)
	}
	if len(ctx.Images) != 1 {
		t.Fatalf("expected 1 image item, got %d", len(ctx.Images))
	}
	img := ctx.Images[0]
	if img.ID != "001" {
		t.Errorf("expected id '001', got '%s'", img.ID)
	}
	if img.Basename != "banner.png" {
		t.Errorf("expected basename 'banner.png', got '%s'", img.Basename)
	}
	if img.URL != "https://example.com/wp-content/uploads/2026/01/banner.png" {
		t.Errorf("unexpected url: %s", img.URL)
	}
	if img.LocalWebPAbsPath != testWebPFile {
		t.Errorf("unexpected localWebPAbsPath: %s", img.LocalWebPAbsPath)
	}
}

func TestSyncCompareHTMLItems(t *testing.T) {
	sampleHTML := `<!DOCTYPE html><html><head></head><body>
	<script>
	const items = [{"id":"001","basename":"old.png"}];
	let currentIndex = 0;
	</script></body></html>`

	newImages := []map[string]interface{}{
		{
			"id":             "001",
			"basename":       "old.png",
			"optimizedBytes": int64(45000),
		},
		{
			"id":             "002",
			"basename":       "new.png",
			"optimizedBytes": int64(12000),
		},
	}

	updated := syncCompareHTMLItems(sampleHTML, newImages)
	if !strings.Contains(updated, `"id":"002"`) {
		t.Errorf("expected updated HTML to contain new item id 002, got:\n%s", updated)
	}
	if !strings.Contains(updated, "let currentIndex = 0;") {
		t.Errorf("expected updated HTML to preserve downstream scripts, got:\n%s", updated)
	}
}

func TestSaveTunedImageToPackage(t *testing.T) {
	// Create mock image server with valid PNG
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}
	pngData := buf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
	defer ts.Close()

	tmpDir, err := os.MkdirTemp("", "speedmap_save_pkg_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	manifestContent := `{
		"domain": "https://example.com",
		"generated": "2026-09-08T12:00:00Z",
		"count": 1,
		"images": [
			{
				"id": "001",
				"sourceUrl": "` + ts.URL + `/sample.png",
				"basename": "sample.png",
				"originalBytes": 100,
				"optimizedBytes": 50,
				"optimizedPath": "images/001/optimized.webp"
			}
		]
	}`

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	comparePath := filepath.Join(tmpDir, "compare.html")
	if err := os.WriteFile(comparePath, []byte("<script>const items = [];\nlet currentIndex=0;</script>"), 0644); err != nil {
		t.Fatalf("failed to write compare: %v", err)
	}

	opts := optimizer.ImageTuneOptions{
		Quality:  95.0,
		Lossless: false,
		Dither:   true,
	}
	cfg := config.ScanConfig{}

	res, err := SaveTunedImageToPackage(tmpDir, "001", ts.URL+"/sample.png", opts, cfg)
	if err != nil {
		t.Fatalf("SaveTunedImageToPackage error: %v", err)
	}

	if res.ID != "001" {
		t.Errorf("expected ID '001', got '%s'", res.ID)
	}
	if res.OptimizedBytes <= 0 {
		t.Errorf("expected positive optimizedBytes, got %d", res.OptimizedBytes)
	}

	// Verify file was written on disk
	savedWebP := filepath.Join(tmpDir, "images", "001", "optimized.webp")
	if _, err := os.Stat(savedWebP); err != nil {
		t.Errorf("expected saved file at %s, got error: %v", savedWebP, err)
	}

	// Verify manifest.json was updated
	updatedManifestBytes, _ := os.ReadFile(manifestPath)
	var raw struct {
		Images []map[string]interface{} `json:"images"`
	}
	_ = json.Unmarshal(updatedManifestBytes, &raw)
	if len(raw.Images) > 0 {
		if raw.Images[0]["isOverridden"] != true {
			t.Errorf("expected isOverridden to be true")
		}
	}
}

func TestRealUserPackageIfExists(t *testing.T) {
	realPath := "/Users/dmytrobuhaiov/Downloads/speedmap-webp-20260904-135055"
	if _, err := os.Stat(realPath); err != nil {
		t.Skip("real package not found")
	}
	ctx, err := LoadPackageForStudio(realPath)
	if err != nil {
		t.Fatalf("LoadPackageForStudio failed on user package: %v", err)
	}
	if ctx.Count != 719 {
		t.Errorf("expected 719 images, got %d", ctx.Count)
	}
	if len(ctx.Images) != 719 {
		t.Errorf("expected 719 images in array, got %d", len(ctx.Images))
	}
	first := ctx.Images[0]
	if first.ID != "001" {
		t.Errorf("expected first image ID 001, got %s", first.ID)
	}
	if first.Basename != "hero-22.png" {
		t.Errorf("expected first image hero-22.png, got %s", first.Basename)
	}
	if first.LocalWebPAbsPath == "" {
		t.Errorf("expected non-empty LocalWebPAbsPath")
	}
	t.Logf("Successfully verified real user package with %d images. First image: %s (%s)", ctx.Count, first.Basename, first.FormattedBytes)
}

