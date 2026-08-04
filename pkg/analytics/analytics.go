package analytics

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"SpeedMap/pkg/scanner"
)

type ResourceImpact struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Occurrences     int     `json:"occurrences"`
	AvgDurationMs   float64 `json:"avgDurationMs"`
	TotalDurationMs float64 `json:"totalDurationMs"`
}

type SiteAnalytics struct {
	TotalPages scannedCount `json:"totalPages"`
	HealthScore int          `json:"healthScore"` // 0 - 100

	StatusCounts map[string]int `json:"statusCounts"` // good, needs-improvement, poor, error

	AverageMetrics map[string]float64 `json:"averageMetrics"` // TTFB, FCP, LCP, CLS, TBT

	TopResourceBottlenecks []ResourceImpact `json:"topResourceBottlenecks"`
	GlobalFixes            []string         `json:"globalFixes"`
}

type scannedCount = int

func ComputeSiteAnalytics(results []scanner.PageResult) SiteAnalytics {
	total := len(results)
	if total == 0 {
		return SiteAnalytics{
			StatusCounts:   make(map[string]int),
			AverageMetrics: make(map[string]float64),
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

	// Calculate Health Score (0 - 100)
	goodCount := statusCounts["good"]
	needsImpCount := statusCounts["needs-improvement"]
	healthScore := 0
	if total > 0 {
		healthScore = int(math.Round((float64(goodCount)*1.0 + float64(needsImpCount)*0.5) / float64(total) * 100))
	}

	// Build Global Fixes Recommendations
	fixes := generateGlobalFixes(avgMetrics, resourceList, statusCounts, total)

	return SiteAnalytics{
		TotalPages:             total,
		HealthScore:            healthScore,
		StatusCounts:           statusCounts,
		AverageMetrics:         avgMetrics,
		TopResourceBottlenecks: resourceList,
		GlobalFixes:            fixes,
	}
}

func sanitizeResourceKey(rawName string) string {
	// Trim query string if overly long for clean grouping
	idx := strings.Index(rawName, "?")
	if idx > 0 && len(rawName) > 60 {
		return rawName[:idx]
	}
	return rawName
}

func generateGlobalFixes(avg map[string]float64, topRes []ResourceImpact, status map[string]int, total int) []string {
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
	if status["poor"] > 0 {
		fixes = append(fixes, fmt.Sprintf("%d з %d сторінок мають критичні проблеми з продуктивністю. Проведіть першочергову оптимізацію цих сторінок.", status["poor"], total))
	}

	if len(fixes) == 0 {
		fixes = append(fixes, "Сайт у чудовому стані! Продовжуйте моніторити показники Core Web Vitals.")
	}

	return fixes
}
