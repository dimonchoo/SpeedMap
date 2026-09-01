package wpexport

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
)

//go:embed apply_template.php
var applyTemplate string

//go:embed rollback_template.php
var rollbackTemplate string

type ManifestImage struct {
	SourceURL              string   `json:"sourceUrl"`
	PathHint               string   `json:"pathHint"`
	WebpRel                string   `json:"webpRel"`
	Basename               string   `json:"basename"`
	Format                 string   `json:"format"`
	IsHeavy                bool     `json:"isHeavy"`
	Bytes                  int64    `json:"bytes"`
	Pages                  []string `json:"pages"`
	ID                     string   `json:"id,omitempty"`
	PackageWebP            string   `json:"packageWebp,omitempty"` // relative to apply.php, e.g. images/001/optimized.webp
	NaturalWidth           int      `json:"naturalWidth,omitempty"`
	NaturalHeight          int      `json:"naturalHeight,omitempty"`
	MaxRenderedWidth        int      `json:"maxRenderedWidth,omitempty"`
	MaxRenderedHeight       int      `json:"maxRenderedHeight,omitempty"`
	RecommendedRetinaWidth  int      `json:"recommendedRetinaWidth,omitempty"`
	RecommendedRetinaHeight int      `json:"recommendedRetinaHeight,omitempty"`
}

type Manifest struct {
	Domain        string          `json:"domain"`
	Generated     string          `json:"generated"`
	Quality       int             `json:"quality"`
	WordPressPath string          `json:"wordpressPath"`
	Images        []ManifestImage `json:"images"`
}

// WrittenImage is a converted heavy image with payloads for package + review ZIP.
type WrittenImage struct {
	ManifestImage
	OrigExt            string
	OrigData           []byte
	WebPData           []byte
	OptimizedWidth     int
	OptimizedHeight    int
	OriginalBytes      int64
	OptimizedBytes     int64
	SavingsPercent     float64
	OriginalFormatted  string
	OptimizedFormatted string
}

// ExportResult paths for the deploy package (not WP uploads).
type ExportResult struct {
	ApplyPHP      string `json:"applyPHP"`
	RollbackPHP   string `json:"rollbackPHP"`
	ReviewZIP     string `json:"reviewZIP"`
	PackageDir    string `json:"packageDir"`
	WebPCount     int    `json:"webpCount"`
	WordPressPath string `json:"wordpressPath"`
}

var rasterFormats = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "bmp": true,
}

// WP resized file: hero-1024x768.png / hero-465x203.png
var sizeSuffixRE = regexp.MustCompile(`-\d+x\d+(\.[A-Za-z0-9]+)(\?.*)?$`)

// CollectHeavyImages dedupes by lowercase webpRel (same stem .png+.jpg → one entry,
// prefer larger bytes) and keeps only IsHeavy.
func CollectHeavyImages(images []analytics.AggregatedImage) []ManifestImage {
	byKey := make(map[string]int)
	out := make([]ManifestImage, 0)

	for _, img := range images {
		if img.URL == "" {
			continue
		}
		format := strings.ToLower(img.Format)
		if format == "" {
			format = guessFormat(img.URL)
		}
		if format == "svg" || format == "avif" || format == "webp" {
			continue
		}
		if !rasterFormats[format] {
			continue
		}

		sourceURL := preferOriginalURL(img.URL)
		basename, pathHint := basenameAndHint(sourceURL)
		webpRel := webpRelFromHint(pathHint, basename)
		key := strings.ToLower(webpRel)
		if key == "" || key == ".webp" {
			key = strings.ToLower(strings.Split(sourceURL, "?")[0])
		}

		pages := img.Pages
		if pages == nil {
			pages = []string{}
		}

		if idx, ok := byKey[key]; ok {
			ex := &out[idx]
			ex.Pages = mergePages(ex.Pages, pages)
			if img.IsHeavy {
				ex.IsHeavy = true
			}
			if img.NaturalWidth > ex.NaturalWidth {
				ex.NaturalWidth = img.NaturalWidth
			}
			if img.NaturalHeight > ex.NaturalHeight {
				ex.NaturalHeight = img.NaturalHeight
			}
			if img.MaxRenderedWidth > ex.MaxRenderedWidth {
				ex.MaxRenderedWidth = img.MaxRenderedWidth
			}
			if img.MaxRenderedHeight > ex.MaxRenderedHeight {
				ex.MaxRenderedHeight = img.MaxRenderedHeight
			}
			if img.RecommendedRetinaWidth > ex.RecommendedRetinaWidth {
				ex.RecommendedRetinaWidth = img.RecommendedRetinaWidth
			}
			if img.RecommendedRetinaHeight > ex.RecommendedRetinaHeight {
				ex.RecommendedRetinaHeight = img.RecommendedRetinaHeight
			}
			// Prefer larger source when same webpRel (png vs jpg collision)
			if img.MaxTransferSize > ex.Bytes {
				ex.Bytes = img.MaxTransferSize
				ex.SourceURL = sourceURL
				ex.PathHint = pathHint
				ex.Basename = basename
				ex.Format = format
				ex.WebpRel = webpRel
			} else if hasSizeSuffix(ex.SourceURL) && !hasSizeSuffix(sourceURL) {
				ex.SourceURL = sourceURL
			}
			continue
		}

		byKey[key] = len(out)
		out = append(out, ManifestImage{
			SourceURL:               sourceURL,
			PathHint:                pathHint,
			WebpRel:                 webpRel,
			Basename:                basename,
			Format:                  format,
			IsHeavy:                 img.IsHeavy,
			Bytes:                   img.MaxTransferSize,
			Pages:                   append([]string(nil), pages...),
			NaturalWidth:            img.NaturalWidth,
			NaturalHeight:           img.NaturalHeight,
			MaxRenderedWidth:        img.MaxRenderedWidth,
			MaxRenderedHeight:       img.MaxRenderedHeight,
			RecommendedRetinaWidth:  img.RecommendedRetinaWidth,
			RecommendedRetinaHeight: img.RecommendedRetinaHeight,
		})
	}

	heavy := make([]ManifestImage, 0, len(out))
	for _, im := range out {
		// Include if:
		// 1. Marked heavy from network (IsHeavy == true)
		// 2. TransferSize was 0 (lazy-loaded in DOM, actual size verified on download)
		// 3. Candidate was a resized thumbnail (hasSizeSuffix == true) where master image on server may exceed threshold
		if im.IsHeavy || im.Bytes == 0 || hasSizeSuffix(im.SourceURL) {
			heavy = append(heavy, im)
		}
	}
	return heavy
}

