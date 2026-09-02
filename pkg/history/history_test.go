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

func TestCompareHistoryRunsLazyLoadedNotDegraded(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "speedmap-history-diff-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// In base run, image was lazy-loaded in DOM with 0 transferSize
	baseRun := fmt.Sprintf(`{
		"id": "run_base_1",
		"domain": "test.com",
		"formattedTime": "01.09.2026 10:00",
		"results": [
			{
				"url": "https://test.com/page1",
				"diagnostics": {
					"largestImages": [
						{"url": "https://test.com/hero.png", "transferSize": 0, "encodedSize": 0}
					]
				}
			}
		]
	}`)
	// In current run, image was transferred over network with 900KB
	currRun := fmt.Sprintf(`{
		"id": "run_curr_2",
		"domain": "test.com",
		"formattedTime": "02.09.2026 10:00",
		"results": [
			{
				"url": "https://test.com/page1",
				"diagnostics": {
					"largestImages": [
						{"url": "https://test.com/hero.png", "transferSize": 900000, "encodedSize": 900000}
					]
				}
			}
		]
	}`)

	_ = os.WriteFile(filepath.Join(tempDir, "run_run_base_1.json"), []byte(baseRun), 0644)
	_ = os.WriteFile(filepath.Join(tempDir, "run_run_curr_2.json"), []byte(currRun), 0644)

	// Save history dir override or load directly
	// Let's test the logic directly by checking files returned
	origDir := customHistoryDir
	customHistoryDir = tempDir
	defer func() { customHistoryDir = origDir }()

	res, err := CompareHistoryRuns("run_base_1", "run_curr_2")
	if err != nil {
		t.Fatalf("CompareHistoryRuns failed: %v", err)
	}

	// Must NOT be marked as degraded!
	if res.DegradedCount != 0 {
		t.Errorf("expected 0 degraded count, got %d", res.DegradedCount)
	}
	if len(res.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(res.Files))
	}
	if res.Files[0].Status != "new" {
		t.Errorf("expected status 'new' for newly transferred lazy image, got %s", res.Files[0].Status)
	}
	if res.Files[0].BaseFormatted != "—" {
		t.Errorf("expected BaseFormatted '—', got %s", res.Files[0].BaseFormatted)
	}
}

