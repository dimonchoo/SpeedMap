package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/wpexport"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RevealInFinder opens macOS Finder (or File Explorer on Windows / Linux) with the target file or folder selected
func (a *App) RevealInFinder(targetPath string) error {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return fmt.Errorf("empty path")
	}
	targetPath = filepath.Clean(targetPath)
	fmt.Printf("[GO LOG] RevealInFinder called for: %s\n", targetPath)

	switch goRuntime.GOOS {
	case "darwin":
		fi, err := os.Stat(targetPath)
		if err == nil && fi.IsDir() {
			return exec.Command("open", targetPath).Start()
		}
		return exec.Command("open", "-R", targetPath).Start()
	case "windows":
		fi, err := os.Stat(targetPath)
		if err == nil && fi.IsDir() {
			return exec.Command("explorer", targetPath).Start()
		}
		return exec.Command("explorer", "/select,", targetPath).Start()
	default:
		dir := targetPath
		fi, err := os.Stat(targetPath)
		if err == nil && !fi.IsDir() {
			dir = filepath.Dir(targetPath)
		}
		return exec.Command("xdg-open", dir).Start()
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

// ExportWordPressWebPApplyPHP converts heavy images and writes a deploy package
func (a *App) ExportWordPressWebPApplyPHP(domain string, cfg config.ScanConfig, results []scanner.PageResult, wordpressPath string) (*wpexport.ExportResult, error) {
	return a.ExportWordPressWebPApplyPHPWithOverrides(domain, cfg, results, wordpressPath, nil)
}

// ExportWordPressWebPApplyPHPWithOverrides converts heavy images taking per-image studio overrides into account.
func (a *App) ExportWordPressWebPApplyPHPWithOverrides(domain string, cfg config.ScanConfig, results []scanner.PageResult, wordpressPath string, overrides map[string]wpexport.ImageOverride) (*wpexport.ExportResult, error) {
	outBase := strings.TrimSpace(wordpressPath)
	fmt.Printf("[GO LOG] ExportWordPressWebPApplyPHPWithOverrides called for %s (%d pages, out=%s, overrides=%d)\n", domain, len(results), outBase, len(overrides))

	site := analytics.ComputeSiteAnalytics(results, cfg.HeavyImageThresholdKB)
	heavy := wpexport.CollectHeavyImages(site.AllImages)
	if len(heavy) == 0 {
		return nil, fmt.Errorf("no heavy convertible images in scan")
	}

	written, err := wpexport.ConvertHeavyImagesWithProgressAndOverrides(
		heavy,
		cfg.NormalizedWebPQuality(),
		cfg.NormalizedMinWebPQuality(),
		cfg.IsSkipIfNoWebPSavingsEnabled(),
		cfg.NormalizedHeavyThresholdBytes(),
		cfg.IsAdaptiveQualityEnabled(),
		cfg.IsResizeToRetinaEnabled(),
		cfg.AuthUser,
		cfg.AuthPass,
		overrides,
		func(done, total int, name string) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "export:progress", map[string]interface{}{
					"current":  done,
					"total":    total,
					"filename": name,
					"percent":  int(float64(done) / float64(total) * 100),
				})
			}
		},
	)
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

// ExportDOMVirtualizationCSS opens a native macOS Save Dialog and saves the CSS snippet
func (a *App) ExportDOMVirtualizationCSS(cssContent string, defaultFilename string) (string, error) {
	if defaultFilename == "" {
		defaultFilename = "speedmap-dom-virtualization.css"
	}
	var savePath string
	var err error
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти CSS віртуалізації DOM",
			DefaultFilename: defaultFilename,
			Filters: []runtime.FileFilter{
				{DisplayName: "CSS Files (*.css)", Pattern: "*.css"},
			},
		})
		if err != nil {
			return "", err
		}
		if savePath == "" {
			return "", nil
		}
	} else {
		savePath = filepath.Join(os.TempDir(), defaultFilename)
	}

	err = os.WriteFile(savePath, []byte(cssContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save CSS: %w", err)
	}
	return savePath, nil
}

