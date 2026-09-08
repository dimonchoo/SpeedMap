package analytics

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
)


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
	case "svg":
		return int64(float64(originalBytes) * 0.15)
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
					u := strings.Split(img.URL, "?")[0]
					baseName := filepath.Base(u)
					if baseName == "" || baseName == "." || baseName == "/" {
						baseName = "image"
					}
					agg := &AggregatedImage{
						URL:               img.URL,
						Basename:          baseName,
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
		if img.Basename == "" {
			u := strings.Split(img.URL, "?")[0]
			base := filepath.Base(u)
			if base == "" || base == "." || base == "/" {
				base = "image"
			}
			img.Basename = base
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

		if img.Format != "webp" && img.Format != "avif" && (img.Format != "svg" || img.IsHeavy) {
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

	// Aggregate DOM Virtualization across all pages
	totalDomNodes := 0
	heavyDomPages := 0
	selectorFrequency := make(map[string]int)
	selectorNodes := make(map[string]int)

	for _, p := range results {
		if p.Diagnostics.DOMVirtualization != nil {
			totalDomNodes += p.Diagnostics.DOMVirtualization.TotalDOMNodes
			if p.Diagnostics.DOMVirtualization.TotalDOMNodes > 800 {
				heavyDomPages++
			}
			for _, c := range p.Diagnostics.DOMVirtualization.Candidates {
				if !c.IsOptimized && c.Selector != "" {
					selectorFrequency[c.Selector]++
					selectorNodes[c.Selector] += c.DOMNodes
				}
			}
		}
	}

	type selScore struct {
		selector string
		freq     int
		nodes    int
	}
	var scoredSelectors []selScore
	for sel, freq := range selectorFrequency {
		scoredSelectors = append(scoredSelectors, selScore{
			selector: sel,
			freq:     freq,
			nodes:    selectorNodes[sel],
		})
	}
	sort.Slice(scoredSelectors, func(i, j int) bool {
		if scoredSelectors[i].freq != scoredSelectors[j].freq {
			return scoredSelectors[i].freq > scoredSelectors[j].freq
		}
		return scoredSelectors[i].nodes > scoredSelectors[j].nodes
	})

	var globalCandidateSelectors []string
	for _, s := range scoredSelectors {
		globalCandidateSelectors = append(globalCandidateSelectors, s.selector)
	}

	globalCSS := ""
	globalPHP := ""
	if len(globalCandidateSelectors) > 0 {
		joined := strings.Join(globalCandidateSelectors, ",\n")
		globalCSS = "/* SpeedMap: Global DOM Virtualization (Auto-Generated) */\n" +
			joined + " {\n" +
			"    content-visibility: auto;\n" +
			"    contain-intrinsic-size: auto 600px;\n" +
			"}"

		globalPHP = "add_action('wp_head', function() {\n" +
			"    ?>\n" +
			"    <style id=\"speedmap-dom-virtualization\">\n" +
			"    " + strings.ReplaceAll(joined, "\n", "\n    ") + " {\n" +
			"        content-visibility: auto;\n" +
			"        contain-intrinsic-size: auto 600px;\n" +
			"    }\n" +
			"    </style>\n" +
			"    <?php\n" +
			"}, 1);"
	}

	avgDomNodes := 0
	if total > 0 {
		avgDomNodes = totalDomNodes / total
	}

	if heavyDomPages > 0 {
		fixes = append(fixes, fmt.Sprintf("Виявлено %d сторінок із важким DOM (>800 елементів). Використайте «DOM Virtualization» (content-visibility) для суттєвого прискорення FCP/LCP.", heavyDomPages))
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
		SVGCount:                   formatBreakdown["svg"],
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
		TotalDOMNodesAcrossPages:   totalDomNodes,
		AverageDOMNodesPerPage:     avgDomNodes,
		HeavyDOMPagesCount:         heavyDomPages,
		GlobalCandidateSelectors:   globalCandidateSelectors,
		GlobalVirtualizationCSS:    globalCSS,
		GlobalVirtualizationPHP:    globalPHP,
	}
}

