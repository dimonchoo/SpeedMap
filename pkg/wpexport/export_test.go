package wpexport

import (
	"archive/zip"
	"bytes"
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
		{URL: "https://ex.com/wp-content/uploads/b.png", Format: "png", IsHeavy: false},
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
	for _, want := range []string{"compare.html", "manifest.json", "images/001/original.png", "images/001/optimized.webp"} {
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
