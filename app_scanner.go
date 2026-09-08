package main

import (
	"context"
	"fmt"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/sitemap"
	"SpeedMap/pkg/w3c"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ResolveDomain resolves the domain's IP addresses and detects Cloudflare CDN proxying
func (a *App) ResolveDomain(targetURL string) (scanner.DomainResolution, error) {
	fmt.Printf("[GO LOG] ResolveDomain called: %s\n", targetURL)
	return scanner.ResolveDomain(targetURL)
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
	defer sc.Cancel()
	res := sc.ScanSingleURL(id, url)
	fmt.Printf("[GO LOG] RescanSingleURL completed: ID=%d, Status=%d, Error='%s'\n", id, res.StatusCode, res.Error)
	return res, nil
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

// ValidateW3C validates a page's live rendered HTML using the official W3C Nu HTML Checker
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
