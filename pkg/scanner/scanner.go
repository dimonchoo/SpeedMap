package scanner

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"SpeedMap/pkg/config"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type Scanner struct {
	cfg        config.ScanConfig
	ctx        context.Context
	cancel     context.CancelFunc
	isCanceled int32
}

func NewScanner(cfg config.ScanConfig) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scanner{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scanner) Cancel() {
	atomic.StoreInt32(&s.isCanceled, 1)
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scanner) IsCanceled() bool {
	return atomic.LoadInt32(&s.isCanceled) == 1
}

func (s *Scanner) ScanURLs(urls []string, onProgress func(progress ScanProgress)) ([]PageResult, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs to scan")
	}

	concurrency := s.cfg.NormalizedConcurrency()
	total := len(urls)

	results := make([]PageResult, total)
	urlChan := make(chan struct {
		index int
		url   string
	}, total)

	for i, u := range urls {
		urlChan <- struct {
			index int
			url   string
		}{index: i, url: u}
	}
	close(urlChan)

	// Chromedp Allocator Options - SHARED browser process with clean teardown flags
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(s.ctx, opts...)
	defer allocCancel() // Guarantees Chrome process tree termination upon function exit

	// Master Browser Context: keeps a SINGLE Chrome browser process running throughout the entire scan session.
	// Worker goroutines open lightweight tabs (CDP Targets) inside this single Chrome instance instead of relaunching Chrome for every URL.
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(browserCtx); err != nil {
		return nil, fmt.Errorf("failed to start master Chrome instance: %w", err)
	}
	defer browserCancel() // Gracefully shuts down Chrome process tree when scan batch completes

	var wg sync.WaitGroup
	var processedCount int32

	// Launch continuous worker pipeline
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range urlChan {
				if s.IsCanceled() {
					return
				}

				// Continuously scan URL in a lightweight tab target inside the single master Chrome browser
				res := s.scanWithContext(browserCtx, item.index+1, item.url, false)
				results[item.index] = res

				currentCount := atomic.AddInt32(&processedCount, 1)

				// Real-time progress update immediately per completed item
				if onProgress != nil {
					onProgress(ScanProgress{
						TotalProcessed: int(currentCount),
						TotalURLs:      total,
						CurrentURL:     item.url,
						LatestResult:   &res,
						IsFinished:     int(currentCount) == total || s.IsCanceled(),
					})
				}
			}
		}()
	}

	wg.Wait()
	return results, nil
}


// ScanSingleURL executes an on-demand single URL scan with immediate Chrome teardown.
// Cache is disabled so re-checks after site edits see fresh HTML (not Chrome/HTTP cache).
func (s *Scanner) ScanSingleURL(id int, rawURL string) PageResult {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disk-cache-size", "0"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(s.ctx, opts...)
	defer allocCancel()

	return s.scanWithContext(allocCtx, id, rawURL, true)
}

