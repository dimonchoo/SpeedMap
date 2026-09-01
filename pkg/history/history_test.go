package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestAutoPruneHistory(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create 25 runs for domain "example_com"
	for i := 1; i <= 25; i++ {
		runTime := now.Add(time.Duration(-i) * time.Hour)
		fileName := fmt.Sprintf("run_example_com_%d.json", runTime.Unix())
		runData := fmt.Sprintf(`{"id": "example_com_%d", "domain": "example_com"}`, runTime.Unix())
		_ = os.WriteFile(filepath.Join(tempDir, fileName), []byte(runData), 0644)
	}

	// Create 1 very old run (40 days old) for domain "old_site_com"
	oldTime := now.AddDate(0, 0, -40)
	oldFileName := fmt.Sprintf("run_old_site_com_%d.json", oldTime.Unix())
	_ = os.WriteFile(filepath.Join(tempDir, oldFileName), []byte(`{"id": "old_site"}`), 0644)

	// Run AutoPrune with maxRuns=20, maxAgeDays=30
	if err := AutoPruneHistoryInDir(tempDir, 20, 30); err != nil {
		t.Fatalf("AutoPruneHistoryInDir failed: %v", err)
	}

	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	exampleCount := 0
	oldSiteCount := 0
	for _, f := range files {
		if strings.Contains(f.Name(), "example_com") {
			exampleCount++
		}
		if strings.Contains(f.Name(), "old_site_com") {
			oldSiteCount++
		}
	}

	if exampleCount != 20 {
		t.Errorf("Expected exactly 20 runs for example_com, got %d", exampleCount)
	}

	// The old site run is the ONLY run for old_site_com, so it must be preserved
	if oldSiteCount != 1 {
		t.Errorf("Expected latest run of old_site_com to be preserved, got %d", oldSiteCount)
	}
}
