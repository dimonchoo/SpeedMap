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
