package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/cloud"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/history"
	"SpeedMap/pkg/optimizer"
	"SpeedMap/pkg/profiles"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/sitemap"
	"SpeedMap/pkg/w3c"
	"SpeedMap/pkg/wpexport"

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

	gdriveManager *cloud.GDriveManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		gdriveManager: cloud.NewGDriveManager(),
	}
}


// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("[GO LOG] SpeedMap backend started successfully.")

	// Auto-prune old scan runs in background
	go func() {
		_ = history.AutoPruneHistory(20, 30)
	}()
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

	currentRun, err := history.SaveScanRunWithConfig(domain, results, siteAnalytics, cfg)
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

// ConvertImageToWebP encodes a single image to WebP with the configured quality setting
func (a *App) ConvertImageToWebP(rawURL string, cfg config.ScanConfig) (*optimizer.ConversionResult, error) {
	fmt.Printf("[GO LOG] ConvertImageToWebP called for %s (quality=%.0f, threshold=%d KB, adaptive=%v)\n", rawURL, cfg.NormalizedWebPQuality(), cfg.HeavyImageThresholdKB, cfg.IsAdaptiveQualityEnabled())
	res, err := optimizer.ConvertImageURLToWebPAdaptiveBudgetAuth(rawURL, cfg.NormalizedWebPQuality(), cfg.NormalizedHeavyThresholdBytes(), cfg.IsAdaptiveQualityEnabled(), cfg.AuthUser, cfg.AuthPass)
	if err != nil {
		fmt.Printf("[GO LOG] ConvertImageToWebP error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] ConvertImageToWebP success: %s -> %s (savings %.1f%%, qualityUsed=%.0f, lossless=%v)\n", res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	return res, nil
}

// DownloadSingleWebPImage converts a single image to WebP and saves it via SaveFileDialog
func (a *App) DownloadSingleWebPImage(rawURL string, cfg config.ScanConfig) (string, error) {
	fmt.Printf("[GO LOG] DownloadSingleWebPImage called for %s\n", rawURL)
	res, err := optimizer.ConvertImageURLToWebPAdaptiveBudgetAuth(rawURL, cfg.NormalizedWebPQuality(), cfg.NormalizedHeavyThresholdBytes(), cfg.IsAdaptiveQualityEnabled(), cfg.AuthUser, cfg.AuthPass)
	if err != nil {
		return "", err
	}

	idx := strings.Index(res.OptimizedWebPBase64, ",")
	if idx == -1 {
		return "", fmt.Errorf("invalid base64 image data")
	}
	data, err := base64.StdEncoding.DecodeString(res.OptimizedWebPBase64[idx+1:])
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	var savePath string
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти WebP зображення",
			DefaultFilename: res.Filename,
			Filters: []runtime.FileFilter{
				{DisplayName: "WebP Зображення (*.webp)", Pattern: "*.webp"},
			},
		})
	}
	if savePath == "" || err != nil {
		savePath = res.Filename
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("[GO LOG] WebP image saved to %s\n", savePath)
	return savePath, nil
}

// DownloadOptimizedWebPZIP batch converts multiple heavy images to WebP and saves a single .zip archive
func (a *App) DownloadOptimizedWebPZIP(urls []string, cfg config.ScanConfig) (string, error) {
	fmt.Printf("[GO LOG] DownloadOptimizedWebPZIP called for %d images\n", len(urls))
	if len(urls) == 0 {
		return "", fmt.Errorf("no image URLs provided")
	}

	type convertRes struct {
		res *optimizer.ConversionResult
		err error
	}

	ch := make(chan convertRes, len(urls))
	var wg sync.WaitGroup

	quality := cfg.NormalizedWebPQuality()
	threshold := cfg.NormalizedHeavyThresholdBytes()
	adaptive := cfg.IsAdaptiveQualityEnabled()
	authUser, authPass := cfg.AuthUser, cfg.AuthPass
	for _, rawURL := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			res, err := optimizer.ConvertImageURLToWebPAdaptiveBudgetAuth(u, quality, threshold, adaptive, authUser, authPass)
			ch <- convertRes{res: res, err: err}
		}(rawURL)
	}

	wg.Wait()
	close(ch)

	var convertedList []*optimizer.ConversionResult
	for item := range ch {
		if item.err == nil && item.res != nil {
			convertedList = append(convertedList, item.res)
		}
	}

	if len(convertedList) == 0 {
		return "", fmt.Errorf("failed to convert any images to WebP")
	}

	zipBytes, err := optimizer.CreateZIPArchive(convertedList)
	if err != nil {
		return "", fmt.Errorf("failed to build ZIP archive: %w", err)
	}

	var savePath string
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти ZIP архів WebP зображень",
			DefaultFilename: "optimized_images_webp.zip",
			Filters: []runtime.FileFilter{
				{DisplayName: "ZIP Архіви (*.zip)", Pattern: "*.zip"},
			},
		})
	}
	if savePath == "" || err != nil {
		savePath = "optimized_images_webp.zip"
	}

	if err := os.WriteFile(savePath, zipBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write ZIP file: %w", err)
	}

	fmt.Printf("[GO LOG] ZIP archive saved to %s (%d files, %d bytes)\n", savePath, len(convertedList), len(zipBytes))
	return savePath, nil
}

