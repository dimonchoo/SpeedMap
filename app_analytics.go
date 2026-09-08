package main

import (
	"fmt"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/history"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/wpexport"
)

type AnalyticsResult struct {
	Analytics  analytics.SiteAnalytics `json:"analytics"`
	Comparison history.RunComparison    `json:"comparison"`
}

// ComputeSiteAnalytics computes analytics and compares with previous scan run
func (a *App) ComputeSiteAnalytics(domain string, cfg config.ScanConfig, results []scanner.PageResult) (AnalyticsResult, error) {
	fmt.Printf("[GO LOG] ComputeSiteAnalytics called for %s (%d pages, threshold=%d KB)\n", domain, len(results), cfg.HeavyImageThresholdKB)
	siteAnalytics := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)

	prevRun, _ := history.GetPreviousRun(domain, "")
	currentRun, err := history.SaveScanRunWithConfig(domain, results, siteAnalytics, cfg)
	if err != nil {
		return AnalyticsResult{
			Analytics: siteAnalytics,
		}, nil
	}

	var comparison history.RunComparison
	if prevRun != nil {
		comparison = history.CompareRuns(prevRun, currentRun)
	}

	return AnalyticsResult{
		Analytics:  siteAnalytics,
		Comparison: comparison,
	}, nil
}

// GetAllHistoryRuns returns all recorded scan runs for the domain
func (a *App) GetAllHistoryRuns(domain string) ([]history.ScanRunSummary, error) {
	return history.GetAllHistoryRuns(domain)
}

// CompareHistoryRuns compares two historical scan runs by ID
func (a *App) CompareHistoryRuns(baseRunID, currentRunID string) (*history.RunsDiffResult, error) {
	return history.CompareHistoryRuns(baseRunID, currentRunID)
}

// GetExportHistory returns all recorded export packages for the domain
func (a *App) GetExportHistory(domain string) ([]wpexport.ExportRecord, error) {
	return wpexport.GetExportHistory(domain)
}

// CompareExportPackages compares two export packages by their manifest.json file paths
func (a *App) CompareExportPackages(baseManifestPath, currentManifestPath string) (*wpexport.ExportDiffReport, error) {
	return wpexport.CompareExportPackages(baseManifestPath, currentManifestPath)
}