// ConvertHeavyImages downloads and converts each image via pkg/optimizer.
// Does NOT write into WordPress uploads — PHP apply copies from the package.
func ConvertHeavyImages(images []ManifestImage, quality float32, authUser, authPass string) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithThreshold(images, quality, 0, true, true, authUser, authPass)
}

// ConvertHeavyImagesWithThreshold converts images with custom threshold byte budget using concurrent workers.
func ConvertHeavyImagesWithThreshold(images []ManifestImage, quality float32, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithProgress(images, quality, thresholdBytes, adaptive, resizeToRetina, authUser, authPass, nil)
}

// ConvertHeavyImagesWithProgress converts images with custom threshold byte budget and reports live progress.
func ConvertHeavyImagesWithProgress(images []ManifestImage, quality float32, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string, onProgress func(done, total int, name string)) ([]WrittenImage, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images to convert")
	}

	numWorkers := runtime.NumCPU() * 4
	if numWorkers < 8 {
		numWorkers = 8
	}
	if numWorkers > 32 {
		numWorkers = 32
	}
	if numWorkers > len(images) {
		numWorkers = len(images)
	}

	type convertTask struct {
		index int
		img   ManifestImage
	}

	type convertResult struct {
		index int
		item  *WrittenImage
		err   error
	}

	tasks := make(chan convertTask, len(images))
	resultsChan := make(chan convertResult, len(images))
	var processed int32
	totalImages := len(images)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				img := task.img
				func() {
					defer func() {
						done := int(atomic.AddInt32(&processed, 1))
						if onProgress != nil {
							onProgress(done, totalImages, img.Basename)
						}
					}()

					maxW := 0
					maxH := 0
					if resizeToRetina && img.MaxRenderedWidth > 0 {
						maxW = img.MaxRenderedWidth * 2
						maxH = img.MaxRenderedHeight * 2
					}

					res, err := optimizer.ConvertImageURLToWebPAdaptiveBudgetAuthResize(img.SourceURL, quality, thresholdBytes, adaptive, maxW, maxH, authUser, authPass)
					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: %w", img.SourceURL, err)}
						return
					}
					webpData, err := decodeDataURL(res.OptimizedWebPBase64)
					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("decode webp %s: %w", img.SourceURL, err)}
						return
					}
					origData, err := decodeDataURL(res.OriginalDataBase64)
					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("decode orig %s: %w", img.SourceURL, err)}
						return
					}

					// If thresholdBytes is set, ensure the actual downloaded original meets the heavy threshold
					if thresholdBytes > 0 && int64(len(origData)) < thresholdBytes {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: downloaded size %d B < threshold %d B", img.SourceURL, len(origData), thresholdBytes)}
						return
					}

					if res.OriginalWidth > 0 {
						img.NaturalWidth = res.OriginalWidth
					}
					if res.OriginalHeight > 0 {
						img.NaturalHeight = res.OriginalHeight
					}
					if img.MaxRenderedWidth > 0 {
						retW := img.MaxRenderedWidth * 2
						retH := img.MaxRenderedHeight * 2
						if img.NaturalWidth > 0 && retW > img.NaturalWidth {
							retW = img.NaturalWidth
						}
						if img.NaturalHeight > 0 && retH > img.NaturalHeight {
							retH = img.NaturalHeight
						}
						img.RecommendedRetinaWidth = retW
						img.RecommendedRetinaHeight = retH
					}

					rel := img.WebpRel
					if rel == "" {
						rel = webpRelFromHint(img.PathHint, img.Basename)
					}
					rel = filepath.ToSlash(rel)
					img.WebpRel = rel

					ext := path.Ext(img.Basename)
					if ext == "" {
						ext = "." + img.Format
					}

					optW := res.OptimizedWidth
					optH := res.OptimizedHeight
					if optW == 0 {
						optW = img.NaturalWidth
					}
					if optH == 0 {
						optH = img.NaturalHeight
					}

					written := &WrittenImage{
						ManifestImage:      img,
						OrigExt:            strings.TrimPrefix(strings.ToLower(ext), "."),
						OrigData:           origData,
						WebPData:           webpData,
						OptimizedWidth:     optW,
						OptimizedHeight:    optH,
						OriginalBytes:      res.OriginalBytes,
						OptimizedBytes:     res.OptimizedBytes,
						SavingsPercent:     res.SavingsPercent,
						OriginalFormatted:  res.OriginalFormatted,
						OptimizedFormatted: res.OptimizedFormatted,
					}
					resultsChan <- convertResult{index: task.index, item: written}
				}()
			}
		}()
	}

	for i, img := range images {
		tasks <- convertTask{index: i, img: img}
	}
	close(tasks)

	wg.Wait()
	close(resultsChan)

	rawResults := make([]*WrittenImage, len(images))
	for res := range resultsChan {
		if res.err != nil {
			fmt.Printf("[wpexport] %v\n", res.err)
			continue
		}
		rawResults[res.index] = res.item
	}

	ok := make([]WrittenImage, 0, len(images))
	for _, item := range rawResults {
		if item != nil {
			id := fmt.Sprintf("%03d", len(ok)+1)
			item.ID = id
			item.PackageWebP = fmt.Sprintf("images/%s/optimized.webp", id)
			ok = append(ok, *item)
		}
	}

	if len(ok) == 0 {
		return nil, fmt.Errorf("no WebP conversions succeeded (check auth / image URLs)")
	}
	return ok, nil
}

// WriteWebPFiles is deprecated: use ConvertHeavyImages + WriteDeployPackage.
// Kept as a thin wrapper that only converts (does not touch uploads).
func WriteWebPFiles(wordpressPath string, images []ManifestImage, quality float32, authUser, authPass string) ([]WrittenImage, int, error) {
	_ = wordpressPath
	written, err := ConvertHeavyImages(images, quality, authUser, authPass)
	if err != nil {
		return nil, 0, err
	}
	return written, len(written), nil
}

