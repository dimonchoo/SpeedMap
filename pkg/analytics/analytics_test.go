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

	if len(analytics.GlobalFixes) == 0 {
		t.Errorf("Expected global fixes to be generated")
	}
}
