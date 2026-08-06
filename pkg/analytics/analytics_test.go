package analytics

import (
	"testing"

	"SpeedMap/pkg/scanner"
)

func TestComputeSiteAnalytics(t *testing.T) {
	pages := []scanner.PageResult{
		{
			ID:            1,
			URL:           "https://example.com/page1",
			StatusCode:    200,
			OverallStatus: "good",
			Metrics: scanner.WebVitals{
				TTFB: 300,
				FCP:  800,
				LCP:  1200,
				CLS:  0.01,
				TBT:  50,
			},
			Diagnostics: scanner.PageDiagnostics{
				SlowestResources: []scanner.ResourceTiming{
					{Name: "https://example.com/js/app.js", Type: "script", Duration: 350},
				},
				LargestImages: []scanner.ImageDetail{
					{URL: "https://example.com/img/hero.jpg", TransferSize: 524288, Duration: 250, Width: 1200, Height: 800, FormattedSize: "512 KB"},
				},
				Fonts: []scanner.FontDetail{
					{Family: "Inter", URL: "https://example.com/fonts/inter.woff2", Type: "woff2", TransferSize: 25000, Duration: 60},
				},
				Iframes: []scanner.IframeDetail{
					{Src: "https://www.youtube.com/embed/abc", Title: "Video", Width: 560, Height: 315, LoadedDuringScan: true, Duration: 120},
					{Src: "https://maps.example.com/embed", Title: "Map", Width: 600, Height: 400, IsLazy: true, LoadedDuringScan: false},
				},
			},
		},
		{
			ID:            2,
			URL:           "https://example.com/page2",
			StatusCode:    200,
			OverallStatus: "needs-improvement",
			Metrics: scanner.WebVitals{
				TTFB: 900,
				FCP:  1800,
				LCP:  3200,
				CLS:  0.15,
				TBT:  250,
			},
			Diagnostics: scanner.PageDiagnostics{
				SlowestResources: []scanner.ResourceTiming{
					{Name: "https://example.com/js/app.js", Type: "script", Duration: 400},
					{Name: "https://example.com/fonts/inter.woff2", Type: "font", Duration: 200},
				},
				LargestImages: []scanner.ImageDetail{
					{URL: "https://example.com/img/hero.jpg", TransferSize: 524288, Duration: 280, Width: 1200, Height: 800, FormattedSize: "512 KB"},
					{URL: "https://example.com/img/banner.png", TransferSize: 1048576, Duration: 450, Width: 1920, Height: 1080, FormattedSize: "1.0 MB"},
				},
				Fonts: []scanner.FontDetail{
					{Family: "Inter", URL: "https://example.com/fonts/inter.woff2", Type: "woff2", TransferSize: 25000, Duration: 80},
					{Family: "Roboto", URL: "https://example.com/fonts/roboto.woff2", Type: "woff2", TransferSize: 30000, Duration: 90},
				},
				Iframes: []scanner.IframeDetail{
					{Src: "https://www.youtube.com/embed/abc", Title: "Video", Width: 560, Height: 315, LoadedDuringScan: false, IsLazy: true},
				},
			},
		},
	}

	analytics := ComputeSiteAnalytics(pages)

	if analytics.TotalPages != 2 {
		t.Errorf("Expected TotalPages = 2, got %d", analytics.TotalPages)
	}

	if analytics.HealthScore != 75 { // (1 * 1.0 + 1 * 0.5) / 2 * 100 = 75
		t.Errorf("Expected HealthScore = 75, got %d", analytics.HealthScore)
	}

	if len(analytics.TopResourceBottlenecks) != 2 {
		t.Fatalf("Expected 2 bottleneck resources, got %d", len(analytics.TopResourceBottlenecks))
	}

	// app.js should be first with occurrences = 2
	if analytics.TopResourceBottlenecks[0].Name != "https://example.com/js/app.js" {
		t.Errorf("Expected top resource to be app.js, got %s", analytics.TopResourceBottlenecks[0].Name)
	}
	if analytics.TopResourceBottlenecks[0].Occurrences != 2 {
		t.Errorf("Expected app.js occurrences = 2, got %d", analytics.TopResourceBottlenecks[0].Occurrences)
	}

	// Check Largest Images
	if len(analytics.LargestImages) != 2 {
		t.Fatalf("Expected 2 aggregated largest images, got %d", len(analytics.LargestImages))
	}
	if analytics.LargestImages[0].URL != "https://example.com/img/banner.png" {
		t.Errorf("Expected top image to be banner.png, got %s", analytics.LargestImages[0].URL)
	}
	if analytics.LargestImages[1].PageCount != 2 {
		t.Errorf("Expected hero.jpg page count = 2, got %d", analytics.LargestImages[1].PageCount)
	}
	if len(analytics.LargestImages[1].Pages) != 2 {
		t.Fatalf("Expected hero.jpg on 2 pages, got %v", analytics.LargestImages[1].Pages)
	}
	wantPages := map[string]bool{"https://example.com/page1": true, "https://example.com/page2": true}
	for _, pg := range analytics.LargestImages[1].Pages {
		if !wantPages[pg] {
			t.Errorf("unexpected page %s", pg)
		}
	}

	// Check Font Usage
	if len(analytics.FontUsage) != 2 {
		t.Fatalf("Expected 2 fonts aggregated, got %d", len(analytics.FontUsage))
	}
	if analytics.FontUsage[0].Family != "Inter" {
		t.Errorf("Expected top font to be Inter, got %s", analytics.FontUsage[0].Family)
	}
	if analytics.FontUsage[0].Occurrences != 2 || analytics.FontUsage[0].Percentage != 100 {
		t.Errorf("Expected Inter font occurrences=2, percentage=100%%, got %d (%.1f%%)", analytics.FontUsage[0].Occurrences, analytics.FontUsage[0].Percentage)
	}

	if analytics.TotalIframeCount != 2 {
		t.Errorf("Expected TotalIframeCount = 2, got %d", analytics.TotalIframeCount)
	}
	if analytics.MissedIframeCount != 2 {
		t.Errorf("Expected MissedIframeCount = 2, got %d", analytics.MissedIframeCount)
	}
	if analytics.LoadedIframeCount != 1 {
		t.Errorf("Expected LoadedIframeCount = 1, got %d", analytics.LoadedIframeCount)
	}
	if len(analytics.Iframes) != 2 {
		t.Fatalf("Expected 2 aggregated iframes, got %d", len(analytics.Iframes))
	}

	if len(analytics.GlobalFixes) == 0 {
		t.Errorf("Expected global fixes to be generated")
	}

	// Verify SEOAEO-235 Image Optimization Analytics
	if analytics.TotalImageCount != 2 {
		t.Errorf("Expected TotalImageCount = 2, got %d", analytics.TotalImageCount)
	}
	if analytics.HeavyImagesCount != 2 {
		t.Errorf("Expected HeavyImagesCount = 2 (>100KB), got %d", analytics.HeavyImagesCount)
	}
	if analytics.TotalWebPSavingsBytes <= 0 {
		t.Errorf("Expected positive WebP savings, got %d", analytics.TotalWebPSavingsBytes)
	}

	// Test GenerateImageComparisonHTML
	htmlReport := GenerateImageComparisonHTML(analytics, "example.com")
	if len(htmlReport) == 0 {
		t.Fatalf("Expected non-empty HTML report string")
	}
	if !testing.Short() && len(htmlReport) < 100 {
		t.Errorf("Report HTML seems unexpectedly small: %d chars", len(htmlReport))
	}
}
