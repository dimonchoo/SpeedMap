package history

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
)

type ScanRun struct {
	ID          string                  `json:"id"`
	Timestamp   time.Time               `json:"timestamp"`
	FormattedTime string                `json:"formattedTime"`
	Domain      string                  `json:"domain"`
	TotalURLs   int                     `json:"totalUrls"`
	HealthScore int                     `json:"healthScore"`
	Analytics   analytics.SiteAnalytics `json:"analytics"`
	Results     []scanner.PageResult    `json:"results"`
}

type RunComparison struct {
	HasPrevious      bool               `json:"hasPrevious"`
	PreviousTime     string             `json:"previousTime"`
	PreviousScore    int                `json:"previousScore"`
	CurrentScore     int                `json:"currentScore"`
	ScoreDelta       int                `json:"scoreDelta"`
	MetricDeltas     map[string]float64 `json:"metricDeltas"` // Delta in ms or unit (negative is improvement for speed)
	SummaryStatus    string             `json:"summaryStatus"` // "improved", "degraded", "unchanged"
	SummaryText      string             `json:"summaryText"`
}

func getHistoryDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".speedmap", "history")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func SaveScanRun(domain string, results []scanner.PageResult, siteAnalytics analytics.SiteAnalytics) (*ScanRun, error) {
	return SaveScanRunWithConfig(domain, results, siteAnalytics, config.ScanConfig{})
}

func SaveScanRunWithConfig(domain string, results []scanner.PageResult, siteAnalytics analytics.SiteAnalytics, cfg config.ScanConfig) (*ScanRun, error) {
	dir, err := getHistoryDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get history directory: %w", err)
	}

	cleanDomain := sanitizeDomain(domain)
	now := time.Now()
	id := fmt.Sprintf("%s_%d", cleanDomain, now.Unix())

	run := ScanRun{
		ID:            id,
		Timestamp:     now,
		FormattedTime: now.Format("02.01.2006 15:04"),
		Domain:        cleanDomain,
		TotalURLs:     len(results),
		HealthScore:   siteAnalytics.HealthScore,
		Analytics:     siteAnalytics,
		Results:       results,
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to encode scan run JSON: %w", err)
	}

	filePath := filepath.Join(dir, fmt.Sprintf("run_%s.json", id))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write history file: %w", err)
	}

	// Trigger async auto-pruning if enabled
	if cfg.IsAutoPruneEnabled() {
		maxRuns := cfg.NormalizedRetentionRuns()
		maxDays := cfg.NormalizedRetentionDays()
		go func() {
			_ = AutoPruneHistory(maxRuns, maxDays)
		}()
	}

	return &run, nil
}

func GetPreviousRun(domain string, currentRunID string) (*ScanRun, error) {
	dir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}

	cleanDomain := sanitizeDomain(domain)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var runs []ScanRun
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), fmt.Sprintf("run_%s_", cleanDomain)) && strings.HasSuffix(f.Name(), ".json") {
			filePath := filepath.Join(dir, f.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var run ScanRun
			if err := json.Unmarshal(data, &run); err == nil {
				if run.ID != currentRunID {
					runs = append(runs, run)
				}
			}
		}
	}

	if len(runs) == 0 {
		return nil, nil // No previous run found
	}

	// Sort runs by timestamp descending
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Timestamp.After(runs[j].Timestamp)
	})

	return &runs[0], nil
}