// WriteDeployPackage writes a self-contained folder:
// apply.php, rollback.php, compare.html, manifest.json, images/NNN/{original,optimized}.
// PHP apply copies optimized.webp → wp uploads/{webpRel} and retargets attachments.
func WriteDeployPackage(packageDir, domain string, cfg config.ScanConfig, written []WrittenImage) (*ExportResult, error) {
	if len(written) == 0 {
		return nil, fmt.Errorf("no images for deploy package")
	}
	pkg := strings.TrimSpace(packageDir)
	if pkg == "" {
		return nil, fmt.Errorf("package dir is required")
	}
	if err := os.MkdirAll(pkg, 0755); err != nil {
		return nil, err
	}

	for _, im := range written {
		id := im.ID
		if id == "" {
			return nil, fmt.Errorf("missing package id for %s", im.Basename)
		}
		ext := im.OrigExt
		if ext == "" {
			ext = "bin"
		}
		imgDir := filepath.Join(pkg, "images", id)
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(imgDir, "original."+ext), im.OrigData, 0644); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(imgDir, "optimized.webp"), im.WebPData, 0644); err != nil {
			return nil, err
		}
	}

	manifestImgs := ManifestImagesFromWritten(written)
	php, err := BuildApplyPHPFromManifest(domain, cfg, pkg, manifestImgs)
	if err != nil {
		return nil, err
	}
	applyPath := filepath.Join(pkg, "apply.php")
	if err := os.WriteFile(applyPath, []byte(php), 0644); err != nil {
		return nil, err
	}
	rollbackPath := filepath.Join(pkg, "rollback.php")
	if err := os.WriteFile(rollbackPath, []byte(BuildRollbackPHP(pkg)), 0644); err != nil {
		return nil, err
	}

	zipBytes, err := BuildReviewZIP(domain, written)
	if err != nil {
		return nil, err
	}
	// Also materialize compare.html + manifest.json beside apply.php (same as ZIP root).
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != "compare.html" && f.Name != "manifest.json" && f.Name != "render-report.html" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(pkg, f.Name), data, 0644); err != nil {
			return nil, err
		}
	}

	zipPath := pkg + ".zip"
	if err := os.WriteFile(zipPath, zipBytes, 0644); err != nil {
		// Non-fatal for folder use; still return package.
		zipPath = ""
	} else {
		// Prefer a full package zip that includes apply.php — rebuild with apply inside.
		fullZip, zerr := buildPackageZIP(pkg)
		if zerr == nil {
			_ = os.WriteFile(zipPath, fullZip, 0644)
		}
	}

	return &ExportResult{
		ApplyPHP:      applyPath,
		RollbackPHP:   rollbackPath,
		ReviewZIP:     zipPath,
		PackageDir:    pkg,
		WebPCount:     len(written),
		WordPressPath: pkg,
	}, nil
}

