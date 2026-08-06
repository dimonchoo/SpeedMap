package wpexport

import (
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
		{URL: "https://example.com/wp-content/uploads/2024/03/hero-1024x768.png", Format: "png", IsHeavy: false, MaxTransferSize: 100000},
		{URL: "https://example.com/wp-content/uploads/logo.svg", Format: "svg", IsHeavy: false},
		{URL: "https://example.com/wp-content/uploads/ok.webp", Format: "webp", IsHeavy: false},
	}

	php, err := BuildApplyPHP("https://example.com", cfg, images, "/var/www/site")
	if err != nil {
		t.Fatalf("BuildApplyPHP: %v", err)
	}
	if !strings.Contains(php, "hero.png") {
		t.Fatalf("expected hero.png in manifest")
	}
	if !strings.Contains(php, "2024/03/hero.png") {
		t.Fatalf("expected pathHint 2024/03/hero.png")
	}
	if !strings.Contains(php, "wp eval-file") {
		t.Fatalf("expected runbook comment")
	}
	if !strings.Contains(php, "--path=/var/www/site") {
		t.Fatalf("expected wordpress path in header")
	}
	if !strings.Contains(php, `"wordpressPath": "/var/www/site"`) && !strings.Contains(php, `"wordpressPath":"/var/www/site"`) {
		t.Fatalf("expected wordpressPath in JSON")
	}
	if !strings.Contains(php, "https://example.com/about") {
		t.Fatalf("expected page URL in manifest")
	}
	if strings.Contains(php, "{{SPEEDMAP_MANIFEST_JSON}}") || strings.Contains(php, "{{WORDPRESS_PATH}}") {
		t.Fatalf("placeholder not replaced")
	}
	if !strings.Contains(php, `"quality": 80`) && !strings.Contains(php, `"quality":80`) {
		t.Fatalf("expected quality in JSON")
	}
}

func TestBuildApplyPHPRequiresPath(t *testing.T) {
	_, err := BuildApplyPHP("https://example.com", config.ScanConfig{}, []analytics.AggregatedImage{
		{URL: "https://example.com/a.png", Format: "png"},
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
