package analytics

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
)

type ResourceImpact struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Occurrences     int     `json:"occurrences"`
	AvgDurationMs   float64 `json:"avgDurationMs"`
	TotalDurationMs float64 `json:"totalDurationMs"`
}

type AggregatedImage struct {
	URL                       string   `json:"url"`
	MaxTransferSize           int64    `json:"maxTransferSize"` // bytes
	FormattedSize             string   `json:"formattedSize"`   // e.g. "1.2 MB"
	AvgDurationMs             float64  `json:"avgDurationMs"`   // ms
	PageCount                 int      `json:"pageCount"` // how many pages use this image
	Pages                     []string `json:"pages"`     // page URLs where this image was seen
	Width                     int      `json:"width"`     // intrinsic / natural width
	Height                    int      `json:"height"`    // intrinsic / natural height
	NaturalWidth              int      `json:"naturalWidth"`
	NaturalHeight             int      `json:"naturalHeight"`
	MaxRenderedWidth          int      `json:"maxRenderedWidth"`        // max CSS display width across all pages
	MaxRenderedHeight         int      `json:"maxRenderedHeight"`       // max CSS display height across all pages
	RecommendedRetinaWidth    int      `json:"recommendedRetinaWidth"`  // MaxRenderedWidth * 2 (optimal for Retina)
	RecommendedRetinaHeight   int      `json:"recommendedRetinaHeight"` // MaxRenderedHeight * 2
	IsOversized               bool     `json:"isOversized"`             // NaturalWidth > MaxRenderedWidth * 2
	Format                    string   `json:"format"`                  // png, jpg, webp, svg, avif, gif
	IsHeavy                   bool     `json:"isHeavy"`                 // > threshold
	IsLazy                    bool     `json:"isLazy"`
	IsLCP                     bool     `json:"isLCP"`
	EstimatedWebPSize         int64    `json:"estimatedWebPSize"`
	EstimatedWebPFormatted    string   `json:"estimatedWebPFormatted"`
	EstimatedSavingsBytes     int64    `json:"estimatedSavingsBytes"`
	EstimatedSavingsFormatted string   `json:"estimatedSavingsFormatted"`
	EstimatedSavingsPercent   float64  `json:"estimatedSavingsPercent"`
}

type AggregatedFont struct {
	Family        string   `json:"family"`
	URL           string   `json:"url"`
	Type          string   `json:"type"`
	Occurrences   int      `json:"occurrences"`   // how many pages use this font
	Percentage    float64  `json:"percentage"`    // % of total scanned pages
	AvgDurationMs float64  `json:"avgDurationMs"` // ms
	TransferSize  int64    `json:"transferSize"`
	FormattedSize string   `json:"formattedSize"`
	PageURLs      []string `json:"pageUrls"`
}

// AggregatedIframe groups the same iframe src across scanned pages.
// MissedCount = pages where iframe was in DOM but did not load during the scan window.
type AggregatedIframe struct {
	Src             string   `json:"src"`
	Title           string   `json:"title"`
	PageCount       int      `json:"pageCount"`
	Pages           []string `json:"pages"`
	Occurrences     int      `json:"occurrences"`
	LoadedCount     int      `json:"loadedCount"`
	MissedCount     int      `json:"missedCount"`
	IsLazy          bool     `json:"isLazy"`
	AvgDurationMs   float64  `json:"avgDurationMs"`
	MaxTransferSize int64    `json:"maxTransferSize"`
	FormattedSize   string   `json:"formattedSize"`
	Width           int      `json:"width"`
	Height          int      `json:"height"`
}

// AggregatedForm groups unique forms across scanned pages
type AggregatedForm struct {
	ID               string                    `json:"id"`
	Title            string                    `json:"title"`
	Engine           string                    `json:"engine"`
	Method           string                    `json:"method"`
	Action           string                    `json:"action"`
	PageCount        int                       `json:"pageCount"`
	Pages            []string                  `json:"pages"`
	Fields           []scanner.FormFieldDetail `json:"fields"`
	FieldCount       int                       `json:"fieldCount"`
	HasFileUpload    bool                      `json:"hasFileUpload"`
	AllowedFileTypes string                    `json:"allowedFileTypes,omitempty"`
	Captcha          scanner.CaptchaDetail     `json:"captcha"`
	HiddenTokens     map[string]string         `json:"hiddenTokens,omitempty"`
}