// BuildReviewZIP packs orig + webp + compare.html + manifest.json for task handoff.
func BuildReviewZIP(domain string, images []WrittenImage) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images for review ZIP")
	}

	type zipEntry struct {
		ID                      string   `json:"id"`
		SourceURL               string   `json:"sourceUrl"`
		PathHint                string   `json:"pathHint"`
		WebpRel                 string   `json:"webpRel"`
		Basename                string   `json:"basename"`
		Pages                   []string `json:"pages"`
		NaturalWidth            int      `json:"naturalWidth,omitempty"`
		NaturalHeight           int      `json:"naturalHeight,omitempty"`
		OptimizedWidth          int      `json:"optimizedWidth,omitempty"`
		OptimizedHeight         int      `json:"optimizedHeight,omitempty"`
		MaxRenderedWidth        int      `json:"maxRenderedWidth,omitempty"`
		MaxRenderedHeight       int      `json:"maxRenderedHeight,omitempty"`
		RecommendedRetinaWidth  int      `json:"recommendedRetinaWidth,omitempty"`
		RecommendedRetinaHeight int      `json:"recommendedRetinaHeight,omitempty"`
		OriginalBytes           int64    `json:"originalBytes"`
		OptimizedBytes          int64    `json:"optimizedBytes"`
		SavingsPercent          float64  `json:"savingsPercent"`
		OriginalFormatted       string   `json:"originalFormatted"`
		OptimizedFormatted      string   `json:"optimizedFormatted"`
		OriginalPath            string   `json:"originalPath"`
		OptimizedPath           string   `json:"optimizedPath"`
	}

	entries := make([]zipEntry, 0, len(images))
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for i, im := range images {
		id := im.ID
		if id == "" {
			id = fmt.Sprintf("%03d", i+1)
		}
		ext := im.OrigExt
		if ext == "" {
			ext = "bin"
		}
		origPath := fmt.Sprintf("images/%s/original.%s", id, ext)
		webpPath := fmt.Sprintf("images/%s/optimized.webp", id)

		if err := writeZipFile(zw, origPath, im.OrigData); err != nil {
			_ = zw.Close()
			return nil, err
		}
		if err := writeZipFile(zw, webpPath, im.WebPData); err != nil {
			_ = zw.Close()
			return nil, err
		}

		entries = append(entries, zipEntry{
			ID:                      id,
			SourceURL:               im.SourceURL,
			PathHint:                im.PathHint,
			WebpRel:                 im.WebpRel,
			Basename:                im.Basename,
			Pages:                   im.Pages,
			NaturalWidth:            im.NaturalWidth,
			NaturalHeight:           im.NaturalHeight,
			OptimizedWidth:          im.OptimizedWidth,
			OptimizedHeight:         im.OptimizedHeight,
			MaxRenderedWidth:        im.MaxRenderedWidth,
			MaxRenderedHeight:       im.MaxRenderedHeight,
			RecommendedRetinaWidth:  im.RecommendedRetinaWidth,
			RecommendedRetinaHeight: im.RecommendedRetinaHeight,
			OriginalBytes:           im.OriginalBytes,
			OptimizedBytes:          im.OptimizedBytes,
			SavingsPercent:          im.SavingsPercent,
			OriginalFormatted:       im.OriginalFormatted,
			OptimizedFormatted:      im.OptimizedFormatted,
			OriginalPath:            origPath,
			OptimizedPath:           webpPath,
		})
	}

	mani := map[string]interface{}{
		"domain":    domain,
		"generated": time.Now().UTC().Format(time.RFC3339),
		"count":     len(entries),
		"images":    entries,
	}
	maniJSON, err := json.MarshalIndent(mani, "", "  ")
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := writeZipFile(zw, "manifest.json", maniJSON); err != nil {
		_ = zw.Close()
		return nil, err
	}

	// Generate render-report.html (documenting cross-page render optimization)
	var renderReportBuf strings.Builder
	renderReportBuf.WriteString("<!DOCTYPE html><html lang=\"uk\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>Звіт оптимізації рендеру зображень — ")
	renderReportBuf.WriteString(esc(domain))
	renderReportBuf.WriteString("</title><style>")
	renderReportBuf.WriteString("body{font-family:system-ui,-apple-system,sans-serif;margin:0;padding:16px 24px;background:#0f172a;color:#f8fafc}")
	renderReportBuf.WriteString(".container{width:100%;max-width:100%;margin:0;box-sizing:border-box}")
	renderReportBuf.WriteString("h1{color:#38bdf8;font-size:24px;margin:0 0 8px}p.meta{color:#94a3b8;font-size:13px;margin:0 0 20px}")
	renderReportBuf.WriteString("table{width:100%;border-collapse:separate;border-spacing:0;background:#1e293b;border-radius:12px;border:1px solid #334155;font-size:13px}")
	renderReportBuf.WriteString("thead{position:sticky;top:0;z-index:100}")
	renderReportBuf.WriteString("thead th{position:sticky;top:0;background:#090d16;color:#94a3b8;padding:14px 12px;text-align:left;font-size:11px;text-transform:uppercase;letter-spacing:0.05em;z-index:100;box-shadow:0 3px 6px rgba(0,0,0,0.7);border-bottom:2px solid #334155;white-space:nowrap}")
	renderReportBuf.WriteString("thead th:first-child{border-top-left-radius:12px}")
	renderReportBuf.WriteString("thead th:last-child{border-top-right-radius:12px}")
	renderReportBuf.WriteString("td{padding:12px;border-bottom:1px solid #334155;vertical-align:top}")
	renderReportBuf.WriteString("tr:last-child td:first-child{border-bottom-left-radius:12px}")
	renderReportBuf.WriteString("tr:last-child td:last-child{border-bottom-right-radius:12px}")
	renderReportBuf.WriteString(".pages-list{font-size:12px;color:#cbd5e1;list-style:disc;padding-left:18px;margin:4px 0}")
	renderReportBuf.WriteString(".pages-list a{color:#38bdf8;text-decoration:underline;word-break:break-all}")
	renderReportBuf.WriteString(".badge-opt{color:#10b981;background:rgba(16,185,129,0.15);padding:2px 8px;border-radius:4px;font-weight:bold;font-size:11px;white-space:nowrap}")
	renderReportBuf.WriteString(".badge-dim{color:#38bdf8;font-family:monospace;font-size:12px;font-weight:bold;white-space:nowrap}")
	renderReportBuf.WriteString(".badge-orig-dim{color:#f59e0b;font-family:monospace;font-size:12px;font-weight:bold;white-space:nowrap}")
	renderReportBuf.WriteString(".badge-tag{display:inline-block;padding:2px 6px;border-radius:4px;font-size:10px;font-weight:700;margin-top:4px;white-space:nowrap;letter-spacing:0.03em}")
	renderReportBuf.WriteString(".badge-resized{background:rgba(168,85,247,0.2);color:#c084fc;border:1px solid rgba(168,85,247,0.4)}")
	renderReportBuf.WriteString(".badge-native{background:rgba(56,189,248,0.15);color:#38bdf8;border:1px solid rgba(56,189,248,0.3)}")
	renderReportBuf.WriteString(".preview-box{display:flex;gap:10px;margin-top:10px;align-items:center}")
	renderReportBuf.WriteString(".thumb-card{position:relative;background:#0b1120;border:1px solid #334155;border-radius:6px;padding:4px;display:inline-flex;flex-direction:column;align-items:center}")
	renderReportBuf.WriteString(".thumb-badge{font-size:9px;font-weight:700;text-transform:uppercase;padding:2px 6px;border-radius:3px;margin-bottom:4px;letter-spacing:0.04em}")
	renderReportBuf.WriteString(".thumb-orig{background:rgba(245,158,11,0.2);color:#f59e0b;border:1px solid rgba(245,158,11,0.4)}")
	renderReportBuf.WriteString(".thumb-webp{background:rgba(16,185,129,0.2);color:#10b981;border:1px solid rgba(16,185,129,0.4)}")
	renderReportBuf.WriteString(".thumb-img{max-width:110px;max-height:75px;width:auto;height:auto;object-fit:contain;border-radius:4px;background:#1e293b;transition:transform 0.15s ease;cursor:zoom-in}")
	renderReportBuf.WriteString(".thumb-img:hover{transform:scale(1.06)}")
	renderReportBuf.WriteString(".toolbar{display:flex;gap:10px;margin:16px 0 20px;align-items:center;flex-wrap:wrap}")
	renderReportBuf.WriteString(".search-input{background:#1e293b;border:1px solid #334155;border-radius:8px;padding:8px 14px;color:#f8fafc;font-size:13px;width:300px;outline:none}")
	renderReportBuf.WriteString(".search-input:focus{border-color:#38bdf8;box-shadow:0 0 0 2px rgba(56,189,248,0.2)}")
	renderReportBuf.WriteString(".filter-btn{background:#1e293b;border:1px solid #334155;border-radius:8px;padding:7px 12px;color:#94a3b8;font-size:12px;font-weight:600;cursor:pointer;transition:all 0.15s ease}")
	renderReportBuf.WriteString(".filter-btn:hover{background:#334155;color:#f8fafc}")
	renderReportBuf.WriteString(".filter-btn.active{background:#0284c7;border-color:#38bdf8;color:#fff;font-weight:700}")
	renderReportBuf.WriteString(".filter-btn.btn-resized.active{background:#9333ea;border-color:#c084fc;color:#fff}")
	renderReportBuf.WriteString(".filter-btn.btn-native.active{background:#0d9488;border-color:#2dd4bf;color:#fff}")
	renderReportBuf.WriteString(".filter-btn.btn-heavy.active{background:#e11d48;border-color:#fb7185;color:#fff}")
	renderReportBuf.WriteString("</style></head><body><div class=\"container\">")
	renderReportBuf.WriteString("<h1>📊 Звіт оптимізації за рендером на сторінках (Render-Aware WebP Optimization)</h1>")
	renderReportBuf.WriteString(fmt.Sprintf("<p class=\"meta\">Домен: %s · Всього зображень: %d · Згенеровано: %s</p>", esc(domain), len(entries), time.Now().UTC().Format("2006-01-02 15:04:05 UTC")))

	renderReportBuf.WriteString("<div class=\"toolbar\">")
	renderReportBuf.WriteString("<input type=\"text\" id=\"searchInput\" placeholder=\"🔍 Пошук за назвою або URL...\" oninput=\"filterReport()\" class=\"search-input\" />")
	renderReportBuf.WriteString("<button type=\"button\" class=\"filter-btn active\" onclick=\"setFilter('all', this)\">Всі (<span id=\"countAll\">0</span>)</button>")
	renderReportBuf.WriteString("<button type=\"button\" class=\"filter-btn btn-resized\" onclick=\"setFilter('resized', this)\">📐 Ресайз під рендер (<span id=\"countResized\">0</span>)</button>")
	renderReportBuf.WriteString("<button type=\"button\" class=\"filter-btn btn-native\" onclick=\"setFilter('native', this)\">⚡ 100% оригінал (<span id=\"countNative\">0</span>)</button>")
	renderReportBuf.WriteString("<button type=\"button\" class=\"filter-btn btn-heavy\" onclick=\"setFilter('heavy', this)\">🔥 > 1 MB (<span id=\"countHeavy\">0</span>)</button>")
	renderReportBuf.WriteString("<span id=\"visibleCount\" style=\"color:#94a3b8;font-size:12px;margin-left:auto;font-weight:600;\"></span>")
	renderReportBuf.WriteString("</div>")

	renderReportBuf.WriteString("<table><thead><tr><th>#</th><th>Зображення</th><th>Сторінки використання</th><th>Оригінальні розміри (W×H)</th><th>Фактичний рендер</th><th>Retina 2x ціль</th><th>Оригінал (вага)</th><th>Оптимізований WebP</th><th>Економія</th></tr></thead><tbody>")

	for idx, e := range entries {
		pagesHTML := strings.Builder{}
		if len(e.Pages) > 0 {
			pagesHTML.WriteString("<ul class=\"pages-list\">")
			for _, pg := range e.Pages {
				pagesHTML.WriteString(fmt.Sprintf("<li><a href=\"%s\" target=\"_blank\">%s</a></li>", esc(pg), esc(pg)))
			}
			pagesHTML.WriteString("</ul>")
		} else {
			pagesHTML.WriteString("<span style=\"color:#64748b;\">—</span>")
		}

		origDimText := "—"
		if e.NaturalWidth > 0 {
			origDimText = fmt.Sprintf("<span class=\"badge-orig-dim\">%d×%d px</span>", e.NaturalWidth, e.NaturalHeight)
		}
		rendText := "—"
		if e.MaxRenderedWidth > 0 {
			rendText = fmt.Sprintf("<span class=\"badge-dim\">%d×%d px</span>", e.MaxRenderedWidth, e.MaxRenderedHeight)
		}
		retinaText := "—"
		if e.RecommendedRetinaWidth > 0 {
			retinaText = fmt.Sprintf("<span class=\"badge-dim\" style=\"color:#10b981;\">%d×%d px</span>", e.RecommendedRetinaWidth, e.RecommendedRetinaHeight)
		}

		optDimText := ""
		if e.OptimizedWidth > 0 && e.OptimizedHeight > 0 {
			tagHTML := ""
			if e.NaturalWidth > 0 && e.OptimizedWidth < e.NaturalWidth {
				tagHTML = "<br><span class=\"badge-tag badge-resized\">📐 Ресайз під рендер</span>"
			} else {
				tagHTML = "<br><span class=\"badge-tag badge-native\">⚡ 100% оригінал</span>"
			}
			optDimText = fmt.Sprintf("%s<br><span class=\"badge-dim\" style=\"color:#10b981;font-size:11px;\">%d×%d px</span>", tagHTML, e.OptimizedWidth, e.OptimizedHeight)
		}

		previewHTML := fmt.Sprintf("<div class=\"preview-box\"><div class=\"thumb-card\"><span class=\"thumb-badge thumb-orig\">Оригінал (Before)</span><a href=\"%s\" target=\"_blank\"><img src=\"%s\" loading=\"lazy\" class=\"thumb-img\" alt=\"Original\" /></a></div><div class=\"thumb-card\"><span class=\"thumb-badge thumb-webp\">WebP (After)</span><a href=\"%s\" target=\"_blank\"><img src=\"%s\" loading=\"lazy\" class=\"thumb-img\" alt=\"WebP\" /></a></div></div>",
			esc(e.OriginalPath), esc(e.OriginalPath), esc(e.OptimizedPath), esc(e.OptimizedPath))

		renderReportBuf.WriteString(fmt.Sprintf("<tr><td>%d</td><td><strong style=\"color:#f1f5f9;font-size:14px;\">%s</strong><br><a href=\"%s\" target=\"_blank\" style=\"color:#64748b;font-size:11px;word-break:break-all;\">%s</a>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><strong style=\"color:#10b981;\">%s</strong>%s</td><td><span class=\"badge-opt\">-%.1f%%</span></td></tr>",
			idx+1, esc(e.Basename), esc(e.SourceURL), esc(e.SourceURL), previewHTML, pagesHTML.String(), origDimText, rendText, retinaText, esc(e.OriginalFormatted), esc(e.OptimizedFormatted), optDimText, e.SavingsPercent))
	}
	renderReportBuf.WriteString("</tbody></table></div>")

	// Vanilla JS filter & search script
	renderReportBuf.WriteString("<script>")
	renderReportBuf.WriteString("let currentFilter='all';")
	renderReportBuf.WriteString("function setFilter(f,btn){currentFilter=f;document.querySelectorAll('.filter-btn').forEach(b=>b.classList.remove('active'));btn.classList.add('active');filterReport();}")
	renderReportBuf.WriteString("function filterReport(){const q=document.getElementById('searchInput').value.toLowerCase().trim();const rows=document.querySelectorAll('tbody tr');let visible=0;rows.forEach(tr=>{const text=tr.innerText.toLowerCase();const isResized=tr.querySelector('.badge-resized')!==null;const origSizeText=tr.children[6]?.innerText||'';const isHeavy=origSizeText.includes('MB');let mf=true;if(currentFilter==='resized')mf=isResized;else if(currentFilter==='native')mf=!isResized;else if(currentFilter==='heavy')mf=isHeavy;const mq=!q||text.includes(q);if(mf&&mq){tr.style.display='';visible++;}else{tr.style.display='none';}});document.getElementById('visibleCount').innerText=`Відображено: ${visible} з ${rows.length}`;}")
	renderReportBuf.WriteString("window.addEventListener('DOMContentLoaded',()=>{const rows=document.querySelectorAll('tbody tr');let r=0,n=0,h=0;rows.forEach(tr=>{if(tr.querySelector('.badge-resized'))r++;else n++;if((tr.children[6]?.innerText||'').includes('MB'))h++;});document.getElementById('countAll').innerText=rows.length;document.getElementById('countResized').innerText=r;document.getElementById('countNative').innerText=n;document.getElementById('countHeavy').innerText=h;document.getElementById('visibleCount').innerText=`Відображено: ${rows.length} з ${rows.length}`;});")
	renderReportBuf.WriteString("</script></body></html>")

	if err := writeZipFile(zw, "render-report.html", []byte(renderReportBuf.String())); err != nil {
		_ = zw.Close()
		return nil, err
	}

	entriesJSON, _ := json.Marshal(entries)

	var htmlBuf strings.Builder
	htmlBuf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>SpeedMap WebP review — ")
	htmlBuf.WriteString(esc(domain))
	htmlBuf.WriteString("</title><style>")
	htmlBuf.WriteString("body{font-family:system-ui,-apple-system,sans-serif;margin:24px;color:#111;background:#fafafa}")
	htmlBuf.WriteString("h1{font-size:1.25rem;margin:0 0 8px}p.meta{color:#555;font-size:13px;margin:0 0 24px}")
	htmlBuf.WriteString(".pair{display:flex;flex-direction:column;gap:16px;max-width:1200px;margin:0 0 48px;padding:0 0 32px;border-bottom:1px solid #ddd}")
	htmlBuf.WriteString(".pair h2{font-size:1rem;font-weight:600;margin:0;color:#222}")
	htmlBuf.WriteString(".ctx{font-size:13px;color:#444;line-height:1.5}")
	htmlBuf.WriteString(".ctx a{color:#0645ad;word-break:break-all}")
	htmlBuf.WriteString(".ctx .more{color:#666}")
	htmlBuf.WriteString("figure{margin:0}.pair img{display:block;width:100%;max-width:100%;height:auto;background:#eee;cursor:zoom-in;transition:opacity 0.15s;border-radius:4px}.pair img:hover{opacity:0.92;box-shadow:0 4px 12px rgba(0,0,0,0.1)}")
	htmlBuf.WriteString("figcaption{font-size:13px;color:#444;margin-top:8px} .sav{color:#066;font-weight:bold}")

	// Lightbox Styles
	htmlBuf.WriteString("#lightbox{display:none;position:fixed;inset:0;background:rgba(5,10,20,0.95);backdrop-filter:blur(10px);z-index:99999;flex-direction:column;justify-content:space-between;padding:16px 24px;box-sizing:border-box}")
	htmlBuf.WriteString(".lb-header{display:flex;align-items:center;justify-content:space-between;width:100%;gap:16px;color:#f1f5f9}")
	htmlBuf.WriteString(".lb-title-wrap{display:flex;align-items:center;gap:12px;min-width:0}")
	htmlBuf.WriteString(".lb-title{font-size:15px;font-weight:700;color:#fff;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:40vw}")
	htmlBuf.WriteString(".badge-orig{background:rgba(239,68,68,0.2);color:#f87171;border:1px solid rgba(239,68,68,0.4);padding:3px 10px;border-radius:20px;font-size:11px;font-weight:700}")
	htmlBuf.WriteString(".badge-webp{background:rgba(16,185,129,0.2);color:#34d399;border:1px solid rgba(16,185,129,0.4);padding:3px 10px;border-radius:20px;font-size:11px;font-weight:700}")
	htmlBuf.WriteString(".lb-mode-toggle{display:flex;background:#1e293b;border-radius:24px;padding:3px;border:1px solid #334155;gap:4px}")
	htmlBuf.WriteString(".lb-tab{background:transparent;border:none;color:#94a3b8;padding:6px 16px;border-radius:20px;font-size:12px;font-weight:700;cursor:pointer;transition:all 0.2s}")
	htmlBuf.WriteString(".lb-tab:hover{color:#fff}")
	htmlBuf.WriteString(".lb-tab-active-orig{background:#dc2626!important;color:#fff!important;box-shadow:0 2px 8px rgba(220,38,38,0.4)}")
	htmlBuf.WriteString(".lb-tab-active-webp{background:#059669!important;color:#fff!important;box-shadow:0 2px 8px rgba(5,150,105,0.4)}")
	htmlBuf.WriteString(".lb-close-btn{background:#334155;color:#f1f5f9;border:none;padding:6px 14px;border-radius:10px;font-size:12px;font-weight:bold;cursor:pointer}")
	htmlBuf.WriteString(".lb-close-btn:hover{background:#475569}")

	htmlBuf.WriteString(".lb-body{position:relative;flex:1;display:flex;align-items:center;justify-content:center;width:100%;height:calc(100vh - 130px);margin:8px 0}")
	htmlBuf.WriteString(".lb-main-img{max-height:calc(100vh - 140px);max-width:88vw;object-fit:contain;border-radius:8px;box-shadow:0 20px 40px rgba(0,0,0,0.8);background:transparent;opacity:1!important;transition:none!important;cursor:pointer}.lb-main-img:hover{opacity:1!important;box-shadow:0 20px 40px rgba(0,0,0,0.8)!important}")
	htmlBuf.WriteString(".lb-nav-btn{position:absolute;top:50%;transform:translateY(-50%);background:rgba(30,41,59,0.85);color:#fff;border:1px solid #475569;width:48px;height:48px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:22px;font-weight:bold;cursor:pointer;transition:all 0.2s;user-select:none;z-index:10}")
	htmlBuf.WriteString(".lb-nav-btn:hover{background:#2563eb;border-color:#60a5fa;transform:translateY(-50%) scale(1.08)}")
	htmlBuf.WriteString(".lb-prev{left:16px}")
	htmlBuf.WriteString(".lb-next{right:16px}")

	htmlBuf.WriteString(".lb-footer{display:flex;align-items:center;justify-content:space-between;width:100%;color:#94a3b8;font-size:12px;font-family:monospace;border-top:1px solid #1e293b;padding-top:8px}")
	htmlBuf.WriteString(".lb-shortcuts{color:#64748b;font-size:11px}")
	htmlBuf.WriteString("</style></head><body>")

	htmlBuf.WriteString("<h1>SpeedMap WebP review</h1>")
	htmlBuf.WriteString("<p class=\"meta\">")
	htmlBuf.WriteString(esc(domain))
	htmlBuf.WriteString(" · ")
	htmlBuf.WriteString(fmt.Sprintf("%d", len(entries)))
	htmlBuf.WriteString(" images · open this file from the unzipped archive (клікніть на будь-яке зображення для інтерактивного перегляду Before/After)</p>")

	for idx, e := range entries {
		htmlBuf.WriteString("<section class=\"pair\">")
		htmlBuf.WriteString("<h2>")
		htmlBuf.WriteString(esc(e.Basename))
		htmlBuf.WriteString("</h2>")
		writeReviewContextHTML(&htmlBuf, e.SourceURL, e.Pages)
		htmlBuf.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"original\" onclick=\"openGallery(%d, 'orig')\"><figcaption>Before · %s · %s</figcaption></figure>", esc(e.OriginalPath), idx, esc(e.Basename), esc(e.OriginalFormatted)))
		htmlBuf.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"webp\" onclick=\"openGallery(%d, 'webp')\"><figcaption>After · WebP · %s · <span class=\"sav\">−%.1f%%</span></figcaption></figure>", esc(e.OptimizedPath), idx, esc(e.OptimizedFormatted), e.SavingsPercent))
		htmlBuf.WriteString("</section>")
	}

	// Fullscreen Gallery Modal DOM
	htmlBuf.WriteString("<div id=\"lightbox\" onclick=\"onBackdropClick(event)\">")
	htmlBuf.WriteString("<div class=\"lb-header\">")
	htmlBuf.WriteString("<div class=\"lb-title-wrap\"><span id=\"lb-count\" style=\"color:#38bdf8;font-weight:bold;font-family:monospace;\"></span><span id=\"lb-title\" class=\"lb-title\"></span><span id=\"lb-mode-badge\" class=\"badge-webp\"></span></div>")
	htmlBuf.WriteString("<div class=\"lb-mode-toggle\">")
	htmlBuf.WriteString("<button id=\"btn-before\" class=\"lb-tab\" onclick=\"toggleMode('orig')\">🔴 Before (Original)</button>")
	htmlBuf.WriteString("<button id=\"btn-after\" class=\"lb-tab lb-tab-active-webp\" onclick=\"toggleMode('webp')\">🟢 After (WebP)</button>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("<button class=\"lb-close-btn\" onclick=\"closeLb()\">✕ Закрити (Esc)</button>")
	htmlBuf.WriteString("</div>")

	htmlBuf.WriteString("<div class=\"lb-body\">")
	htmlBuf.WriteString("<div class=\"lb-nav-btn lb-prev\" onclick=\"prevItem()\" title=\"Попереднє (ArrowLeft)\">‹</div>")
	htmlBuf.WriteString("<img id=\"lb-img\" class=\"lb-main-img\" src=\"\" alt=\"preview\" onclick=\"toggleMode()\" title=\"Клацніть для перемикання Before ↔ After\">")
	htmlBuf.WriteString("<div class=\"lb-nav-btn lb-next\" onclick=\"nextItem()\" title=\"Наступне (ArrowRight)\">›</div>")
	htmlBuf.WriteString("</div>")

	htmlBuf.WriteString("<div class=\"lb-footer\">")
	htmlBuf.WriteString("<div id=\"lb-stats\" style=\"color:#e2e8f0;font-weight:bold;\"></div>")
	htmlBuf.WriteString("<div class=\"lb-shortcuts\">Гарячі клавіші: ⬅ ➡ (перегортання) | Пробіл / 1 / 2 (Before ↔ After) | Esc (вихід)</div>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("</div>")

	// Gallery Scripts
	htmlBuf.WriteString("<script>")
	htmlBuf.WriteString("const items = ")
	htmlBuf.Write(entriesJSON)
	htmlBuf.WriteString(";\n")
	htmlBuf.WriteString("let currentIndex = 0;\n")
	htmlBuf.WriteString("let currentMode = 'webp';\n")
	htmlBuf.WriteString(`
function openGallery(idx, mode) {
	currentIndex = idx;
	currentMode = mode || 'webp';
	renderGallery();
	document.getElementById('lightbox').style.display = 'flex';
}

function renderGallery() {
	if (currentIndex < 0) currentIndex = 0;
	if (currentIndex >= items.length) currentIndex = items.length - 1;
	const item = items[currentIndex];
	const isOrig = currentMode === 'orig';
	const imgEl = document.getElementById('lb-img');
	imgEl.src = isOrig ? item.originalPath : item.optimizedPath;
	
	document.getElementById('lb-title').textContent = item.basename;
	document.getElementById('lb-count').textContent = (currentIndex + 1) + ' / ' + items.length;
	
	const badgeEl = document.getElementById('lb-mode-badge');
	badgeEl.textContent = isOrig ? '🔴 Before (Original)' : '🟢 After (WebP)';
	badgeEl.className = isOrig ? 'badge-orig' : 'badge-webp';
	
	document.getElementById('btn-before').className = isOrig ? 'lb-tab lb-tab-active-orig' : 'lb-tab';
	document.getElementById('btn-after').className = !isOrig ? 'lb-tab lb-tab-active-webp' : 'lb-tab';
	
	const sav = item.savingsPercent > 0 ? ('-' + item.savingsPercent.toFixed(1) + '%') : '0%';
	document.getElementById('lb-stats').textContent = 
		'Before: ' + item.originalFormatted + ' ➜ After: ' + item.optimizedFormatted + ' (' + sav + ')';
}

function nextItem() {
	if (currentIndex < items.length - 1) {
		currentIndex++;
		renderGallery();
	}
}

function prevItem() {
	if (currentIndex > 0) {
		currentIndex--;
		renderGallery();
	}
}

function toggleMode(mode) {
	if (mode) {
		currentMode = mode;
	} else {
		currentMode = currentMode === 'orig' ? 'webp' : 'orig';
	}
	renderGallery();
}

function closeLb() {
	document.getElementById('lightbox').style.display = 'none';
}

function onBackdropClick(e) {
	if (e.target.id === 'lightbox' || e.target.classList.contains('lb-body')) {
		closeLb();
	}
}

document.addEventListener('keydown', function(e) {
	const lb = document.getElementById('lightbox');
	if (lb.style.display !== 'flex') return;
	
	if (e.key === 'ArrowLeft' || e.key === 'KeyA' || e.key === 'KeyH') {
		e.preventDefault();
		prevItem();
	} else if (e.key === 'ArrowRight' || e.key === 'KeyD' || e.key === 'KeyL') {
		e.preventDefault();
		nextItem();
	} else if (e.key === ' ' || e.key === 'Spacebar' || e.key === 'KeyB' || e.key === 'KeyW' || e.key === 'Tab') {
		e.preventDefault();
		toggleMode();
	} else if (e.key === '1') {
		e.preventDefault();
		toggleMode('orig');
	} else if (e.key === '2') {
		e.preventDefault();
		toggleMode('webp');
	} else if (e.key === 'Escape') {
		closeLb();
	}
});
</script>
</body></html>`)

	if err := writeZipFile(zw, "compare.html", []byte(htmlBuf.String())); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// buildPackageZIP zips an on-disk deploy package directory (apply.php + images/ + …).
func buildPackageZIP(packageDir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeZipFile(zw, rel, data)
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildApplyPHP creates a self-contained WP-CLI eval-file script with embedded manifest.
func BuildApplyPHP(domain string, cfg config.ScanConfig, images []analytics.AggregatedImage, wordpressPath string) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}

	heavy := CollectHeavyImages(images)
	if len(heavy) == 0 {
		return "", fmt.Errorf("no heavy convertible images in scan (raise threshold or rescan)")
	}
	return BuildApplyPHPFromManifest(domain, cfg, wpPath, heavy)
}

