package history

import (
	"testing"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/scanner"
)

func TestCompareRuns(t *testing.T) {
	prevRun := &ScanRun{
		ID:            "test_1",
		Timestamp:     time.Now().Add(-24 * time.Hour),
		FormattedTime: "03.08.2026 18:00",
		HealthScore:   60,
		Analytics: analytics.SiteAnalytics{
			AverageMetrics: map[string]float64{
				"TTFB": 600,
				"LCP":  3000,
			},
		},
	}

	currRun := &ScanRun{
		ID:            "test_2",
		Timestamp:     time.Now(),
		FormattedTime: "04.08.2026 18:00",
		HealthScore:   80,
		Analytics: analytics.SiteAnalytics{
			AverageMetrics: map[string]float64{
				"TTFB": 400,
				"LCP":  2200,
			},
		},
	}

	cmp := CompareRuns(currRun, prevRun)

	if !cmp.HasPrevious {
		t.Errorf("Expected HasPrevious = true")
	}
	if cmp.ScoreDelta != 20 {
		t.Errorf("Expected ScoreDelta = 20, got %d", cmp.ScoreDelta)
	}
	if cmp.SummaryStatus != "improved" {
		t.Errorf("Expected SummaryStatus = 'improved', got %s", cmp.SummaryStatus)
	}
	if cmp.MetricDeltas["LCP"] != -800 { // 2200 - 3000 = -800
		t.Errorf("Expected LCP delta = -800, got %.2f", cmp.MetricDeltas["LCP"])
	}
}

func TestSaveAndFetchHistory(t *testing.T) {
	results := []scanner.PageResult{
		{ID: 1, URL: "https://test-domain.com/1", OverallStatus: "good"},
	}
	siteAnalytics := analytics.ComputeSiteAnalytics(results)

	run, err := SaveScanRun("https://test-domain.com/sitemap.xml", results, siteAnalytics)
	if err != nil {
		t.Fatalf("SaveScanRun error: %v", err)
	}

	if run.Domain != "test-domain_com" {
		t.Errorf("Expected sanitized domain 'test-domain_com', got %s", run.Domain)
	}
}