func CompareRuns(current *ScanRun, previous *ScanRun) RunComparison {
	if previous == nil {
		return RunComparison{
			HasPrevious:   false,
			CurrentScore:  current.HealthScore,
			SummaryStatus: "new",
			SummaryText:   "Перше сканування даного домену. Історія буде порівнюватися з наступними прогонами.",
		}
	}

	scoreDelta := current.HealthScore - previous.HealthScore
	metricDeltas := make(map[string]float64)

	metrics := []string{"TTFB", "FCP", "LCP", "CLS", "TBT"}
	for _, m := range metrics {
		currVal := current.Analytics.AverageMetrics[m]
		prevVal := previous.Analytics.AverageMetrics[m]
		if currVal > 0 && prevVal > 0 {
			metricDeltas[m] = math.Round((currVal-prevVal)*100) / 100
		}
	}

	status := "unchanged"
	var text string
	if scoreDelta > 0 {
		status = "improved"
		text = fmt.Sprintf("🟢 Оцінка продуктивності покращилася на +%d балів (з %d до %d)", scoreDelta, previous.HealthScore, current.HealthScore)
	} else if scoreDelta < 0 {
		status = "degraded"
		text = fmt.Sprintf("🔴 Оцінка продуктивності знизилася на %d балів (з %d до %d)", scoreDelta, previous.HealthScore, current.HealthScore)
	} else {
		text = fmt.Sprintf("🟡 Оцінка продуктивності без змін (%d балів)", current.HealthScore)
	}

	return RunComparison{
		HasPrevious:   true,
		PreviousTime:  previous.FormattedTime,
		PreviousScore: previous.HealthScore,
		CurrentScore:  current.HealthScore,
		ScoreDelta:    scoreDelta,
		MetricDeltas:  metricDeltas,
		SummaryStatus: status,
		SummaryText:   text,
	}
}

// AutoPruneHistory cleans up old or excessive history run files in ~/.speedmap/history/.
// Retains at most maxRunsPerDomain (default 20) runs per domain and removes runs older
// than maxAgeDays (default 30 days), always preserving the latest run.
func AutoPruneHistory(maxRunsPerDomain, maxAgeDays int) error {
	dir, err := getHistoryDir()
	if err != nil {
		return err
	}
	return AutoPruneHistoryInDir(dir, maxRunsPerDomain, maxAgeDays)
}

// AutoPruneHistoryInDir cleans up history runs within a specific directory.
// Set maxRunsPerDomain = 0 or maxAgeDays = 0 for unlimited.
func AutoPruneHistoryInDir(dir string, maxRunsPerDomain, maxAgeDays int) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type runFile struct {
		path      string
		domain    string
		timestamp time.Time
	}

	domainFiles := make(map[string][]runFile)
	var cutoffTime time.Time
	if maxAgeDays > 0 {
		cutoffTime = time.Now().AddDate(0, 0, -maxAgeDays)
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "run_") || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dir, f.Name())
		info, err := f.Info()
		if err != nil {
			continue
		}

		base := strings.TrimPrefix(f.Name(), "run_")
		base = strings.TrimSuffix(base, ".json")
		lastUnderscore := strings.LastIndex(base, "_")

		domainKey := "unknown"
		ts := info.ModTime()

		if lastUnderscore > 0 {
			domainKey = base[:lastUnderscore]
			var unixSec int64
			if _, err := fmt.Sscanf(base[lastUnderscore+1:], "%d", &unixSec); err == nil && unixSec > 0 {
				ts = time.Unix(unixSec, 0)
			}
		}

		domainFiles[domainKey] = append(domainFiles[domainKey], runFile{
			path:      filePath,
			domain:    domainKey,
			timestamp: ts,
		})
	}

	for _, runs := range domainFiles {
		sort.Slice(runs, func(i, j int) bool {
			return runs[i].timestamp.After(runs[j].timestamp)
		})

		for idx, rf := range runs {
			if idx == 0 {
				continue // Always preserve the newest run
			}

			shouldDelete := false
			if maxRunsPerDomain > 0 && idx >= maxRunsPerDomain {
				shouldDelete = true
			}
			if maxAgeDays > 0 && !cutoffTime.IsZero() && rf.timestamp.Before(cutoffTime) {
				shouldDelete = true
			}

			if shouldDelete {
				_ = os.Remove(rf.path)
			}
		}
	}

	return nil
}

func sanitizeDomain(rawURL string) string {
	d := strings.TrimPrefix(rawURL, "https://")
	d = strings.TrimPrefix(d, "http://")
	idx := strings.Index(d, "/")
	if idx > 0 {
		d = d[:idx]
	}
	d = strings.ReplaceAll(d, ":", "_")
	d = strings.ReplaceAll(d, ".", "_")
	if d == "" {
		return "site"
	}
	return d
}