type SiteAnalytics struct {
	TotalPages  scannedCount `json:"totalPages"`
	HealthScore int          `json:"healthScore"` // 0 - 100

	StatusCounts map[string]int `json:"statusCounts"` // good, needs-improvement, poor, error

	AverageMetrics map[string]float64 `json:"averageMetrics"` // TTFB, FCP, LCP, CLS, TBT

	TopResourceBottlenecks []ResourceImpact   `json:"topResourceBottlenecks"`
	LargestImages          []AggregatedImage  `json:"largestImages"`
	AllImages              []AggregatedImage  `json:"allImages"`
	FontUsage              []AggregatedFont   `json:"fontUsage"`
	Iframes                []AggregatedIframe `json:"iframes"`
	Forms                  []AggregatedForm   `json:"forms"`
	GlobalFixes            []string           `json:"globalFixes"`

	// Image Optimization Analytics (SEOAEO-235)
	TotalImagePayloadBytes     int64          `json:"totalImagePayloadBytes"`
	TotalImagePayloadFormatted string         `json:"totalImagePayloadFormatted"`
	TotalImageCount            int            `json:"totalImageCount"`
	HeavyImagesCount           int            `json:"heavyImagesCount"`
	OversizedImagesCount       int            `json:"oversizedImagesCount"`
	NonWebPCount               int            `json:"nonWebPCount"`
	MissingLazyCount           int            `json:"missingLazyCount"`
	TotalWebPSavingsBytes      int64          `json:"totalWebPSavingsBytes"`
	TotalWebPSavingsFormatted  string         `json:"totalWebPSavingsFormatted"`
	FormatBreakdown            map[string]int `json:"formatBreakdown"`

	// Iframe audit
	TotalIframeCount  int `json:"totalIframeCount"`
	MissedIframeCount int `json:"missedIframeCount"`
	LoadedIframeCount int `json:"loadedIframeCount"`

	// Form audit
	TotalFormsCount       int            `json:"totalFormsCount"`
	PagesWithFormsCount   int            `json:"pagesWithFormsCount"`
	CaptchaProtectedCount int            `json:"captchaProtectedCount"`
	UnprotectedFormsCount int            `json:"unprotectedFormsCount"`
	FileUploadFormsCount  int            `json:"fileUploadFormsCount"`
	FormEngineBreakdown   map[string]int `json:"formEngineBreakdown"`
}

type scannedCount = int

func EstimateWebPSize(format string, originalBytes int64, cfg ...config.ScanConfig) int64 {
	if originalBytes <= 0 {
		return 0
	}
	f := strings.ToLower(format)

	pngRatio := 0.30
	jpgRatio := 0.60
	gifRatio := 0.50

	if len(cfg) > 0 {
		pngRatio = cfg[0].NormalizedPngRatio()
		jpgRatio = cfg[0].NormalizedJpgRatio()
		gifRatio = cfg[0].NormalizedGifRatio()
	}

	switch f {
	case "png":
		return int64(float64(originalBytes) * pngRatio)
	case "jpg", "jpeg":
		return int64(float64(originalBytes) * jpgRatio)
	case "gif":
		return int64(float64(originalBytes) * gifRatio)
	default:
		return originalBytes
	}
}

