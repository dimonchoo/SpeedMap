package wpexport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"
)

// ManifestVerifyRequest holds arguments for verifying an exported package/manifest against a live target site.
type ManifestVerifyRequest struct {
	ManifestPathOrDir string `json:"manifestPath"`
	TargetURL         string `json:"targetUrl"`         // e.g. "https://uat.infuse.com"
	CheckPages        bool   `json:"checkPages"`        // check HTML on pages for replacement
	MaxPagesPerImage  int    `json:"maxPagesPerImage"`   // e.g. 2
	Concurrency       int    `json:"concurrency"`       // e.g. 10
	TimeoutSec        int    `json:"timeoutSec"`        // e.g. 15
	MaxImages         int    `json:"maxImages"`         // 0 for all
}

// ImageVerifyResult holds the verification result for a single image in the manifest.
type ImageVerifyResult struct {
	ID               string   `json:"id"`
	Basename         string   `json:"basename"`
	OriginalURL      string   `json:"originalUrl"`
	TargetWebPURL    string   `json:"targetWebpUrl"`
	TargetOldURL     string   `json:"targetOldUrl"`
	HTTPStatus       int      `json:"httpStatus"`
	ContentType      string   `json:"contentType"`
	ContentLength    int64    `json:"contentLength"`
	Width            int      `json:"width,omitempty"`
	Height           int      `json:"height,omitempty"`
	ExpectedWidth    int      `json:"expectedWidth,omitempty"`
	ExpectedHeight   int      `json:"expectedHeight,omitempty"`
	DimensionsMatch  bool     `json:"dimensionsMatch"`
	Status           string   `json:"status"` // "pass", "warn", "fail"
	PagesTotal       int      `json:"pagesTotal"`
	PagesChecked     int      `json:"pagesChecked"`
	PagesReplaced    int      `json:"pagesReplaced"`
	PagesOldFound    int      `json:"pagesOldFound"`
	SamplePageURL    string   `json:"samplePageUrl,omitempty"`
	SamplePageStatus string   `json:"samplePageStatus,omitempty"` // "replaced", "old_found", "both", "neither", "fetch_error"
	Errors           []string `json:"errors,omitempty"`
}

// ManifestVerifySummary holds high-level verification metrics.
type ManifestVerifySummary struct {
	TotalImages        int    `json:"totalImages"`
	PassedImages       int    `json:"passedImages"`
	WarnedImages       int    `json:"warnedImages"`
	FailedImages       int    `json:"failedImages"`
	TotalPagesChecked  int    `json:"totalPagesChecked"`
	PagesWithWebP      int    `json:"pagesWithWebp"`
	PagesWithOldRaster int    `json:"pagesWithOldRaster"`
	DurationMs         int64  `json:"durationMs"`
	TargetDomain       string `json:"targetDomain"`
	ManifestCount      int    `json:"manifestCount"`
	AllPassed          bool   `json:"allPassed"`
}

// ManifestVerifyProgress represents real-time progress for Wails events or streaming.
type ManifestVerifyProgress struct {
	Phase       string `json:"phase"` // "verifying", "done"
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	CurrentItem string `json:"currentItem"`
	PassedCount int    `json:"passedCount"`
	WarnCount   int    `json:"warnCount"`
	FailCount   int    `json:"failCount"`
}

// ManifestVerifyResponse contains the final verification results.
type ManifestVerifyResponse struct {
	Summary ManifestVerifySummary `json:"summary"`
	Items   []ImageVerifyResult   `json:"items"`
}

var (
	extRegex = regexp.MustCompile(`\.(jpe?g|png|gif|webp|svg)$`)
)

