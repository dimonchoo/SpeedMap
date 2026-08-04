package main

import (
	"context"
	"fmt"
	"sync"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/sitemap"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	activeScanner *scanner.Scanner
	scannerMu     sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown is called when the application closes to ensure all Chrome processes are killed
func (a *App) shutdown(ctx context.Context) {
	a.CancelScan()
}

// ParseSitemap fetches and parses the given sitemap URL
func (a *App) ParseSitemap(sitemapUrl string, cfg config.ScanConfig) ([]string, error) {
	if sitemapUrl == "" {
		sitemapUrl = cfg.SitemapURL
	}
	urls, err := sitemap.FetchAndParse(sitemapUrl, cfg)
	if err != nil {
		return nil, fmt.Errorf("Sitemap error: %w", err)
	}
	return urls, nil
}

// StartScan starts scanning selected URLs asynchronously and emits progress events to frontend
func (a *App) StartScan(cfg config.ScanConfig, urls []string) error {
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
	sc := scanner.NewScanner(cfg)
	defer sc.Cancel() // Ensures Chrome process is killed immediately after single URL scan
	res := sc.ScanSingleURL(id, url)
	return res, nil
}

// OpenURL opens the target URL in the user's default web browser
func (a *App) OpenURL(rawURL string) {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, rawURL)
	}
}

// CancelScan cancels any running scan and kills all Chrome processes
func (a *App) CancelScan() {
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
