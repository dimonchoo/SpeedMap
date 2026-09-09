package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	previewBytesCache  = make(map[string][]byte)
	previewResultCache = make(map[string]*optimizer.ConversionResult)
	previewCacheMutex  sync.RWMutex
)

// ClearPreviewCache invalidates RAM caches for a specific image URL, or completely if rawURL is empty
func (a *App) ClearPreviewCache(rawURL string) {
	previewCacheMutex.Lock()
	defer previewCacheMutex.Unlock()
	if rawURL == "" {
		previewBytesCache = make(map[string][]byte)
		previewResultCache = make(map[string]*optimizer.ConversionResult)
		fmt.Println("[GO LOG] ClearPreviewCache: all caches cleared")
		return
	}
	delete(previewBytesCache, rawURL)
	for k := range previewResultCache {
		if strings.HasPrefix(k, rawURL) {
			delete(previewResultCache, k)
		}
	}
	fmt.Printf("[GO LOG] ClearPreviewCache: invalidated cache for %s\n", rawURL)
}

// TuneImagePreview converts an image with exact tuned parameters for Image Studio live preview.
// Original bytes are cached in memory so subsequent slider movements respond in 10-30ms.
func (a *App) TuneImagePreview(rawURL string, opts optimizer.ImageTuneOptions, cfg config.ScanConfig) (*optimizer.ConversionResult, error) {
	start := time.Now()
	cacheKey := fmt.Sprintf("%s|q:%.1f|l:%v|e:%v|w:%d|h:%d|d:%v", rawURL, opts.Quality, opts.Lossless, opts.Exact, opts.MaxW, opts.MaxH, opts.Dither)

	previewCacheMutex.RLock()
	// Check exact options cache hit first
	if cachedRes, ok := previewResultCache[cacheKey]; ok && cachedRes != nil {
		previewCacheMutex.RUnlock()
		fmt.Printf("[GO LOG] TuneImagePreview (CACHE HIT) for %s in 0ms\n", rawURL)
		return cachedRes, nil
	}
	// Check if this URL is already known as a pure vector SVG (where raster options do not apply)
	if cachedRes, ok := previewResultCache[rawURL]; ok && cachedRes != nil {
		previewCacheMutex.RUnlock()
		fmt.Printf("[GO LOG] TuneImagePreview (VECTOR SVG CACHE HIT) for %s in 0ms\n", rawURL)
		return cachedRes, nil
	}
	cachedBytes, found := previewBytesCache[rawURL]
	previewCacheMutex.RUnlock()

	if !found || len(cachedBytes) == 0 {
		var err error
		cachedBytes, err = optimizer.FetchImageBytes(rawURL, cfg.AuthUser, cfg.AuthPass, cfg.GetUserAgent())
		if err != nil {
			fmt.Printf("[GO LOG] TuneImagePreview fetch error for %s: %v\n", rawURL, err)
			return nil, fmt.Errorf("failed to fetch image: %w", err)
		}
		previewCacheMutex.Lock()
		previewBytesCache[rawURL] = cachedBytes
		previewCacheMutex.Unlock()
	}

	res, err := optimizer.ConvertImageBytesTuned(rawURL, cachedBytes, opts)
	if err != nil {
		fmt.Printf("[GO LOG] TuneImagePreview convert error for %s: %v\n", rawURL, err)
		return nil, err
	}

	if res != nil {
		previewCacheMutex.Lock()
		previewResultCache[cacheKey] = res
		// If it is pure vector SVG (output remains data:image/svg+xml, not WebP), cache by URL so options don't re-run
		if strings.HasPrefix(res.OptimizedWebPBase64, "data:image/svg+xml") {
			previewResultCache[rawURL] = res
		}
		previewCacheMutex.Unlock()
	}

	fmt.Printf("[GO LOG] TuneImagePreview completed for %s in %v\n", rawURL, time.Since(start))
	return res, nil
}

