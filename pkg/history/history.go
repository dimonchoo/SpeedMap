package history

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

var customHistoryDir string

func getHistoryDir() (string, error) {
	if customHistoryDir != "" {
		return customHistoryDir, nil
	}
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

type ScanRunSummary struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	FormattedTime string    `json:"formattedTime"`
	Domain        string    `json:"domain"`
	TotalURLs     int       `json:"totalUrls"`
	HealthScore   int       `json:"healthScore"`
	TotalImages   int       `json:"totalImages"`
}

type FileDiff struct {
	URL               string   `json:"url"`
	Basename          string   `json:"basename"`
	OriginalBytes     int64    `json:"originalBytes"`
	OriginalFormatted string   `json:"originalFormatted"`
	BaseBytes         int64    `json:"baseBytes"`
	BaseFormatted     string   `json:"baseFormatted"`
	CurrentBytes      int64    `json:"currentBytes"`
	CurrentFormatted  string   `json:"currentFormatted"`
	DeltaBytes        int64    `json:"deltaBytes"`
	DeltaFormatted    string   `json:"deltaFormatted"`
	Status            string   `json:"status"` // "degraded", "improved", "same", "new", "removed"
	Pages             []string `json:"pages,omitempty"`
}

type RunsDiffResult struct {
	BaseRunID         string     `json:"baseRunId"`
	BaseRunTime       string     `json:"baseRunTime"`
	CurrentRunID      string     `json:"currentRunId"`
	CurrentRunTime    string     `json:"currentRunTime"`
	TotalFiles        int        `json:"totalFiles"`
	DegradedCount     int        `json:"degradedCount"`
	ImprovedCount     int        `json:"improvedCount"`
	SameCount         int        `json:"sameCount"`
	NewCount          int        `json:"newCount"`
	RemovedCount      int        `json:"removedCount"`
	BaseTotalBytes    int64      `json:"baseTotalBytes"`
	CurrentTotalBytes int64      `json:"currentTotalBytes"`
	DeltaTotalBytes   int64      `json:"deltaTotalBytes"`
	Files             []FileDiff `json:"files"`
}