// ExportDOMVirtualizationPHP opens a native macOS Save Dialog and saves the PHP hook snippet
func (a *App) ExportDOMVirtualizationPHP(phpContent string, defaultFilename string) (string, error) {
	if defaultFilename == "" {
		defaultFilename = "speedmap-dom-virtualization.php"
	}
	var savePath string
	var err error
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти PHP хук (wp_head) для functions.php",
			DefaultFilename: defaultFilename,
			Filters: []runtime.FileFilter{
				{DisplayName: "PHP Files (*.php)", Pattern: "*.php"},
			},
		})
		if err != nil {
			return "", err
		}
		if savePath == "" {
			return "", nil
		}
	} else {
		savePath = filepath.Join(os.TempDir(), defaultFilename)
	}

	err = os.WriteFile(savePath, []byte(phpContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save PHP: %w", err)
	}
	return savePath, nil
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

// LoadExportPackageForStudio loads an existing export package for fine-tuning in Image Studio
func (a *App) LoadExportPackageForStudio(dirOrManifest string) (*wpexport.PackageStudioContext, error) {
	fmt.Printf("[GO LOG] LoadExportPackageForStudio called for: %s\n", dirOrManifest)
	return wpexport.LoadPackageForStudio(dirOrManifest)
}

// SaveTunedImageToPackage fine-tunes and overwrites an image inside an export package, updating manifest.json & compare.html
func (a *App) SaveTunedImageToPackage(packageDir string, imageID string, rawURL string, opts optimizer.ImageTuneOptions, cfg config.ScanConfig) (*wpexport.TunedSaveResult, error) {
	fmt.Printf("[GO LOG] SaveTunedImageToPackage called for image #%s (%s) in package %s\n", imageID, rawURL, packageDir)
	return wpexport.SaveTunedImageToPackage(packageDir, imageID, rawURL, opts, cfg)
}

// OpenPackageCompareHTML opens compare.html from the export package in default browser, optionally jumping to a specific item
func (a *App) OpenPackageCompareHTML(dirOrManifest string, targetAnchor string) error {
	fmt.Printf("[GO LOG] OpenPackageCompareHTML called for: %s (anchor: %s)\n", dirOrManifest, targetAnchor)
	return wpexport.OpenPackageCompareHTML(dirOrManifest, targetAnchor)
}

// RegeneratePackageCompareHTML regenerates compare.html from manifest.json in the package folder
func (a *App) RegeneratePackageCompareHTML(dirOrManifest string) error {
	fmt.Printf("[GO LOG] RegeneratePackageCompareHTML called for: %s\n", dirOrManifest)
	return wpexport.RegeneratePackageCompareHTML(dirOrManifest)
}

// SelectImageFile opens native file picker dialog to pick an image file
func (a *App) SelectImageFile(title string) (string, error) {
	if title == "" {
		title = "Виберіть зображення для заміни (WebP, PNG, JPG, SVG)"
	}
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Зображення (*.webp, *.png, *.jpg, *.jpeg, *.svg)",
				Pattern:     "*.webp;*.png;*.jpg;*.jpeg;*.svg",
			},
		},
	})
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// ReplacePackageImageWithCustomFile replaces an optimized image in package with a custom local file
func (a *App) ReplacePackageImageWithCustomFile(packageDir, imageID, sourceFilePath, rawURL string) (*wpexport.TunedSaveResult, error) {
	fmt.Printf("[GO LOG] ReplacePackageImageWithCustomFile image #%s with %s in %s\n", imageID, sourceFilePath, packageDir)
	res, err := wpexport.ReplacePackageImageWithFile(packageDir, imageID, sourceFilePath)
	if err == nil && rawURL != "" {
		a.ClearPreviewCache(rawURL)
	}
	return res, err
}

// ReloadPackageImageFromDisk refreshes manifest & stats if file in package was changed directly in Finder
func (a *App) ReloadPackageImageFromDisk(packageDir, imageID, rawURL string) (*wpexport.TunedSaveResult, error) {
	fmt.Printf("[GO LOG] ReloadPackageImageFromDisk image #%s in %s\n", imageID, packageDir)
	res, err := wpexport.ReloadPackageImageFromDisk(packageDir, imageID)
	if err == nil && rawURL != "" {
		a.ClearPreviewCache(rawURL)
	}
	return res, err
}

// RevertPackageImageToRemote restores original from site URL
func (a *App) RevertPackageImageToRemote(packageDir, imageID, rawURL string, cfg config.ScanConfig) (*wpexport.TunedSaveResult, error) {
	fmt.Printf("[GO LOG] RevertPackageImageToRemote image #%s in %s\n", imageID, packageDir)
	res, err := wpexport.RevertPackageImageToRemote(packageDir, imageID, cfg)
	if err == nil && rawURL != "" {
		a.ClearPreviewCache(rawURL)
	}
	return res, err
}

// GetPackageImagePreview returns ConversionResult based on local package file
func (a *App) GetPackageImagePreview(packageDir, imageID string) (*optimizer.ConversionResult, error) {
	return wpexport.GetPackageImagePreview(packageDir, imageID)
}