// DownloadSingleWebPTuned converts and saves a single image using exact tuned options from Image Studio
func (a *App) DownloadSingleWebPTuned(rawURL string, opts optimizer.ImageTuneOptions, cfg config.ScanConfig) (string, error) {
	res, err := a.TuneImagePreview(rawURL, opts, cfg)
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

	saveTitle := "Зберегти налаштоване WebP зображення"
	filters := []runtime.FileFilter{
		{DisplayName: "WebP Зображення (*.webp)", Pattern: "*.webp"},
	}
	if strings.HasPrefix(res.OptimizedWebPBase64, "data:image/svg+xml") {
		saveTitle = "Зберегти оптимізоване SVG зображення"
		filters = []runtime.FileFilter{
			{DisplayName: "SVG Векторне Зображення (*.svg)", Pattern: "*.svg"},
		}
	}

	var savePath string
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           saveTitle,
			DefaultFilename: res.Filename,
			Filters:         filters,
		})
	}
	if savePath == "" || err != nil {
		savePath = res.Filename
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return savePath, nil
}

// ConvertImageToWebP encodes a single image to WebP with the configured quality setting
func (a *App) ConvertImageToWebP(rawURL string, cfg config.ScanConfig) (*optimizer.ConversionResult, error) {
	optimizer.SetDefaultUserAgent(cfg.GetUserAgent())
	fmt.Printf("[GO LOG] ConvertImageToWebP called for %s (quality=%.0f, threshold=%d KB, adaptive=%v)\n", rawURL, cfg.NormalizedWebPQuality(), cfg.HeavyImageThresholdKB, cfg.IsAdaptiveQualityEnabled())
	res, err := optimizer.ConvertImageURLToWebPAdaptiveBudgetAuth(rawURL, cfg.NormalizedWebPQuality(), cfg.NormalizedHeavyThresholdBytes(), cfg.IsAdaptiveQualityEnabled(), cfg.AuthUser, cfg.AuthPass)
	if err != nil {
		fmt.Printf("[GO LOG] ConvertImageToWebP error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] ConvertImageToWebP success: %s -> %s (savings %.1f%%, qualityUsed=%.0f, lossless=%v)\n", res.OriginalFormatted, res.OptimizedFormatted, res.SavingsPercent, res.QualityUsed, res.IsLossless)
	return res, nil
}

// DownloadSingleWebPImage converts a single image to WebP (or saves clean SVG) and prompts via SaveFileDialog
func (a *App) DownloadSingleWebPImage(rawURL string, cfg config.ScanConfig) (string, error) {
	optimizer.SetDefaultUserAgent(cfg.GetUserAgent())
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

	saveTitle := "Зберегти WebP зображення"
	filters := []runtime.FileFilter{
		{DisplayName: "WebP Зображення (*.webp)", Pattern: "*.webp"},
	}
	if strings.HasPrefix(res.OptimizedWebPBase64, "data:image/svg+xml") {
		saveTitle = "Зберегти оптимізоване SVG зображення"
		filters = []runtime.FileFilter{
			{DisplayName: "SVG Векторне Зображення (*.svg)", Pattern: "*.svg"},
		}
	}

	var savePath string
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           saveTitle,
			DefaultFilename: res.Filename,
			Filters:         filters,
		})
	}
	if savePath == "" || err != nil {
		savePath = res.Filename
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("[GO LOG] DownloadSingleWebPImage saved to %s\n", savePath)
	return savePath, nil
}

// DownloadOriginalImage downloads the raw original image (e.g. SVG, PNG, JPG) and prompts the user to save it
func (a *App) DownloadOriginalImage(rawURL string, cfg config.ScanConfig) (string, error) {
	optimizer.SetDefaultUserAgent(cfg.GetUserAgent())
	fmt.Printf("[GO LOG] DownloadOriginalImage called for %s\n", rawURL)
	data, err := optimizer.FetchImageBytes(rawURL, cfg.AuthUser, cfg.AuthPass, cfg.GetUserAgent())
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}

	origName := optimizer.ExtractOriginalFilename(rawURL)
	ext := filepath.Ext(origName)
	pattern := "*" + ext
	if ext == "" {
		pattern = "*.*"
	}

	var savePath string
	if a.ctx != nil {
		savePath, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Зберегти оригінальне зображення",
			DefaultFilename: origName,
			Filters: []runtime.FileFilter{
				{DisplayName: fmt.Sprintf("Зображення (%s)", pattern), Pattern: pattern},
				{DisplayName: "Всі файли (*.*)", Pattern: "*.*"},
			},
		})
	}
	if savePath == "" || err != nil {
		savePath = origName
	}

	if err := os.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("[GO LOG] Original image saved to %s\n", savePath)
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
	optimizer.SetDefaultUserAgent(cfg.GetUserAgent())
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