// GetAllHistoryRuns returns summaries of all historical runs, sorted newest first
func GetAllHistoryRuns(domain string) ([]ScanRunSummary, error) {
	dir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}

	cleanDomain := ""
	if domain != "" {
		cleanDomain = sanitizeDomain(domain)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var summaries []ScanRunSummary
	for _, f := range files {
		if f.IsDir() || !strings.HasPrefix(f.Name(), "run_") || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		if cleanDomain != "" && !strings.HasPrefix(f.Name(), fmt.Sprintf("run_%s_", cleanDomain)) {
			continue
		}

		filePath := filepath.Join(dir, f.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var run ScanRun
		if err := json.Unmarshal(data, &run); err == nil {
			imgCount := 0
			for _, r := range run.Results {
				imgCount += len(r.Diagnostics.LargestImages)
			}
			summaries = append(summaries, ScanRunSummary{
				ID:            run.ID,
				Timestamp:     run.Timestamp,
				FormattedTime: run.FormattedTime,
				Domain:        run.Domain,
				TotalURLs:     run.TotalURLs,
				HealthScore:   run.HealthScore,
				TotalImages:   imgCount,
			})
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})

	return summaries, nil
}

var (
	historyDiffMu    sync.RWMutex
	historyDiffCache = make(map[string]*RunsDiffResult)
)

// CompareHistoryRuns performs a file-by-file regression comparison between two run IDs
func CompareHistoryRuns(baseRunID, currentRunID string) (*RunsDiffResult, error) {
	cacheKey := fmt.Sprintf("%s:%s", baseRunID, currentRunID)
	historyDiffMu.RLock()
	if cached, ok := historyDiffCache[cacheKey]; ok {
		historyDiffMu.RUnlock()
		return cached, nil
	}
	historyDiffMu.RUnlock()

	dir, err := getHistoryDir()
	if err != nil {
		return nil, err
	}

	loadRun := func(id string) (*ScanRun, error) {
		path := filepath.Join(dir, fmt.Sprintf("run_%s.json", id))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var r ScanRun
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, err
		}
		return &r, nil
	}

	baseRun, err := loadRun(baseRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to load base run %s: %w", baseRunID, err)
	}
	currentRun, err := loadRun(currentRunID)
	if err != nil {
		return nil, fmt.Errorf("failed to load current run %s: %w", currentRunID, err)
	}

	// Extract unique images from base run
	baseImages := make(map[string]scanner.ImageDetail)
	for _, p := range baseRun.Results {
		for _, im := range p.Diagnostics.LargestImages {
			if im.URL != "" {
				baseImages[im.URL] = im
			}
		}
	}

	// Extract unique images from current run
	currentImages := make(map[string]scanner.ImageDetail)
	for _, p := range currentRun.Results {
		for _, im := range p.Diagnostics.LargestImages {
			if im.URL != "" {
				currentImages[im.URL] = im
			}
		}
	}

	allURLs := make(map[string]bool)
	for u := range baseImages {
		allURLs[u] = true
	}
	for u := range currentImages {
		allURLs[u] = true
	}

	formatKB := func(b int64) string {
		if b >= 1024*1024 {
			return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
		}
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}

	var files []FileDiff
	var degraded, improved, same, newCount, removedCount int
	var baseTotal, currentTotal int64

	for u := range allURLs {
		bIm, hasBase := baseImages[u]
		cIm, hasCurr := currentImages[u]

		basename := u
		if idx := strings.LastIndex(u, "/"); idx != -1 && idx < len(u)-1 {
			basename = u[idx+1:]
		}

		var bBytes, cBytes, origBytes int64
		if hasBase {
			bBytes = bIm.TransferSize
			if bBytes == 0 {
				bBytes = bIm.EncodedSize
			}
			origBytes = bBytes
		}
		if hasCurr {
			cBytes = cIm.TransferSize
			if cBytes == 0 {
				cBytes = cIm.EncodedSize
			}
			if origBytes == 0 {
				origBytes = cBytes
			}
		}

		// FILTER OUT DUMMY / UNMEASURED IMAGES:
		// If neither run transferred this image over the network (0 B in both),
		// it was never loaded by the browser and should not pollute the diff.
		if bBytes == 0 && cBytes == 0 {
			continue
		}

		baseTotal += bBytes
		currentTotal += cBytes

		var delta int64
		status := "same"
		var deltaFormatted string

		if !hasBase && hasCurr {
			status = "new"
			newCount++
			deltaFormatted = "🆕 Новий"
		} else if hasBase && !hasCurr {
			status = "removed"
			removedCount++
			deltaFormatted = "🗑️ Видалено"
		} else if bBytes == 0 && cBytes > 0 {
			status = "new"
			newCount++
			deltaFormatted = "🆕 Завантажено"
		} else if bBytes > 0 && cBytes == 0 {
			status = "removed"
			removedCount++
			deltaFormatted = "⏸️ Не завантаж."
		} else {
			delta = cBytes - bBytes
			if delta > 5*1024 {
				status = "degraded"
				degraded++
			} else if delta < -5*1024 {
				status = "improved"
				improved++
			} else {
				status = "same"
				same++
			}
			deltaFormatted = formatKB(delta)
			if delta > 0 {
				deltaFormatted = "+" + deltaFormatted
			}
		}

		baseFmt := formatKB(bBytes)
		if bBytes == 0 {
			baseFmt = "—"
		}
		currFmt := formatKB(cBytes)
		if cBytes == 0 {
			currFmt = "—"
		}

		files = append(files, FileDiff{
			URL:               u,
			Basename:          basename,
			OriginalBytes:     origBytes,
			OriginalFormatted: formatKB(origBytes),
			BaseBytes:         bBytes,
			BaseFormatted:     baseFmt,
			CurrentBytes:      cBytes,
			CurrentFormatted:  currFmt,
			DeltaBytes:        delta,
			DeltaFormatted:    deltaFormatted,
			Status:            status,
		})
	}

	// Priority sorting:
	// 1. Degraded (largest regression first)
	// 2. Improved (largest savings first)
	// 3. New (largest current file size first)
	// 4. Removed (largest base file size first)
	// 5. Same (largest file size first)
	statusRank := func(s string) int {
		switch s {
		case "degraded":
			return 0
		case "improved":
			return 1
		case "new":
			return 2
		case "removed":
			return 3
		default:
			return 4
		}
	}

	sort.Slice(files, func(i, j int) bool {
		ri := statusRank(files[i].Status)
		rj := statusRank(files[j].Status)
		if ri != rj {
			return ri < rj
		}
		if files[i].Status == "degraded" {
			return files[i].DeltaBytes > files[j].DeltaBytes
		}
		if files[i].Status == "improved" {
			return files[i].DeltaBytes < files[j].DeltaBytes
		}
		if files[i].Status == "new" {
			return files[i].CurrentBytes > files[j].CurrentBytes
		}
		if files[i].Status == "removed" {
			return files[i].BaseBytes > files[j].BaseBytes
		}
		return files[i].CurrentBytes > files[j].CurrentBytes
	})

	res := &RunsDiffResult{
		BaseRunID:         baseRunID,
		BaseRunTime:       baseRun.FormattedTime,
		CurrentRunID:      currentRunID,
		CurrentRunTime:    currentRun.FormattedTime,
		TotalFiles:        len(files),
		DegradedCount:     degraded,
		ImprovedCount:     improved,
		SameCount:         same,
		NewCount:          newCount,
		RemovedCount:      removedCount,
		BaseTotalBytes:    baseTotal,
		CurrentTotalBytes: currentTotal,
		DeltaTotalBytes:   currentTotal - baseTotal,
		Files:             files,
	}

	historyDiffMu.Lock()
	historyDiffCache[cacheKey] = res
	historyDiffMu.Unlock()

	return res, nil
}