// VerifyManifest verifies that all assets in manifest.json are reachable as WebP on targetURL,
// and checks if referencing pages have been updated to WebP.
func VerifyManifest(ctx context.Context, req ManifestVerifyRequest, progressCb func(ManifestVerifyProgress)) (*ManifestVerifyResponse, error) {
	startTime := time.Now()

	packageCtx, err := LoadPackageForStudio(req.ManifestPathOrDir)
	if err != nil {
		return nil, fmt.Errorf("помилка завантаження маніфесту: %w", err)
	}

	targetBase := strings.TrimRight(strings.TrimSpace(req.TargetURL), "/")
	if targetBase == "" {
		if packageCtx.Domain != "" {
			targetBase = strings.TrimRight(packageCtx.Domain, "/")
			if !strings.HasPrefix(targetBase, "http://") && !strings.HasPrefix(targetBase, "https://") {
				targetBase = "https://" + targetBase
			}
		} else {
			return nil, fmt.Errorf("не вказано цільовий сайт (TargetURL) для перевірки")
		}
	}

	targetParsed, err := url.Parse(targetBase)
	if err != nil {
		return nil, fmt.Errorf("невалідний URL цільового сайту: %w", err)
	}
	targetDomain := targetParsed.Host
	targetScheme := targetParsed.Scheme

	concurrency := req.Concurrency
	if concurrency <= 0 || concurrency > 25 {
		concurrency = 8
	}

	timeout := req.TimeoutSec
	if timeout <= 0 {
		timeout = 15
	}

	maxPagesPerImage := req.MaxPagesPerImage
	if maxPagesPerImage <= 0 {
		maxPagesPerImage = 2
	}

	images := packageCtx.Images
	if req.MaxImages > 0 && len(images) > req.MaxImages {
		images = images[:req.MaxImages]
	}
	totalImages := len(images)

	// Shared HTTP client with connection pooling and insecure TLS skip for staging environments
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeout) * time.Second,
	}

	// HTML page cache to prevent redundant HTTP requests for the same page
	var pageCacheMu sync.RWMutex
	pageCache := make(map[string]string) // pageURL -> HTML content

	fetchPageHTML := func(pageURL string) (string, error) {
		pageCacheMu.RLock()
		content, ok := pageCache[pageURL]
		pageCacheMu.RUnlock()
		if ok {
			return content, nil
		}

		req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "SpeedMap-Verifier/1.0 (+https://speedmap.io)")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024)) // limit to 5MB
		if err != nil {
			return "", err
		}

		bodyStr := string(bodyBytes)
		pageCacheMu.Lock()
		pageCache[pageURL] = bodyStr
		pageCacheMu.Unlock()
		return bodyStr, nil
	}

	results := make([]ImageVerifyResult, totalImages)
	var passedCount, warnCount, failCount int
	var countMu sync.Mutex

	jobs := make(chan int, totalImages)
	for i := 0; i < totalImages; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				img := images[idx]
				res := verifySingleImage(ctx, client, img, targetScheme, targetDomain, req.CheckPages, maxPagesPerImage, fetchPageHTML)
				results[idx] = res

				countMu.Lock()
				switch res.Status {
				case "pass":
					passedCount++
				case "warn":
					warnCount++
				default:
					failCount++
				}
				curPassed := passedCount
				curWarn := warnCount
				curFail := failCount
				countMu.Unlock()

				if progressCb != nil {
					progressCb(ManifestVerifyProgress{
						Phase:       "verifying",
						Current:     idx + 1,
						Total:       totalImages,
						CurrentItem: img.Basename,
						PassedCount: curPassed,
						WarnCount:   curWarn,
						FailCount:   curFail,
					})
				}
			}
		}()
	}

	wg.Wait()

	// Compute summary
	totalPagesChecked := 0
	pagesWithWebP := 0
	pagesWithOldRaster := 0
	for _, r := range results {
		totalPagesChecked += r.PagesChecked
		if r.PagesReplaced > 0 {
			pagesWithWebP++
		}
		if r.PagesOldFound > 0 {
			pagesWithOldRaster++
		}
	}

	allPassed := (failCount == 0 && warnCount == 0)

	summary := ManifestVerifySummary{
		TotalImages:        totalImages,
		PassedImages:       passedCount,
		WarnedImages:       warnCount,
		FailedImages:       failCount,
		TotalPagesChecked:  len(pageCache),
		PagesWithWebP:      pagesWithWebP,
		PagesWithOldRaster: pagesWithOldRaster,
		DurationMs:         time.Since(startTime).Milliseconds(),
		TargetDomain:       targetDomain,
		ManifestCount:      packageCtx.Count,
		AllPassed:          allPassed,
	}

	if progressCb != nil {
		progressCb(ManifestVerifyProgress{
			Phase:       "done",
			Current:     totalImages,
			Total:       totalImages,
			CurrentItem: "Готово",
			PassedCount: passedCount,
			WarnCount:   warnCount,
			FailCount:   failCount,
		})
	}

	return &ManifestVerifyResponse{
		Summary: summary,
		Items:   results,
	}, nil
}