// ExportWordPressWebPApplyPHP converts heavy images and writes a deploy package
// (images/ + apply.php + rollback.php + compare.html). PHP apply copies webp into
// uploads and retargets attachments — SpeedMap does not write into WP uploads.
func (a *App) ExportWordPressWebPApplyPHP(domain string, cfg config.ScanConfig, results []scanner.PageResult, wordpressPath string) (*wpexport.ExportResult, error) {
	outBase := strings.TrimSpace(wordpressPath)
	fmt.Printf("[GO LOG] ExportWordPressWebPApplyPHP called for %s (%d pages, out=%s)\n", domain, len(results), outBase)

	site := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)
	heavy := wpexport.CollectHeavyImages(site.AllImages)
	if len(heavy) == 0 {
		return nil, fmt.Errorf("no heavy convertible images in scan")
	}

	written, err := wpexport.ConvertHeavyImagesWithThreshold(heavy, cfg.NormalizedWebPQuality(), cfg.NormalizedHeavyThresholdBytes(), cfg.IsAdaptiveQualityEnabled(), cfg.IsResizeToRetinaEnabled(), cfg.AuthUser, cfg.AuthPass)
	if err != nil {
		return nil, err
	}

	stamp := time.Now().Format("20060102-150405")
	pkgDir := outBase
	if info, statErr := os.Stat(outBase); statErr == nil && info.IsDir() {
		pkgDir = filepath.Join(outBase, fmt.Sprintf("speedmap-webp-%s", stamp))
	} else if outBase == "" {
		return nil, fmt.Errorf("output path is required (folder where the package will be written)")
	} else {
		// Treat as parent path to create
		if err := os.MkdirAll(outBase, 0755); err != nil {
			return nil, err
		}
		pkgDir = filepath.Join(outBase, fmt.Sprintf("speedmap-webp-%s", stamp))
	}

	out, err := wpexport.WriteDeployPackage(pkgDir, domain, cfg, written)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[GO LOG] WP WebP package: %d WebP → %s (apply=%s zip=%s)\n", out.WebPCount, out.PackageDir, out.ApplyPHP, out.ReviewZIP)
	return out, nil
}