func (s *Scanner) scanWithContext(allocCtx context.Context, id int, rawURL string, bypassCache bool) PageResult {
	startTime := time.Now()
	res := PageResult{
		ID:  id,
		URL: rawURL,
	}

	if s.IsCanceled() {
		res.OverallStatus = "error"
		res.Error = "Сканування скасовано користувачем"
		return res
	}

	// Create lightweight tab context in shared Chrome process
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	timeoutSec := s.cfg.NormalizedTimeout()
	timeoutCtx, timeCancel := context.WithTimeout(taskCtx, time.Duration(timeoutSec)*time.Second)
	defer timeCancel()

	// Capture Response Status Code & Cache Status via CDP event listener
	var statusCode int = 200
	var pluginCacheStatus string = "NONE"
	var cfCacheStatus string = "NONE"
	var cfPop string = ""
	var cfRay string = ""

	chromedp.ListenTarget(timeoutCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Response != nil && (e.Type == network.ResourceTypeDocument || e.Response.URL == rawURL || e.Response.URL == rawURL+"/") {
				statusCode = int(e.Response.Status)
				if e.Response.Headers != nil {
					for k, v := range e.Response.Headers {
						lk := strings.ToLower(k)
						val := strings.ToUpper(fmt.Sprintf("%v", v))
						rawVal := fmt.Sprintf("%v", v)

						// 1. Plugin / Redis Cache Header Detection
						if status, ok := parsePluginCacheHeader(lk, val); ok {
							pluginCacheStatus = status
						}

						// 2. Cloudflare Cache Header Detection
						if lk == "cf-cache-status" {
							cfCacheStatus = val
						}

						// 3. Cloudflare Ray and Data Center (PoP) Detection
						if lk == "cf-ray" {
							cfRay = rawVal
							parts := strings.Split(rawVal, "-")
							if len(parts) > 1 {
								cfPop = strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
							}
						}
					}
				}
			}
		}
	})

	// Prepare Headers
	headers := make(network.Headers)
	if s.cfg.AuthUser != "" || s.cfg.AuthPass != "" {
		auth := s.cfg.AuthUser + ":" + s.cfg.AuthPass
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	}
	for _, h := range s.cfg.Headers {
		if strings.TrimSpace(h.Key) != "" {
			headers[h.Key] = h.Value
		}
	}
	if bypassCache {
		headers["Cache-Control"] = "no-cache"
		headers["Pragma"] = "no-cache"
	}

	// Build Tasks
	var vitals WebVitals
	var diagnostics PageDiagnostics
	var tasks chromedp.Tasks

	// Enable Network & Extra Headers
	tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := network.Enable().Do(ctx); err != nil {
			return err
		}
		if bypassCache {
			// ponytail: Chrome HTTP cache can hide site edits after rescan; disable for single-URL path.
			if err := network.SetCacheDisabled(true).Do(ctx); err != nil {
				return err
			}
		}
		if len(headers) > 0 {
			if err := network.SetExtraHTTPHeaders(headers).Do(ctx); err != nil {
				return err
			}
		}
		return nil
	}))

	// Device Emulation (Mobile vs Desktop)
	if s.cfg.IsMobile {
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			// 1. Mobile Viewport (375x812, DPR 3.0, mobile screen orientation)
			if err := emulation.SetDeviceMetricsOverride(375, 812, 3.0, true).Do(ctx); err != nil {
				return err
			}
			_ = emulation.SetTouchEmulationEnabled(true).Do(ctx)

			// 2. Mobile User Agent
			mobileUA := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 SpeedMap/1.0"
			if err := emulation.SetUserAgentOverride(mobileUA).Do(ctx); err != nil {
				return err
			}

			// 3. Google Lighthouse Simulated 4G Throttling (1.638 Mbps download, 150ms RTT latency)
			cond := &network.Conditions{
				URLPattern:         "", // match all URLs
				Latency:            150,
				DownloadThroughput: 204800, // 1.6384 Mbps = 204.8 KB/s
				UploadThroughput:   93750,  // 750 Kbps
				ConnectionType:     network.ConnectionTypeCellular4g,
			}
			_, _ = network.EmulateNetworkConditionsByRule([]*network.Conditions{cond}).Do(ctx)

			// 4. CPU Throttling: 4x slowdown emulating mid-tier mobile hardware
			_ = emulation.SetCPUThrottlingRate(4.0).Do(ctx)

			return nil
		}))
	} else {
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			if err := emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false).Do(ctx); err != nil {
				return err
			}
			_ = emulation.SetTouchEmulationEnabled(false).Do(ctx)
			_ = emulation.SetCPUThrottlingRate(1.0).Do(ctx)

			desktopUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 SpeedMap/1.0"
			return emulation.SetUserAgentOverride(desktopUA).Do(ctx)
		}))
	}

	// Fast Navigation: Initiate page navigation & poll for DOM readiness
	tasks = append(tasks,
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, _, _, _, err := page.Navigate(rawURL).Do(ctx)
			return err
		}),
		chromedp.Poll(`document.readyState === "interactive" || document.readyState === "complete"`, nil, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			evalJS := fmt.Sprintf(WebVitalsCollectorJS, s.cfg.AutoScroll)
			resObj, expObj, err := runtime.Evaluate(evalJS).
				WithAwaitPromise(true).
				WithReturnByValue(true).
				Do(ctx)

			if err != nil {
				return fmt.Errorf("CDP evaluation error: %w", err)
			}
			if expObj != nil {
				return fmt.Errorf("JS evaluation exception: %s", expObj.Text)
			}
			if resObj == nil || resObj.Value == nil {
				return fmt.Errorf("JS evaluation returned empty result")
			}

			var evalRes ScanEvalResult
			if err := json.Unmarshal(resObj.Value, &evalRes); err != nil {
				return fmt.Errorf("failed to parse WebVitals: %w", err)
			}
			vitals = evalRes.Metrics
			diagnostics = evalRes.Diagnostics

			// Dual-Viewport Safety: if in mobile mode, quickly expand viewport to 1920x1080
			// without reloading the page or re-running full metrics to capture max desktop layout sizes.
			if s.cfg.IsMobile && len(diagnostics.LargestImages) > 0 {
				if err := emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false).Do(ctx); err == nil {
					desktopJS := `(() => {
						const map = {};
						document.querySelectorAll('img').forEach(img => {
							const src = img.currentSrc || img.src || img.getAttribute('data-src') || img.getAttribute('data-lazy-src');
							if (src && !src.startsWith('data:')) {
								try {
									const fullUrl = new URL(src, document.baseURI).href;
									const rect = img.getBoundingClientRect();
									const w = Math.round(rect.width || img.clientWidth || img.offsetWidth || 0);
									const h = Math.round(rect.height || img.clientHeight || img.offsetHeight || 0);
									if (w > 0 || h > 0) {
										map[fullUrl] = { width: w, height: h };
									}
								} catch(e) {}
							}
						});
						return map;
					})()`
					if dtObj, _, err := runtime.Evaluate(desktopJS).WithReturnByValue(true).Do(ctx); err == nil && dtObj != nil && dtObj.Value != nil {
						var dtMap map[string]struct {
							Width  int `json:"width"`
							Height int `json:"height"`
						}
						if err := json.Unmarshal(dtObj.Value, &dtMap); err == nil && len(dtMap) > 0 {
							for i := range diagnostics.LargestImages {
								u := diagnostics.LargestImages[i].URL
								if dt, ok := dtMap[u]; ok {
									if dt.Width > diagnostics.LargestImages[i].RenderedWidth {
										diagnostics.LargestImages[i].RenderedWidth = dt.Width
									}
									if dt.Height > diagnostics.LargestImages[i].RenderedHeight {
										diagnostics.LargestImages[i].RenderedHeight = dt.Height
									}
								}
							}
						}
					}
				}
			}
			return nil
		}),
	)

	// Execute Tasks
	if err := chromedp.Run(timeoutCtx, tasks); err != nil {
		res.DurationMs = time.Since(startTime).Milliseconds()
		res.OverallStatus = "error"
		res.StatusCode = statusCode

		// Friendly Ukrainian error translation
		errText := err.Error()
		if strings.Contains(errText, "context deadline exceeded") {
			res.Error = fmt.Sprintf("Перевищено таймаут завантаження сторінки (%ds)", timeoutSec)
		} else {
			res.Error = fmt.Sprintf("Помилка завантаження сторінки: %v", err)
		}
		return res
	}

	res.DurationMs = time.Since(startTime).Milliseconds()
	res.StatusCode = statusCode
	res.PluginCacheStatus = pluginCacheStatus
	res.CloudflareCacheStatus = cfCacheStatus
	res.CloudflarePop = cfPop
	res.CloudflareRay = cfRay
	res.Metrics = vitals

	// Calculate Grades
	grades := DetailedGrades{
		TTFB: CalculateGrade("TTFB", vitals.TTFB),
		FCP:  CalculateGrade("FCP", vitals.FCP),
		LCP:  CalculateGrade("LCP", vitals.LCP),
		CLS:  CalculateGrade("CLS", vitals.CLS),
		TBT:  CalculateGrade("TBT", vitals.TBT),
	}
	res.Grades = grades
	diagnostics.Categories = BuildCategoryDiagnostics(vitals, grades, diagnostics)
	res.Diagnostics = diagnostics
	res.OverallStatus = CalculateOverallStatus(grades)
	res.Recommendations = GenerateRecommendations(vitals, grades, diagnostics)

	return res
}

// parsePluginCacheHeader maps known plugin/Redis cache response headers to a status string.
// Legacy header x-lightweight-cache: yes means HIT.
func parsePluginCacheHeader(headerName, rawValue string) (string, bool) {
	lk := strings.ToLower(strings.TrimSpace(headerName))
	val := strings.ToUpper(strings.TrimSpace(rawValue))
	if val == "" {
		return "", false
	}

	switch lk {
	case "x-lightweight-cache":
		// Old Lightweight Redis Cache variant: "yes" = served from cache
		if val == "YES" {
			return "HIT", true
		}
		if val == "NO" {
			return "MISS", true
		}
		return val, true
	case "x-lightweight-cache-status", "x-redis-cache", "x-cache-status", "x-cache", "x-page-cache":
		return val, true
	default:
		return "", false
	}
}