func verifySingleImage(
	ctx context.Context,
	client *http.Client,
	img StudioPackageImageItem,
	targetScheme string,
	targetDomain string,
	checkPages bool,
	maxPages int,
	fetchPageHTML func(string) (string, error),
) ImageVerifyResult {
	res := ImageVerifyResult{
		ID:             img.ID,
		Basename:       img.Basename,
		OriginalURL:    img.SourceURL,
		ExpectedWidth:  img.OptimizedWidth,
		ExpectedHeight: img.OptimizedHeight,
		PagesTotal:     len(img.Pages),
		Status:         "pass",
	}

	// Compute target URLs on the target domain
	res.TargetWebPURL = mapURLToTargetDomain(img.SourceURL, targetScheme, targetDomain, img.Format, true)
	res.TargetOldURL = mapURLToTargetDomain(img.SourceURL, targetScheme, targetDomain, img.Format, false)

	// 1. Check direct HTTP availability of the WebP asset
	httpReq, err := http.NewRequestWithContext(ctx, "GET", res.TargetWebPURL, nil)
	if err != nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, fmt.Sprintf("помилка формування запиту: %v", err))
		return res
	}
	httpReq.Header.Set("User-Agent", "SpeedMap-Verifier/1.0")

	resp, err := client.Do(httpReq)
	if err != nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, fmt.Sprintf("HTTP запит не вдався: %v", err))
		return res
	}
	defer resp.Body.Close()

	res.HTTPStatus = resp.StatusCode
	res.ContentType = resp.Header.Get("Content-Type")

	if resp.StatusCode != http.StatusOK {
		res.Status = "fail"
		res.Errors = append(res.Errors, fmt.Sprintf("WebP повертає HTTP %d замість 200", resp.StatusCode))
		return res
	}

	// Read image bytes (up to 2MB) to verify dimensions
	imgBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		res.Status = "warn"
		res.Errors = append(res.Errors, fmt.Sprintf("не вдалося прочитати тіло відповіді: %v", err))
	} else {
		res.ContentLength = int64(len(imgBytes))
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(imgBytes)); err == nil {
			res.Width = cfg.Width
			res.Height = cfg.Height

			if res.ExpectedWidth > 0 && res.ExpectedHeight > 0 {
				if res.Width == res.ExpectedWidth && res.Height == res.ExpectedHeight {
					res.DimensionsMatch = true
				} else {
					res.DimensionsMatch = false
					res.Status = "warn"
					res.Errors = append(res.Errors, fmt.Sprintf("розмір відрізняється: отримано %dx%d, очікувалось %dx%d", res.Width, res.Height, res.ExpectedWidth, res.ExpectedHeight))
				}
			} else {
				res.DimensionsMatch = true
			}
		}
	}

	// 2. Check HTML of referencing pages (if enabled)
	if checkPages && len(img.Pages) > 0 {
		pagesToCheck := img.Pages
		if len(pagesToCheck) > maxPages {
			pagesToCheck = pagesToCheck[:maxPages]
		}

		oldBasename := filepath.Base(img.SourceURL)
		webpBasename := filepath.Base(res.TargetWebPURL)

		for _, rawPageURL := range pagesToCheck {
			targetPageURL := mapPageURLToTargetDomain(rawPageURL, targetScheme, targetDomain)
			if res.SamplePageURL == "" {
				res.SamplePageURL = targetPageURL
			}

			html, err := fetchPageHTML(targetPageURL)
			if err != nil {
				res.SamplePageStatus = "fetch_error"
				res.Errors = append(res.Errors, fmt.Sprintf("помилка завантаження сторінки %s: %v", targetPageURL, err))
				continue
			}

			res.PagesChecked++
			hasWebP := strings.Contains(html, webpBasename)
			hasOld := strings.Contains(html, oldBasename)

			if hasWebP {
				res.PagesReplaced++
			}
			if hasOld {
				res.PagesOldFound++
			}

			if res.SamplePageStatus == "" {
				if hasWebP && !hasOld {
					res.SamplePageStatus = "replaced"
				} else if hasWebP && hasOld {
					res.SamplePageStatus = "both"
				} else if !hasWebP && hasOld {
					res.SamplePageStatus = "old_found"
				} else {
					res.SamplePageStatus = "neither"
				}
			}
		}

		if res.PagesChecked > 0 {
			if res.PagesOldFound > 0 && res.PagesReplaced == 0 {
				res.Status = "warn"
				res.Errors = append(res.Errors, fmt.Sprintf("на сторінках знайдено старий файл (%s), але відсутнє посилання на WebP", oldBasename))
			}
		}
	}

	return res
}

func mapURLToTargetDomain(sourceURL string, targetScheme string, targetDomain string, format string, toWebP bool) string {
	u, err := url.Parse(sourceURL)
	if err != nil {
		return sourceURL
	}

	p := u.Path
	if toWebP {
		optExt := ".webp"
		if strings.ToLower(format) == "svg" {
			optExt = ".svg"
		}
		p = extRegex.ReplaceAllString(p, optExt)
	}

	return fmt.Sprintf("%s://%s%s", targetScheme, targetDomain, p)
}

func mapPageURLToTargetDomain(pageURL string, targetScheme string, targetDomain string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return pageURL
	}
	return fmt.Sprintf("%s://%s%s", targetScheme, targetDomain, u.Path)
}
