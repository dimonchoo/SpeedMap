package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/history"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/sitemap"
	"SpeedMap/pkg/w3c"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AnalyticsResult struct {
	Analytics  analytics.SiteAnalytics `json:"analytics"`
	Comparison history.RunComparison    `json:"comparison"`
}

// App struct
type App struct {
	ctx           context.Context
	activeScanner *scanner.Scanner
	scannerMu     sync.Mutex

	reportServerMu    sync.Mutex
	reportServerPort  int
	currentReportHTML string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("[GO LOG] SpeedMap backend started successfully.")
}

// shutdown is called when the application closes to ensure all Chrome processes are killed
func (a *App) shutdown(ctx context.Context) {
	fmt.Println("[GO LOG] Application shutting down. Canceling active scans...")
	a.CancelScan()
}

// ParseSitemap fetches and parses the given sitemap URL
func (a *App) ParseSitemap(sitemapUrl string, cfg config.ScanConfig) ([]string, error) {
	fmt.Printf("[GO LOG] ParseSitemap called: %s\n", sitemapUrl)
	if sitemapUrl == "" {
		sitemapUrl = cfg.SitemapURL
	}
	urls, err := sitemap.FetchAndParse(sitemapUrl, cfg)
	if err != nil {
		fmt.Printf("[GO LOG] ParseSitemap error: %v\n", err)
		return nil, fmt.Errorf("Sitemap error: %w", err)
	}
	fmt.Printf("[GO LOG] ParseSitemap success: found %d URLs\n", len(urls))
	return urls, nil
}

// StartScan starts scanning selected URLs asynchronously and emits progress events to frontend
func (a *App) StartScan(cfg config.ScanConfig, urls []string) error {
	fmt.Printf("[GO LOG] StartScan called: %d URLs, concurrency=%d\n", len(urls), cfg.NormalizedConcurrency())
	a.scannerMu.Lock()
	if a.activeScanner != nil && !a.activeScanner.IsCanceled() {
		a.scannerMu.Unlock()
		return fmt.Errorf("a scan is already running")
	}

	sc := scanner.NewScanner(cfg)
	a.activeScanner = sc
	a.scannerMu.Unlock()

	go func() {
		defer func() {
			a.scannerMu.Lock()
			if a.activeScanner == sc {
				a.activeScanner = nil
			}
			a.scannerMu.Unlock()
		}()

		_, _ = sc.ScanURLs(urls, func(progress scanner.ScanProgress) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "scan:progress", progress)
			}
		})
	}()

	return nil
}

// RescanSingleURL rescans a single specific URL on demand without re-scanning the entire sitemap
func (a *App) RescanSingleURL(cfg config.ScanConfig, url string, id int) (scanner.PageResult, error) {
	fmt.Printf("[GO LOG] RescanSingleURL called: ID=%d, URL=%s\n", id, url)
	sc := scanner.NewScanner(cfg)
	defer sc.Cancel() // Ensures Chrome process is killed immediately after single URL scan
	res := sc.ScanSingleURL(id, url)
	fmt.Printf("[GO LOG] RescanSingleURL completed: ID=%d, Status=%d, Error='%s'\n", id, res.StatusCode, res.Error)
	return res, nil
}

// ComputeSiteAnalytics computes analytics and compares with previous scan run
func (a *App) ComputeSiteAnalytics(domain string, cfg config.ScanConfig, results []scanner.PageResult) (AnalyticsResult, error) {
	fmt.Printf("[GO LOG] ComputeSiteAnalytics called for %s (%d pages, threshold=%d KB)\n", domain, len(results), cfg.HeavyImageThresholdKB)
	siteAnalytics := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)

	// Fetch previous run for comparison BEFORE saving current run
	prevRun, _ := history.GetPreviousRun(domain, "")

	currentRun, err := history.SaveScanRun(domain, results, siteAnalytics)
	if err != nil {
		return AnalyticsResult{
			Analytics:  siteAnalytics,
			Comparison: history.CompareRuns(&history.ScanRun{HealthScore: siteAnalytics.HealthScore, Analytics: siteAnalytics}, prevRun),
		}, nil
	}

	cmp := history.CompareRuns(currentRun, prevRun)
	return AnalyticsResult{
		Analytics:  siteAnalytics,
		Comparison: cmp,
	}, nil
}