// BuildApplyPHPFromManifest embeds an already-filtered image list.
func BuildApplyPHPFromManifest(domain string, cfg config.ScanConfig, wordpressPath string, images []ManifestImage) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}
	if len(images) == 0 {
		return "", fmt.Errorf("no images for WP apply PHP")
	}

	q := int(cfg.NormalizedWebPQuality())
	m := Manifest{
		Domain:        domain,
		Generated:     time.Now().UTC().Format(time.RFC3339),
		Quality:       q,
		WordPressPath: wpPath,
		Images:        images,
	}

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}

	php := strings.Replace(applyTemplate, "{{SPEEDMAP_MANIFEST_JSON}}", string(raw), 1)
	if php == applyTemplate {
		return "", fmt.Errorf("template placeholder {{SPEEDMAP_MANIFEST_JSON}} not found")
	}
	php = strings.ReplaceAll(php, "{{WORDPRESS_PATH}}", wpPath)
	return php, nil
}

// BuildRollbackPHP returns a standalone rollback eval-file script.
func BuildRollbackPHP(wordpressPath string) string {
	return strings.ReplaceAll(rollbackTemplate, "{{WORDPRESS_PATH}}", strings.TrimSpace(wordpressPath))
}

// ManifestImagesFromWritten strips payloads for PHP embedding.
func ManifestImagesFromWritten(written []WrittenImage) []ManifestImage {
	out := make([]ManifestImage, len(written))
	for i, w := range written {
		out[i] = w.ManifestImage
	}
	return out
}