func ComputeSiteAnalytics(results []scanner.PageResult, cfg ...interface{}) SiteAnalytics {
	thresholdBytes := int64(100 * 1024)
	var activeConfig *config.ScanConfig

	if len(cfg) > 0 {
		switch v := cfg[0].(type) {
		case config.ScanConfig:
			activeConfig = &v
			thresholdBytes = v.NormalizedHeavyThresholdBytes()
		case *config.ScanConfig:
			if v != nil {
				activeConfig = v
				thresholdBytes = v.NormalizedHeavyThresholdBytes()
			}
		case int:
			if v > 0 {
				thresholdBytes = int64(v) * 1024
			}
		}
	}

	total := len(results)
	if total == 0 {
		return SiteAnalytics{
			StatusCounts:    make(map[string]int),
			AverageMetrics:  make(map[string]float64),
			FormatBreakdown: make(map[string]int),
		}
	}

	statusCounts := map[string]int{
		"good text-emerald-400": 0,
		"good":                  0,
		"needs-improvement":    0,
		"poor":                 0,
		"error":                0,
	}

	var sumTTFB, sumFCP, sumLCP, sumCLS, sumTBT float64
	var countTTFB, countFCP, countLCP, countCLS, countTBT int

	resourceMap := make(map[string]*ResourceImpact)
	imageMap := make(map[string]*AggregatedImage)
	fontMap := make(map[string]*AggregatedFont)
	iframeMap := make(map[string]*AggregatedIframe)
	var totalMissedIframes, totalLoadedIframes int

	for _, p := range results {
		statusCounts[p.OverallStatus]++

		if p.OverallStatus != "error" {
			if p.Metrics.TTFB > 0 {
				sumTTFB += p.Metrics.TTFB
				countTTFB++
			}
			if p.Metrics.FCP > 0 {
				sumFCP += p.Metrics.FCP
				countFCP++
			}
			if p.Metrics.LCP > 0 {
				sumLCP += p.Metrics.LCP
				countLCP++
			}
			if p.Metrics.CLS >= 0 {
				sumCLS += p.Metrics.CLS
				countCLS++
			}
			if p.Metrics.TBT >= 0 {
				sumTBT += p.Metrics.TBT
				countTBT++
			}

			// Aggregate slowest resources
			for _, res := range p.Diagnostics.SlowestResources {
				key := sanitizeResourceKey(res.Name)
				if existing, found := resourceMap[key]; found {
					existing.Occurrences++
					existing.TotalDurationMs += res.Duration
				} else {
					resourceMap[key] = &ResourceImpact{
						Name:            key,
						Type:            res.Type,
						Occurrences:     1,
						TotalDurationMs: res.Duration,
					}
				}
			}

			// Aggregate images across pages (tracking max rendered size across all pages)
			for _, img := range p.Diagnostics.LargestImages {
				if img.URL == "" {
					continue
				}
				key := img.URL
				naturalW := img.NaturalWidth
				if naturalW == 0 {
					naturalW = img.Width
				}
				naturalH := img.NaturalHeight
				if naturalH == 0 {
					naturalH = img.Height
				}
				renderedW := img.RenderedWidth
				renderedH := img.RenderedHeight

				if existing, found := imageMap[key]; found {
					if appendUniquePage(existing, p.URL) {
						existing.PageCount = len(existing.Pages)
					}
					existing.AvgDurationMs += img.Duration
					if img.IsLazy {
						existing.IsLazy = true
					}
					if img.IsLCP {
						existing.IsLCP = true
					}
					if naturalW > existing.NaturalWidth {
						existing.NaturalWidth = naturalW
						existing.Width = naturalW
					}
					if naturalH > existing.NaturalHeight {
						existing.NaturalHeight = naturalH
						existing.Height = naturalH
					}
					// Always track the MAXIMUM rendered width & height across all pages where image is used
					if renderedW > existing.MaxRenderedWidth {
						existing.MaxRenderedWidth = renderedW
					}
					if renderedH > existing.MaxRenderedHeight {
						existing.MaxRenderedHeight = renderedH
					}
					if img.TransferSize > existing.MaxTransferSize {
						existing.MaxTransferSize = img.TransferSize
						existing.FormattedSize = img.FormattedSize
						if img.Format != "" {
							existing.Format = img.Format
						}
					}
				} else {
					fmtStr := img.Format
					if fmtStr == "" {
						fmtStr = detectFormatFromURL(img.URL)
					}
					agg := &AggregatedImage{
						URL:               img.URL,
						MaxTransferSize:   img.TransferSize,
						FormattedSize:     img.FormattedSize,
						AvgDurationMs:     img.Duration,
						Width:             naturalW,
						Height:            naturalH,
						NaturalWidth:      naturalW,
						NaturalHeight:     naturalH,
						MaxRenderedWidth:  renderedW,
						MaxRenderedHeight: renderedH,
						Format:            fmtStr,
						IsLazy:            img.IsLazy,
						IsLCP:             img.IsLCP,
					}
					appendUniquePage(agg, p.URL)
					agg.PageCount = len(agg.Pages)
					imageMap[key] = agg
				}
			}

			// Aggregate fonts across pages
			for _, font := range p.Diagnostics.Fonts {
				fam := strings.TrimSpace(font.Family)
				if fam == "" {
					continue
				}
				key := strings.ToLower(fam)
				if existing, found := fontMap[key]; found {
					existing.Occurrences++
					existing.AvgDurationMs += font.Duration
					if font.TransferSize > existing.TransferSize {
						existing.TransferSize = font.TransferSize
					}
					if existing.URL == "" && font.URL != "" {
						existing.URL = font.URL
						existing.Type = font.Type
					}
					if p.URL != "" && !containsString(existing.PageURLs, p.URL) {
						existing.PageURLs = append(existing.PageURLs, p.URL)
					}
				} else {

					var initialURLs []string
					if p.URL != "" {
						initialURLs = append(initialURLs, p.URL)
					}
					fontMap[key] = &AggregatedFont{
						Family:        fam,
						URL:           font.URL,
						Type:          font.Type,
						Occurrences:   1,
						AvgDurationMs: font.Duration,
						TransferSize:  font.TransferSize,
						PageURLs:      initialURLs,
					}
				}
			}

			// Aggregate iframes across pages
			for _, frame := range p.Diagnostics.Iframes {
				key := strings.TrimSpace(frame.Src)
				if key == "" {
					continue
				}
				if frame.LoadedDuringScan {
					totalLoadedIframes++
				} else {
					totalMissedIframes++
				}
				if existing, found := iframeMap[key]; found {
					existing.Occurrences++
					existing.AvgDurationMs += frame.Duration
					if frame.IsLazy {
						existing.IsLazy = true
					}
					if frame.LoadedDuringScan {
						existing.LoadedCount++
					} else {
						existing.MissedCount++
					}
					if frame.TransferSize > existing.MaxTransferSize {
						existing.MaxTransferSize = frame.TransferSize
						existing.FormattedSize = frame.FormattedSize
						existing.Width = frame.Width
						existing.Height = frame.Height
					}
					if existing.Title == "" && frame.Title != "" {
						existing.Title = frame.Title
					}
					if p.URL != "" && !containsString(existing.Pages, p.URL) {
						existing.Pages = append(existing.Pages, p.URL)
						existing.PageCount = len(existing.Pages)
					}
				} else {
					pages := []string{}
					if p.URL != "" {
						pages = append(pages, p.URL)
					}
					loaded, missed := 0, 0
					if frame.LoadedDuringScan {
						loaded = 1
					} else {
						missed = 1
					}
					iframeMap[key] = &AggregatedIframe{
						Src:             key,
						Title:           frame.Title,
						PageCount:       len(pages),
						Pages:           pages,
						Occurrences:     1,
						LoadedCount:     loaded,
						MissedCount:     missed,
						IsLazy:          frame.IsLazy,
						AvgDurationMs:   frame.Duration,
						MaxTransferSize: frame.TransferSize,
						FormattedSize:   frame.FormattedSize,
						Width:           frame.Width,
						Height:          frame.Height,
					}
				}
			}

		}
	}

	avgMetrics := make(map[string]float64)
	if countTTFB > 0 {
		avgMetrics["TTFB"] = math.Round(sumTTFB/float64(countTTFB)*100) / 100
	}
	if countFCP > 0 {
		avgMetrics["FCP"] = math.Round(sumFCP/float64(countFCP)*100) / 100
	}
	if countLCP > 0 {
		avgMetrics["LCP"] = math.Round(sumLCP/float64(countLCP)*100) / 100
	}
	if countCLS > 0 {
		avgMetrics["CLS"] = math.Round(sumCLS/float64(countCLS)*1000) / 1000
	}
	if countTBT > 0 {
		avgMetrics["TBT"] = math.Round(sumTBT/float64(countTBT)*100) / 100
	}

	// Calculate Top Resource Bottlenecks
	var resourceList []ResourceImpact
	for _, item := range resourceMap {
		item.AvgDurationMs = math.Round((item.TotalDurationMs/float64(item.Occurrences))*100) / 100
		resourceList = append(resourceList, *item)
	}

	// Sort by total impact duration descending
	sort.Slice(resourceList, func(i, j int) bool {
		return resourceList[i].TotalDurationMs > resourceList[j].TotalDurationMs
	})

	if len(resourceList) > 8 {
		resourceList = resourceList[:8]
	}

	// Process and finalize ALL images
	var allImages []AggregatedImage
	formatBreakdown := make(map[string]int)
	var totalPayloadBytes int64
	var heavyCount, oversizedCount, nonWebPCount, missingLazyCount int
	var totalSavingsBytes int64

	for _, img := range imageMap {
		if img.PageCount > 0 {
			img.AvgDurationMs = math.Round((img.AvgDurationMs / float64(img.PageCount)) * 100) / 100
		}
		if img.FormattedSize == "" || img.FormattedSize == "0 B" {
			img.FormattedSize = formatBytes(img.MaxTransferSize)
		}

		if img.Format == "" {
			img.Format = detectFormatFromURL(img.URL)
		}

		// Calculate Recommended Retina Dimensions (MaxRenderedWidth * 2, strictly capped at Natural dimensions)
		if img.MaxRenderedWidth > 0 {
			retinaW := img.MaxRenderedWidth * 2
			retinaH := img.MaxRenderedHeight * 2
			// NEVER upscale beyond natural original dimensions
			if img.NaturalWidth > 0 && retinaW > img.NaturalWidth {
				retinaW = img.NaturalWidth
			}
			if img.NaturalHeight > 0 && retinaH > img.NaturalHeight {
				retinaH = img.NaturalHeight
			}
			img.RecommendedRetinaWidth = retinaW
			img.RecommendedRetinaHeight = retinaH

			// Image is oversized if its intrinsic natural width exceeds 2x Retina display requirements
			if img.NaturalWidth > (img.MaxRenderedWidth * 2) {
				img.IsOversized = true
				oversizedCount++
			}
		} else {
			img.RecommendedRetinaWidth = img.NaturalWidth
			img.RecommendedRetinaHeight = img.NaturalHeight
		}

		formatBreakdown[img.Format]++
		totalPayloadBytes += img.MaxTransferSize

		// Check if heavy (>= thresholdBytes, default 100 KB = 102400 B)
		if img.MaxTransferSize >= thresholdBytes {
			img.IsHeavy = true
			heavyCount++
		}

		if img.Format != "webp" && img.Format != "avif" && img.Format != "svg" {
			nonWebPCount++
		}

		if !img.IsLazy && !img.IsLCP {
			missingLazyCount++
		}

		// Calculate WebP compression estimate
		var estSize int64
		if activeConfig != nil {
			estSize = EstimateWebPSize(img.Format, img.MaxTransferSize, *activeConfig)
		} else {
			estSize = EstimateWebPSize(img.Format, img.MaxTransferSize)
		}
		img.EstimatedWebPSize = estSize
		img.EstimatedWebPFormatted = formatBytes(estSize)
		savings := img.MaxTransferSize - estSize
		if savings < 0 {
			savings = 0
		}
		img.EstimatedSavingsBytes = savings
		img.EstimatedSavingsFormatted = formatBytes(savings)
		if img.MaxTransferSize > 0 {
			img.EstimatedSavingsPercent = math.Round((float64(savings)/float64(img.MaxTransferSize)*100)*10) / 10
		}

		totalSavingsBytes += savings
		allImages = append(allImages, *img)
	}

	sort.Slice(allImages, func(i, j int) bool {
		if allImages[i].MaxTransferSize != allImages[j].MaxTransferSize {
			return allImages[i].MaxTransferSize > allImages[j].MaxTransferSize
		}
		return allImages[i].AvgDurationMs > allImages[j].AvgDurationMs
	})

	topLargestImages := allImages
	if len(topLargestImages) > 8 {
		topLargestImages = topLargestImages[:8]
	}

	// Finalize fonts list
	var fontList []AggregatedFont
	for _, font := range fontMap {
		if font.Occurrences > 0 {
			font.AvgDurationMs = math.Round((font.AvgDurationMs/float64(font.Occurrences))*100) / 100
			font.Percentage = math.Round((float64(font.Occurrences)/float64(total)*100)*10) / 10
		}
		if font.TransferSize > 0 {
			font.FormattedSize = formatBytes(font.TransferSize)
		} else {
			font.FormattedSize = "н/д"
		}
		fontList = append(fontList, *font)
	}

	sort.Slice(fontList, func(i, j int) bool {
		return fontList[i].Occurrences > fontList[j].Occurrences
	})

	var iframeList []AggregatedIframe
	for _, frame := range iframeMap {
		if frame.Occurrences > 0 {
			frame.AvgDurationMs = math.Round(frame.AvgDurationMs/float64(frame.Occurrences)*100) / 100
		}
		if frame.FormattedSize == "" && frame.MaxTransferSize > 0 {
			frame.FormattedSize = formatBytes(frame.MaxTransferSize)
		} else if frame.FormattedSize == "" {
			frame.FormattedSize = "н/д"
		}
		iframeList = append(iframeList, *frame)
	}
	sort.Slice(iframeList, func(i, j int) bool {
		if iframeList[i].MissedCount != iframeList[j].MissedCount {
			return iframeList[i].MissedCount > iframeList[j].MissedCount
		}
		return iframeList[i].PageCount > iframeList[j].PageCount
	})

	// Aggregate Forms across pages
	formMap := make(map[string]*AggregatedForm)
	totalFormsCount := 0
	pagesWithFormsSet := make(map[string]bool)
	captchaProtectedCount := 0
	unprotectedFormsCount := 0
	fileUploadFormsCount := 0
	formEngineBreakdown := make(map[string]int)

	for _, res := range results {
		if len(res.Diagnostics.Forms) > 0 {
			pagesWithFormsSet[res.URL] = true
		}
		for _, f := range res.Diagnostics.Forms {
			totalFormsCount++
			formEngineBreakdown[f.Engine]++

			if f.Captcha.IsActive {
				captchaProtectedCount++
			} else {
				unprotectedFormsCount++
			}

			if f.HasFileUpload {
				fileUploadFormsCount++
			}

			formKey := f.Engine + ":" + f.Title
			if f.Title == "" || strings.HasPrefix(f.Title, "form-") {
				formKey = f.Engine + ":" + f.ID + ":" + f.Action
			}

			if existing, found := formMap[formKey]; found {
				existing.PageCount++
				hasPage := false
				for _, p := range existing.Pages {
					if p == res.URL {
						hasPage = true
						break
					}
				}
				if !hasPage {
					existing.Pages = append(existing.Pages, res.URL)
				}
				if len(existing.Fields) == 0 && len(f.Fields) > 0 {
					existing.Fields = f.Fields
					existing.FieldCount = f.FieldCount
				}
				if !existing.HasFileUpload && f.HasFileUpload {
					existing.HasFileUpload = true
					existing.AllowedFileTypes = f.AllowedFileTypes
				}
				if !existing.Captcha.IsActive && f.Captcha.IsActive {
					existing.Captcha = f.Captcha
				}
			} else {
				formMap[formKey] = &AggregatedForm{
					ID:               f.ID,
					Title:            f.Title,
					Engine:           f.Engine,
					Method:           f.Method,
					Action:           f.Action,
					PageCount:        1,
					Pages:            []string{res.URL},
					Fields:           f.Fields,
					FieldCount:       f.FieldCount,
					HasFileUpload:    f.HasFileUpload,
					AllowedFileTypes: f.AllowedFileTypes,
					Captcha:          f.Captcha,
					HiddenTokens:     f.HiddenTokens,
				}
			}
		}
	}

	var formList []AggregatedForm
	for _, f := range formMap {
		formList = append(formList, *f)
	}
	sort.Slice(formList, func(i, j int) bool {
		return formList[i].PageCount > formList[j].PageCount
	})

	// Calculate Health Score (0 - 100)
	goodCount := statusCounts["good"]
	needsImpCount := statusCounts["needs-improvement"]
	healthScore := 0
	if total > 0 {
		healthScore = int(math.Round((float64(goodCount)*1.0 + float64(needsImpCount)*0.5) / float64(total) * 100))
	}

	// Build Global Fixes Recommendations
	fixes := generateGlobalFixes(avgMetrics, resourceList, topLargestImages, fontList, statusCounts, total)
	if totalMissedIframes > 0 {
		fixes = append(fixes, fmt.Sprintf("Виявлено %d iframe без мережевого timing під час load (lazy/below-fold). Перевірте вкладку «iframe» — %d унікальних src.", totalMissedIframes, len(iframeList)))
	}
	if unprotectedFormsCount > 0 {
		fixes = append(fixes, fmt.Sprintf("Виявлено %d форм без анти-спам захисту (відсутня reCAPTCHA/Turnstile). Перевірте вкладку «Форми».", unprotectedFormsCount))
	}

	return SiteAnalytics{
		TotalPages:                 total,
		HealthScore:                healthScore,
		StatusCounts:               statusCounts,
		AverageMetrics:             avgMetrics,
		TopResourceBottlenecks:     resourceList,
		LargestImages:              topLargestImages,
		AllImages:                  allImages,
		FontUsage:                  fontList,
		Iframes:                    iframeList,
		Forms:                      formList,
		GlobalFixes:                fixes,
		TotalImagePayloadBytes:     totalPayloadBytes,
		TotalImagePayloadFormatted: formatBytes(totalPayloadBytes),
		TotalImageCount:            len(allImages),
		HeavyImagesCount:           heavyCount,
		OversizedImagesCount:       oversizedCount,
		NonWebPCount:               nonWebPCount,
		MissingLazyCount:           missingLazyCount,
		TotalWebPSavingsBytes:      totalSavingsBytes,
		TotalWebPSavingsFormatted:  formatBytes(totalSavingsBytes),
		FormatBreakdown:            formatBreakdown,
		TotalIframeCount:           len(iframeList),
		MissedIframeCount:          totalMissedIframes,
		LoadedIframeCount:          totalLoadedIframes,
		TotalFormsCount:            totalFormsCount,
		PagesWithFormsCount:        len(pagesWithFormsSet),
		CaptchaProtectedCount:      captchaProtectedCount,
		UnprotectedFormsCount:      unprotectedFormsCount,
		FileUploadFormsCount:       fileUploadFormsCount,
		FormEngineBreakdown:        formEngineBreakdown,
	}
}

func formatBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func sanitizeResourceKey(rawName string) string {
	// Trim query string if overly long for clean grouping
	idx := strings.Index(rawName, "?")
	if idx > 0 && len(rawName) > 60 {
		return rawName[:idx]
	}
	return rawName
}

func generateGlobalFixes(avg map[string]float64, topRes []ResourceImpact, images []AggregatedImage, fonts []AggregatedFont, status map[string]int, total int) []string {
	var fixes []string

	if avg["LCP"] > 2500 {
		fixes = append(fixes, fmt.Sprintf("Оптимізуйте головні зображення сайту (LCP середовище: %.0fms). Переведіть у WebP/AVIF та додайте fetchpriority='high'.", avg["LCP"]))
	}
	if avg["TTFB"] > 800 {
		fixes = append(fixes, fmt.Sprintf("Високий час відповіді сервера (середній TTFB: %.0fms). Налаштуйте серверне кешування (Redis/CDN/OPcache).", avg["TTFB"]))
	}
	if avg["TBT"] > 200 {
		fixes = append(fixes, fmt.Sprintf("Завдання виділення основного потоку (TBT: %.0fms). Розбийте важкі JS бандли на менші модулі.", avg["TBT"]))
	}
	if len(topRes) > 0 && topRes[0].Occurrences > 1 {
		fixes = append(fixes, fmt.Sprintf("Ресурс '%s' викликає затримку на %d сторінках сайту. Налаштуйте його асинхронне завантаження або CDN-кеш.", topRes[0].Name, topRes[0].Occurrences))
	}
	if len(images) > 0 && images[0].MaxTransferSize > 400*1024 {
		fixes = append(fixes, fmt.Sprintf("Виявлено велике зображення '%s' (%s). Оптимізуйте та стисніть його у формати WebP/AVIF.", images[0].URL, images[0].FormattedSize))
	}
	if len(fonts) > 3 {
		fixes = append(fixes, fmt.Sprintf("На сайті використовується %d різних гарнітур шрифтів. Зменшіть кількість шрифтів та підключіть 'font-display: swap'.", len(fonts)))
	}
	if status["poor"] > 0 {
		fixes = append(fixes, fmt.Sprintf("%d з %d сторінок мають критичні проблеми з продуктивністю. Проведіть першочергову оптимізацію цих сторінок.", status["poor"], total))
	}

	return fixes
}