// ValidateW3C performs official W3C HTML5 validation via W3C Nu API
func (a *App) ValidateW3C(rawURL string) (*w3c.W3CReport, error) {
	fmt.Printf("[GO LOG] ValidateW3C called for %s\n", rawURL)
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	report, err := w3c.ValidateURL(ctx, rawURL)
	if err != nil {
		fmt.Printf("[GO LOG] ValidateW3C error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] ValidateW3C completed for %s: status=%s, errors=%d, warnings=%d\n", rawURL, report.Status, report.ErrorCount, report.WarningCount)
	return report, nil
}

// OpenURL opens the target URL in the user's default web browser
func (a *App) OpenURL(rawURL string) {
	fmt.Printf("[GO LOG] OpenURL called for %s\n", rawURL)
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, rawURL)
	}
}

// CancelScan cancels any running scan and kills all Chrome processes
func (a *App) CancelScan() {
	fmt.Println("[GO LOG] CancelScan called")
	a.scannerMu.Lock()
	defer a.scannerMu.Unlock()

	if a.activeScanner != nil {
		a.activeScanner.Cancel()
		a.activeScanner = nil
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "scan:canceled", true)
		}
	}
}

// PreviewImageComparisonHTML spins up an in-process local HTTP server on a random free port (127.0.0.1:0)
// and opens the report directly in the user's default browser without forcing a file download.
func (a *App) PreviewImageComparisonHTML(domain string, cfg config.ScanConfig, results []scanner.PageResult) (string, error) {
	fmt.Printf("[GO LOG] PreviewImageComparisonHTML called for %s (%d pages)\n", domain, len(results))
	siteAnalytics := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)
	htmlContent := analytics.GenerateImageComparisonHTML(siteAnalytics, domain)

	a.reportServerMu.Lock()
	a.currentReportHTML = htmlContent

	if a.reportServerPort == 0 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			a.reportServerMu.Unlock()
			return "", fmt.Errorf("failed to start local web server: %w", err)
		}
		a.reportServerPort = listener.Addr().(*net.TCPAddr).Port

		mux := http.NewServeMux()
		mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
			a.reportServerMu.Lock()
			content := a.currentReportHTML
			a.reportServerMu.Unlock()

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(content))
		})

		server := &http.Server{Handler: mux}
		go func() {
			_ = server.Serve(listener)
		}()
	}
	port := a.reportServerPort
	a.reportServerMu.Unlock()

	reportURL := fmt.Sprintf("http://127.0.0.1:%d/report", port)
	fmt.Printf("[GO LOG] Ephemeral report web server running at %s\n", reportURL)

	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, reportURL)
	}

	return reportURL, nil
}

// ExportImageComparisonHTML generates and saves an HTML image comparison report for designers & stakeholders
func (a *App) ExportImageComparisonHTML(domain string, cfg config.ScanConfig, results []scanner.PageResult) (string, error) {
	fmt.Printf("[GO LOG] ExportImageComparisonHTML called for %s (%d pages)\n", domain, len(results))
	siteAnalytics := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)
	htmlContent := analytics.GenerateImageComparisonHTML(siteAnalytics, domain)

	var savePath string
	var err error
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти звіт порівняння зображень (SEOAEO-235)",
			DefaultFilename: "image_comparison_report.html",
			Filters: []runtime.FileFilter{
				{DisplayName: "HTML Файли (*.html)", Pattern: "*.html"},
			},
		})
	}

	if savePath == "" || err != nil {
		savePath = "image_comparison_report.html"
	}

	err = os.WriteFile(savePath, []byte(htmlContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save report: %w", err)
	}

	fmt.Printf("[GO LOG] Image comparison HTML saved to %s\n", savePath)
	return savePath, nil
}

