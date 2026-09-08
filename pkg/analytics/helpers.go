package analytics

import (
	"fmt"
	"strings"
)

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


func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