// appendUniquePage adds pageURL to img.Pages if not already present. Returns true when added.
func appendUniquePage(img *AggregatedImage, pageURL string) bool {
	if img == nil || pageURL == "" {
		return false
	}
	for _, u := range img.Pages {
		if u == pageURL {
			return false
		}
	}
	img.Pages = append(img.Pages, pageURL)
	return true
}

func detectFormatFromURL(rawURL string) string {
	if rawURL == "" {
		return "unknown"
	}
	u := strings.ToLower(rawURL)
	idx := strings.Index(u, "?")
	if idx > 0 {
		u = u[:idx]
	}
	if strings.HasSuffix(u, ".png") {
		return "png"
	}
	if strings.HasSuffix(u, ".jpg") || strings.HasSuffix(u, ".jpeg") {
		return "jpg"
	}
	if strings.HasSuffix(u, ".webp") {
		return "webp"
	}
	if strings.HasSuffix(u, ".avif") {
		return "avif"
	}
	if strings.HasSuffix(u, ".svg") {
		return "svg"
	}
	if strings.HasSuffix(u, ".gif") {
		return "gif"
	}
	return "other"
}

func GenerateImageComparisonHTML(analytics SiteAnalytics, domain string) string {
	var rowsHTML strings.Builder
	for idx, img := range analytics.AllImages {
		lazyBadge := `<span style="color: #ef4444; background: rgba(239,68,68,0.1); padding: 2px 8px; border-radius: 4px; font-weight: bold;">Відсутнє ⚠️</span>`
		if img.IsLazy {
			lazyBadge = `<span style="color: #10b981; background: rgba(16,185,129,0.1); padding: 2px 8px; border-radius: 4px; font-weight: bold;">loading="lazy" 🟢</span>`
		}

		lcpBadge := ""
		if img.IsLCP {
			lcpBadge = `<span style="color: #f59e0b; background: rgba(245,158,11,0.1); padding: 2px 8px; border-radius: 4px; font-weight: bold; margin-left: 4px;">LCP Hero 🔥</span>`
		}

		heavyStyle := ""
		if img.IsHeavy {
			heavyStyle = "background-color: rgba(244,63,94,0.05);"
		}

		fmtBadgeClass := "color: #38bdf8; border: 1px solid rgba(56,189,248,0.3);"
		if img.Format == "png" || img.Format == "jpg" {
			fmtBadgeClass = "color: #f59e0b; border: 1px solid rgba(245,158,11,0.3);"
		} else if img.Format == "webp" || img.Format == "avif" {
			fmtBadgeClass = "color: #10b981; border: 1px solid rgba(16,185,129,0.3);"
		}

		// Escape URL quotes for JS function parameter
		escapedURL := strings.ReplaceAll(img.URL, "'", "\\'")

		dimBadge := fmt.Sprintf(`<span style="color: #94a3b8; font-size: 12px;">%dx%d px</span>`, img.Width, img.Height)
		if img.MaxRenderedWidth > 0 {
			retinaBadge := fmt.Sprintf(`<div style="font-size: 11px; color: #38bdf8; margin-top: 2px;">Рендер: %d×%d (Retina 2x: %d×%d)</div>`, img.MaxRenderedWidth, img.MaxRenderedHeight, img.RecommendedRetinaWidth, img.RecommendedRetinaHeight)
			if img.IsOversized {
				retinaBadge += fmt.Sprintf(`<div style="font-size: 10px; color: #ef4444; background: rgba(239,68,68,0.15); padding: 1px 4px; border-radius: 3px; display: inline-block; margin-top: 2px;">⚠️ Завелике для рендеру</div>`)
			}
			dimBadge += retinaBadge
		}

		rowsHTML.WriteString(fmt.Sprintf(`
		<tr style="%s border-bottom: 1px solid #334155;">
			<td style="padding: 12px; font-family: monospace; font-size: 12px;">%d</td>
			<td style="padding: 12px; text-align: center;">
				<img src="%s" loading="lazy" style="height: 44px; max-width: 70px; object-fit: contain; border-radius: 6px; border: 1px solid #475569; background: #020617; cursor: pointer; transition: transform 0.2s;" onclick="openModal('%s', '%s', '%s', '%.1f%%')" title="Клацніть для порівняння якості" />
			</td>
			<td style="padding: 12px; max-width: 280px; word-break: break-all;">
				<a href="%s" target="_blank" style="color: #e2e8f0; text-decoration: underline; font-family: monospace; font-size: 12px;">%s</a>
				<div style="margin-top: 4px;">%s</div>
			</td>
			<td style="padding: 12px; text-align: center;">
				<span style="padding: 2px 8px; border-radius: 4px; font-size: 11px; font-family: monospace; uppercase; %s">%s</span>
			</td>
			<td style="padding: 12px; text-align: center;">%s</td>
			<td style="padding: 12px; text-align: center; color: #38bdf8; font-weight: bold; font-size: 12px;">%d стор.</td>
			<td style="padding: 12px; text-align: right; color: #f1f5f9; font-weight: bold; font-family: monospace; font-size: 13px;">%s</td>
			<td style="padding: 12px; text-align: right; color: #10b981; font-weight: bold; font-family: monospace; font-size: 13px;">%s</td>
			<td style="padding: 12px; text-align: right; color: #10b981; font-weight: bold; font-family: monospace; font-size: 13px;">-%s (%.1f%%)</td>
			<td style="padding: 12px; text-align: center; font-size: 12px;">%s</td>
			<td style="padding: 12px; text-align: center;">
				<button onclick="openModal('%s', '%s', '%s', '%.1f%%')" style="background: #0284c7; color: white; border: none; border-radius: 6px; padding: 6px 12px; font-size: 11px; font-weight: bold; cursor: pointer;">👁️ Порівняти</button>
			</td>
		</tr>
		`, heavyStyle, idx+1, img.URL, escapedURL, img.FormattedSize, img.EstimatedWebPFormatted, img.EstimatedSavingsPercent, img.URL, img.URL, lcpBadge, fmtBadgeClass, strings.ToUpper(img.Format), dimBadge, img.PageCount, img.FormattedSize, img.EstimatedWebPFormatted, img.EstimatedSavingsFormatted, img.EstimatedSavingsPercent, lazyBadge, escapedURL, img.FormattedSize, img.EstimatedWebPFormatted, img.EstimatedSavingsPercent))
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="uk">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Звіт порівняння зображень (Original vs WebP) - %s</title>
	<style>
		body { background-color: #0f172a; color: #f8fafc; font-family: system-ui, -apple-system, sans-serif; padding: 32px; margin: 0; }
		.container { max-width: 1380px; margin: 0 auto; }
		.header { border-bottom: 1px solid #334155; padding-bottom: 24px; margin-bottom: 32px; }
		.title { font-size: 28px; font-weight: 800; color: #38bdf8; margin: 0 0 8px 0; }
		.subtitle { font-size: 14px; color: #94a3b8; margin: 0; }
		.cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 32px; }
		.card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 20px; }
		.card-title { font-size: 12px; font-weight: 600; color: #94a3b8; text-transform: uppercase; margin-bottom: 8px; }
		.card-val { font-size: 24px; font-weight: 800; color: #f8fafc; }
		table { width: 100%%; border-collapse: collapse; background: #1e293b; border-radius: 12px; overflow: hidden; border: 1px solid #334155; }
		th { background: #090d16; color: #94a3b8; font-size: 11px; text-transform: uppercase; padding: 14px 12px; text-align: left; }
		
		/* Modal overlay styling */
		.modal-overlay { display: none; position: fixed; inset: 0; background: rgba(2, 6, 23, 0.88); backdrop-filter: blur(10px); z-index: 1000; justify-content: center; align-items: flex-start; padding: 32px 16px; overflow-y: auto; }
		.modal-card { background: #0f172a; border: 1px solid #334155; border-radius: 24px; width: 100%%; max-width: 1150px; padding: 28px; box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.7); margin: auto; }
		.modal-grid { display: flex; flex-direction: column; gap: 24px; margin-top: 24px; }
		.modal-box { background: #020617; border: 1px solid #334155; border-radius: 16px; padding: 20px; text-align: center; }
		.modal-box img { max-height: 600px; width: auto; max-width: 100%%; object-fit: contain; border-radius: 12px; }
		.close-btn { background: #334155; color: white; border: none; padding: 8px 18px; border-radius: 10px; font-weight: bold; cursor: pointer; float: right; font-size: 13px; }
	</style>
</head>
<body>
	<div class="container">
		<div class="header">
			<h1 class="title">🖼️ Порівняльний аналіз зображень (Original vs WebP)</h1>
			<p class="subtitle">Звіт з оцінки обсягу зображень та потенціалу оптимізації для %s</p>
		</div>

		<div class="cards">
			<div class="card">
				<div class="card-title">Загальний обсяг зображень</div>
				<div class="card-val" style="color: #f59e0b;">%s</div>
			</div>
			<div class="card">
				<div class="card-title">Всього зображень</div>
				<div class="card-val">%d</div>
			</div>
			<div class="card">
				<div class="card-title">Важкі зображення (>100KB)</div>
				<div class="card-val" style="color: #f43f5e;">%d</div>
			</div>
			<div class="card">
				<div class="card-title">Потенційна економія (WebP)</div>
				<div class="card-val" style="color: #10b981;">%s</div>
			</div>
		</div>

		<table>
			<thead>
				<tr>
					<th>#</th>
					<th style="text-align: center;">Прев'ю</th>
					<th>Зображення / URL</th>
					<th style="text-align: center;">Формат</th>
					<th style="text-align: center;">Роздільна здатність</th>
					<th style="text-align: center;">Сторінок</th>
					<th style="text-align: right;">Поточний розмір</th>
					<th style="text-align: right;">Оцінка WebP</th>
					<th style="text-align: right;">Економія (KB / %%)</th>
					<th style="text-align: center;">Lazy Loading</th>
					<th style="text-align: center;">Дії</th>
				</tr>
			</thead>
			<tbody>
				%s
			</tbody>
		</table>
	</div>

	<!-- Interactive Modal for HTML Report (Row-by-Row Stacked & Enlarged) -->
	<div id="modalOverlay" class="modal-overlay" onclick="closeModal(event)">
		<div class="modal-card" onclick="event.stopPropagation()">
			<button class="close-btn" onclick="closeModal()">✕ Закрити</button>
			<h2 style="margin: 0 0 4px 0; font-size: 20px; color: #38bdf8;">🖼️ Порівняльний Аналіз Зображення (Original vs WebP)</h2>
			<div id="modalUrl" style="font-family: monospace; font-size: 12px; color: #94a3b8; word-break: break-all;"></div>

			<div class="modal-grid">
				<div class="modal-box">
					<div style="font-size: 14px; font-weight: bold; color: #f8fafc; margin-bottom: 12px; text-align: left; border-bottom: 1px solid #1e293b; padding-bottom: 8px;">🔴 Оригінальне Зображення (<span id="modalOrigSize"></span>)</div>
					<img id="modalOrigImg" src="" alt="Original Image">
				</div>
				<div class="modal-box" style="border-color: rgba(16,185,129,0.4);">
					<div style="font-size: 14px; font-weight: bold; color: #10b981; margin-bottom: 12px; text-align: left; border-bottom: 1px solid #1e293b; padding-bottom: 8px;">🟢 WebP Оптимізована Оцінка (<span id="modalWebPSize"></span>, Економія: -<span id="modalSavings"></span>)</div>
					<img id="modalWebPImg" src="" alt="WebP Preview">
				</div>
			</div>
		</div>
	</div>

	<script>
		function openModal(url, origSize, webpSize, savings) {
			document.getElementById('modalUrl').innerText = url;
			document.getElementById('modalOrigSize').innerText = origSize;
			document.getElementById('modalWebPSize').innerText = webpSize;
			document.getElementById('modalSavings').innerText = savings;
			document.getElementById('modalOrigImg').src = url;
			document.getElementById('modalWebPImg').src = '';

			// Convert image on-the-fly to real WebP base64 via HTML5 Canvas
			var img = new Image();
			img.crossOrigin = 'Anonymous';
			img.onload = function() {
				try {
					var canvas = document.createElement('canvas');
					canvas.width = img.naturalWidth || img.width;
					canvas.height = img.naturalHeight || img.height;
					var ctx = canvas.getContext('2d');
					ctx.drawImage(img, 0, 0);
					var webpDataUrl = canvas.toDataURL('image/webp', 0.80);
					if (webpDataUrl && webpDataUrl.startsWith('data:image/webp')) {
						document.getElementById('modalWebPImg').src = webpDataUrl;
					} else {
						document.getElementById('modalWebPImg').src = url;
					}
				} catch (e) {
					document.getElementById('modalWebPImg').src = url;
				}
			};
			img.onerror = function() {
				document.getElementById('modalWebPImg').src = url;
			};
			img.src = url;

			document.getElementById('modalOverlay').style.display = 'flex';
		}
		function closeModal() {
			document.getElementById('modalOverlay').style.display = 'none';
		}
		document.addEventListener('keydown', function(e) {
			if (e.key === 'Escape') closeModal();
		});
	</script>
</body>
</html>`, domain, domain, analytics.TotalImagePayloadFormatted, analytics.TotalImageCount, analytics.HeavyImagesCount, analytics.TotalWebPSavingsFormatted, rowsHTML.String())

	return html
}

func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
