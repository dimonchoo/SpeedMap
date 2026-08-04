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

	// Pre-warm Chrome browser process
	warmCtx, warmCancel := chromedp.NewContext(allocCtx)
	_ = chromedp.Run(warmCtx)
	warmCancel()

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

				// Continuously scan URL in a lightweight tab context
				res := s.scanWithContext(allocCtx, item.index+1, item.url)
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

// ScanSingleURL executes an on-demand single URL scan with immediate Chrome teardown
func (s *Scanner) ScanSingleURL(id int, rawURL string) PageResult {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.IgnoreCertErrors,
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-extensions", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(s.ctx, opts...)
	defer allocCancel()

	return s.scanWithContext(allocCtx, id, rawURL)
}

func (s *Scanner) scanWithContext(allocCtx context.Context, id int, rawURL string) PageResult {
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

	// Capture Response Status Code via CDP event listener
	var statusCode int = 200
	chromedp.ListenTarget(timeoutCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventResponseReceived:
			if e.Response != nil && (e.Response.URL == rawURL || e.Response.URL == rawURL+"/") {
				statusCode = int(e.Response.Status)
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

	// Build Tasks
	var vitals WebVitals
	var diagnostics PageDiagnostics
	var tasks chromedp.Tasks

	// Enable Network & Extra Headers
	tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := network.Enable().Do(ctx); err != nil {
			return err
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
			err := emulation.SetDeviceMetricsOverride(375, 812, 3.0, true).Do(ctx)
			if err != nil {
				return err
			}
			mobileUA := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 SpeedMap/1.0"
			return emulation.SetUserAgentOverride(mobileUA).Do(ctx)
		}))
	} else {
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			err := emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false).Do(ctx)
			if err != nil {
				return err
			}
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
			resObj, expObj, err := runtime.Evaluate(WebVitalsCollectorJS).
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