func webpRelFromHint(pathHint, basename string) string {
	base := basename
	if base == "" && pathHint != "" {
		base = path.Base(pathHint)
	}
	name := strings.TrimSuffix(base, path.Ext(base))
	if name == "" {
		name = "image"
	}
	webpName := name + ".webp"
	if pathHint == "" {
		return webpName
	}
	dir := path.Dir(pathHint)
	if dir == "." || dir == "/" {
		return webpName
	}
	return path.Join(dir, webpName)
}

func decodeDataURL(dataURL string) ([]byte, error) {
	idx := strings.Index(dataURL, ",")
	if idx == -1 {
		return nil, fmt.Errorf("invalid data URL")
	}
	return base64.StdEncoding.DecodeString(dataURL[idx+1:])
}

func preferOriginalURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return sizeSuffixRE.ReplaceAllString(rawURL, "$1$2")
	}
	u.Path = sizeSuffixRE.ReplaceAllString(u.Path, "$1")
	return u.String()
}

func hasSizeSuffix(rawURL string) bool {
	pathOnly := strings.Split(rawURL, "?")[0]
	return sizeSuffixRE.MatchString(pathOnly)
}

func mergePages(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, p := range append(a, b...) {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func guessFormat(rawURL string) string {
	u := strings.Split(rawURL, "?")[0]
	ext := strings.ToLower(path.Ext(u))
	return strings.TrimPrefix(ext, ".")
}

func basenameAndHint(rawURL string) (basename, pathHint string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		base := path.Base(strings.Split(rawURL, "?")[0])
		return stripSizeSuffix(base), ""
	}
	base := stripSizeSuffix(path.Base(u.Path))
	marker := "/wp-content/uploads/"
	if idx := strings.Index(u.Path, marker); idx >= 0 {
		rel := stripSizeSuffix(u.Path[idx+len(marker):])
		pathHint = strings.TrimPrefix(rel, "/")
	}
	return base, pathHint
}

func stripSizeSuffix(name string) string {
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	parts := strings.Split(stem, "-")
	if len(parts) < 2 {
		return name
	}
	last := parts[len(parts)-1]
	if looksLikeWxH(last) {
		return strings.Join(parts[:len(parts)-1], "-") + ext
	}
	return name
}

func looksLikeWxH(s string) bool {
	segs := strings.Split(s, "x")
	if len(segs) != 2 {
		return false
	}
	return isDigits(segs[0]) && isDigits(segs[1])
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func esc(s string) string {
	return html.EscapeString(s)
}

// writeReviewContextHTML adds source-file + sample page links so reviewers can open the live context.
// Caps visible page links (sitewide assets can appear on hundreds of URLs).
func writeReviewContextHTML(b *strings.Builder, sourceURL string, pages []string) {
	const maxPages = 5
	b.WriteString(`<p class="ctx">`)
	if strings.TrimSpace(sourceURL) != "" {
		b.WriteString(`File: <a href="`)
		b.WriteString(esc(sourceURL))
		b.WriteString(`" target="_blank" rel="noopener">`)
		b.WriteString(esc(sourceURL))
		b.WriteString(`</a>`)
	}
	ordered := prioritizeReviewPages(pages)
	if len(ordered) == 0 {
		b.WriteString(`</p>`)
		return
	}
	if strings.TrimSpace(sourceURL) != "" {
		b.WriteString(`<br>`)
	}
	b.WriteString(`Seen on `)
	b.WriteString(fmt.Sprintf("%d", len(ordered)))
	if len(ordered) == 1 {
		b.WriteString(` page: `)
	} else {
		b.WriteString(` pages: `)
	}
	show := ordered
	if len(show) > maxPages {
		show = ordered[:maxPages]
	}
	for i, p := range show {
		if i > 0 {
			b.WriteString(` · `)
		}
		b.WriteString(`<a href="`)
		b.WriteString(esc(p))
		b.WriteString(`" target="_blank" rel="noopener">`)
		b.WriteString(esc(shortPageLabel(p)))
		b.WriteString(`</a>`)
	}
	if extra := len(ordered) - len(show); extra > 0 {
		b.WriteString(` <span class="more">(+`)
		b.WriteString(fmt.Sprintf("%d", extra))
		b.WriteString(` more)</span>`)
	}
	b.WriteString(`</p>`)
}

func prioritizeReviewPages(pages []string) []string {
	seen := make(map[string]struct{}, len(pages))
	out := make([]string, 0, len(pages))
	var home []string
	var rest []string
	for _, p := range pages {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if isLikelyHomeURL(p) {
			home = append(home, p)
		} else {
			rest = append(rest, p)
		}
	}
	out = append(out, home...)
	out = append(out, rest...)
	return out
}

func isLikelyHomeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.Trim(u.Path, "/") == ""
}

func shortPageLabel(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return "/"
	}
	if len(p) > 64 {
		return p[:61] + "…"
	}
	return "/" + p
}