// ListSiteProfiles fetches all persistent site profiles from ~/.speedmap/profiles.json
func (a *App) ListSiteProfiles() ([]profiles.SiteProfile, error) {
	fmt.Println("[GO LOG] ListSiteProfiles called")
	list, err := profiles.ListProfiles()
	if err != nil {
		fmt.Printf("[GO LOG] ListSiteProfiles error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] ListSiteProfiles returned %d profiles\n", len(list))
	return list, nil
}

// SaveSiteProfile creates or updates a persistent site profile in ~/.speedmap/profiles.json
func (a *App) SaveSiteProfile(p profiles.SiteProfile) (*profiles.SiteProfile, error) {
	fmt.Printf("[GO LOG] SaveSiteProfile called for '%s' (%s)\n", p.Name, p.SitemapURL)
	saved, err := profiles.SaveProfile(p)
	if err != nil {
		fmt.Printf("[GO LOG] SaveSiteProfile error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] SaveSiteProfile success: ID=%s\n", saved.ID)
	return saved, nil
}

// DeleteSiteProfile removes a site profile by ID
func (a *App) DeleteSiteProfile(id string) error {
	fmt.Printf("[GO LOG] DeleteSiteProfile called for ID=%s\n", id)
	err := profiles.DeleteProfile(id)
	if err != nil {
		fmt.Printf("[GO LOG] DeleteSiteProfile error: %v\n", err)
		return err
	}
	fmt.Printf("[GO LOG] DeleteSiteProfile success for ID=%s\n", id)
	return nil
}

// PlayNotificationSound plays embedded Pikachu MP3 via OS player.
// WebKitGTK often cannot decode MP3 in <audio>, so we play through Pulse/CoreAudio
// (picked up by SonoBus monitor on Linux).
// kind: "page" | "full"
func (a *App) PlayNotificationSound(kind string) error {
	file := "frontend/pika-page.mp3"
	if kind == "full" {
		file = "frontend/pika-full.mp3"
	}

	data, err := assets.ReadFile(file)
	if err != nil {
		fmt.Printf("[GO LOG] PlayNotificationSound read %s: %v\n", file, err)
		return fmt.Errorf("sound asset missing: %w", err)
	}

	tmp, err := os.CreateTemp("", "speedmap-*-"+filepath.Base(file))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	var cmd *exec.Cmd
	if _, err := exec.LookPath("afplay"); err == nil {
		cmd = exec.Command("afplay", tmpPath)
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", tmpPath)
	} else if _, err := exec.LookPath("mpg123"); err == nil {
		cmd = exec.Command("mpg123", "-q", tmpPath)
	} else {
		os.Remove(tmpPath)
		return fmt.Errorf("no audio player found (afplay/ffplay/mpg123)")
	}

	fmt.Printf("[GO LOG] PlayNotificationSound kind=%s via %s\n", kind, cmd.Path)
	go func() {
		defer os.Remove(tmpPath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[GO LOG] PlayNotificationSound error: %v\n", err)
		}
	}()
	return nil
}

// ExportFontsCSV opens a native macOS Save Dialog and saves the CSV report
func (a *App) ExportFontsCSV(csvContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по шрифтах (CSV)",
		DefaultFilename: fmt.Sprintf("speedmap_fonts_report_%d.csv", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(csvContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportFontsJSON opens a native macOS Save Dialog and saves the JSON report
func (a *App) ExportFontsJSON(jsonContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по шрифтах (JSON)",
		DefaultFilename: fmt.Sprintf("speedmap_fonts_report_%d.json", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(jsonContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportIframesCSV opens a native macOS Save Dialog and saves the CSV report
func (a *App) ExportIframesCSV(csvContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по iframe (CSV)",
		DefaultFilename: fmt.Sprintf("speedmap_iframes_report_%d.csv", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(csvContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportIframesJSON opens a native macOS Save Dialog and saves the JSON report
func (a *App) ExportIframesJSON(jsonContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по iframe (JSON)",
		DefaultFilename: fmt.Sprintf("speedmap_iframes_report_%d.json", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(jsonContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportFormsCSV opens a native macOS Save Dialog and saves the Forms CSV report
func (a *App) ExportFormsCSV(csvContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по формах сайту (CSV)",
		DefaultFilename: fmt.Sprintf("speedmap_forms_report_%d.csv", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(csvContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// ExportFormsJSON opens a native macOS Save Dialog and saves the Forms JSON report
func (a *App) ExportFormsJSON(jsonContent string) (string, error) {
	filename, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Зберегти звіт по формах сайту (JSON)",
		DefaultFilename: fmt.Sprintf("speedmap_forms_report_%d.json", time.Now().Unix()),
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil || filename == "" {
		return "", err
	}
	err = os.WriteFile(filename, []byte(jsonContent), 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// SelectDirectory opens native macOS directory picker dialog
func (a *App) SelectDirectory(title string) (string, error) {

	if title == "" {
		title = "Виберіть папку WordPress"
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

// StartGDriveAuth starts Google OAuth2 browser flow
func (a *App) StartGDriveAuth(clientID, clientSecret string) (string, error) {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.StartAuthFlow(clientID, clientSecret)
}


// GetGDriveStatus returns connection status and user email
func (a *App) GetGDriveStatus() map[string]interface{} {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return map[string]interface{}{
		"connected": a.gdriveManager.IsConnected(),
		"email":     a.gdriveManager.GetUserEmail(),
	}
}

// DisconnectGDrive removes saved OAuth token
func (a *App) DisconnectGDrive() error {
	if a.gdriveManager == nil {
		return nil
	}
	return a.gdriveManager.Disconnect()
}

// UploadFileToGDrive uploads a given local file to Google Drive and returns public link
func (a *App) UploadFileToGDrive(filePath string, folderName string) (*cloud.DriveUploadResult, error) {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.UploadFile(filePath, folderName)
}

// SaveGDriveCredentials saves Client ID & Secret
func (a *App) SaveGDriveCredentials(clientID, clientSecret string) error {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.SaveCredentials(clientID, clientSecret)
}

// GetGDriveCredentials gets saved Client ID & Secret
func (a *App) GetGDriveCredentials() map[string]string {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.GetCredentials()
}






