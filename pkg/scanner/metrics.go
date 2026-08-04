package scanner

import (
	"fmt"
)

// WebVitalsCollectorJS returns both WebVitals metrics and granular diagnostic insights
const WebVitalsCollectorJS = `
new Promise((resolve) => {
    const data = {
        metrics: {
            ttfb: 0,
            fcp: 0,
            lcp: 0,
            cls: 0,
            tbt: 0
        },
        diagnostics: {
            dnsTime: 0,
            tcpTime: 0,
            tlsTime: 0,
            serverProcessing: 0,
            renderBlockingCount: 0,
            renderBlockingFiles: [],
            lcpElement: '',
            lcpUrl: '',
            shiftCount: 0,
            shiftCauses: [],
            longTasksCount: 0,
            maxLongTaskMs: 0,
            slowestResources: [],
            categories: {}
        }
    };

    // 1. Navigation Timing & Network breakdown
    try {
        const navEntries = performance.getEntriesByType('navigation');
        if (navEntries.length > 0) {
            const nav = navEntries[0];
            if (nav.domainLookupEnd > 0 && nav.domainLookupStart > 0) {
                data.diagnostics.dnsTime = Math.max(0, nav.domainLookupEnd - nav.domainLookupStart);
            }
            if (nav.connectEnd > 0 && nav.connectStart > 0) {
                data.diagnostics.tcpTime = Math.max(0, nav.connectEnd - nav.connectStart);
            }
            if (nav.secureConnectionStart > 0 && nav.connectEnd > 0) {
                data.diagnostics.tlsTime = Math.max(0, nav.connectEnd - nav.secureConnectionStart);
            }
            if (nav.responseStart > 0 && nav.requestStart > 0 && nav.responseStart >= nav.requestStart) {
                data.diagnostics.serverProcessing = Math.max(0, nav.responseStart - nav.requestStart);
                data.metrics.ttfb = data.diagnostics.serverProcessing;
            } else if (nav.responseStart > 0) {
                data.metrics.ttfb = nav.responseStart;
            }
        }
    } catch (e) {}

    // 2. FCP Paint & Render Blocking Resources
    try {
        const paintEntries = performance.getEntriesByType('paint');
        for (const entry of paintEntries) {
            if (entry.name === 'first-contentful-paint') {
                data.metrics.fcp = entry.startTime;
                break;
            } else if (entry.name === 'first-paint' && !data.metrics.fcp) {
                data.metrics.fcp = entry.startTime;
            }
        }

        const resources = performance.getEntriesByType('resource');
        const blocking = resources.filter(r => r.renderBlockingStatus === 'blocking' || (r.initiatorType === 'css' && r.duration > 100));
        data.diagnostics.renderBlockingCount = blocking.length;
        data.diagnostics.renderBlockingFiles = blocking.slice(0, 3).map(r => {
            try {
                return new URL(r.name).pathname.split('/').pop() || r.name;
            } catch(e) {
                return r.name;
            }
        });
    } catch (e) {}

    // 3. Observers for LCP, CLS, TBT
    let lcpValue = 0;
    let clsValue = 0;
    let longTaskCount = 0;
    let maxLongTask = 0;
    let totalLongTaskDuration = 0;
    let shiftCount = 0;

    try {
        const lcpObserver = new PerformanceObserver((entryList) => {
            const entries = entryList.getEntries();
            if (entries.length > 0) {
                const last = entries[entries.length - 1];
                lcpValue = last.startTime;
                if (last.element) {
                    const tag = last.element.tagName ? last.element.tagName.toLowerCase() : 'element';
                    const idStr = last.element.id ? '#' + last.element.id : '';
                    const classStr = last.element.className && typeof last.element.className === 'string' ? '.' + last.element.className.trim().split(/\s+/)[0] : '';
                    data.diagnostics.lcpElement = '<' + tag + idStr + classStr + '>';
                    data.diagnostics.lcpUrl = last.url || last.element.src || '';
                }
            }
        });
        lcpObserver.observe({ type: 'largest-contentful-paint', buffered: true });
    } catch (e) {}

    try {
        const clsObserver = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                if (!entry.hadRecentInput) {
                    clsValue += entry.value;
                    shiftCount++;
                }
            }
        });
        clsObserver.observe({ type: 'layout-shift', buffered: true });
    } catch (e) {}

    try {
        const longTaskObserver = new PerformanceObserver((entryList) => {
            for (const entry of entryList.getEntries()) {
                longTaskCount++;
                if (entry.duration > maxLongTask) {
                    maxLongTask = entry.duration;
                }
                if (entry.duration > 50) {
                    totalLongTaskDuration += (entry.duration - 50);
                }
            }
        });
        longTaskObserver.observe({ type: 'longtask', buffered: true });

        const existingLong = performance.getEntriesByType('longtask');
        for (const entry of existingLong) {
            longTaskCount++;
            if (entry.duration > maxLongTask) maxLongTask = entry.duration;
            if (entry.duration > 50) totalLongTaskDuration += (entry.duration - 50);
        }
    } catch (e) {}

    // Check for images without explicit dimensions (CLS causes)
    try {
        const imgs = Array.from(document.querySelectorAll('img'));
        const unsized = imgs.filter(i => !i.getAttribute('width') || !i.getAttribute('height'));
        if (unsized.length > 0) {
            data.diagnostics.shiftCauses.push(unsized.length + ' зображень без вказаних розмірів width/height');
        }
    } catch (e) {}

    // 4. Slowest Resources Breakdown
    try {
        const resources = performance.getEntriesByType('resource');
        const slow = resources
            .filter(r => r.duration > 150)
            .sort((a, b) => b.duration - a.duration)
            .slice(0, 5)
            .map(r => {
                let nameStr = r.name;
                try {
                    const urlObj = new URL(r.name);
                    nameStr = urlObj.pathname.split('/').pop() || urlObj.hostname;
                } catch(err) {}
                return {
                    name: nameStr,
                    duration: Math.round(r.duration),
                    type: r.initiatorType || 'resource'
                };
            });
        data.diagnostics.slowestResources = slow;
    } catch (e) {}

    // 1.0s observation window
    setTimeout(() => {
        data.metrics.lcp = lcpValue || data.metrics.fcp || data.metrics.ttfb || 0;
        data.metrics.fcp = data.metrics.fcp || (data.metrics.lcp ? Math.min(data.metrics.fcp || data.metrics.lcp, data.metrics.lcp) : data.metrics.ttfb);
        data.metrics.cls = clsValue || 0;
        data.metrics.tbt = totalLongTaskDuration || 0;
        data.diagnostics.longTasksCount = longTaskCount;
        data.diagnostics.maxLongTaskMs = maxLongTask;
        data.diagnostics.shiftCount = shiftCount;

        resolve(data);
    }, 1000);
});
`

type ScanEvalResult struct {
	Metrics     WebVitals       `json:"metrics"`
	Diagnostics PageDiagnostics `json:"diagnostics"`
}

func BuildCategoryDiagnostics(m WebVitals, grades DetailedGrades, diag PageDiagnostics) map[string]CategoryDiagnostic {
	cats := make(map[string]CategoryDiagnostic)

	// 1. TTFB Category
	ttfbDiag := CategoryDiagnostic{
		Category: "ttfb",
		Title:    "TTFB (Час відповіді сервера)",
		Status:   grades.TTFB.Status,
		Summary:  fmt.Sprintf("Початковий час відповіді сервера становить %s.", grades.TTFB.Formatted),
	}
	if diag.ServerProcessing > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("Час обробки на бекенді: %.0fms", diag.ServerProcessing))
	}
	if diag.DNSTime > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("DNS Lookup: %.0fms", diag.DNSTime))
	}
	if diag.TCPTime > 0 {
		ttfbDiag.Details = append(ttfbDiag.Details, fmt.Sprintf("TCP Handshake / TLS: %.0fms", diag.TCPTime))
	}
	if grades.TTFB.Status != "good" {
		ttfbDiag.Fixes = append(ttfbDiag.Fixes,
			"Увімкніть кешування на рівні сервера (OPcache, Redis, Nginx FastCGI Cache або Cloudflare CDN).",
			"Оптимізуйте повільні запити до бази даних (SQL indexes) та скрипти бекенду.",
		)
	} else {
		ttfbDiag.Fixes = append(ttfbDiag.Fixes, "Сервер відповідає швидко. Показник у межах норми.")
	}
	cats["ttfb"] = ttfbDiag

	// 2. FCP Category
	fcpDiag := CategoryDiagnostic{
		Category: "fcp",
		Title:    "FCP (Перше малювання вмісту)",
		Status:   grades.FCP.Status,
		Summary:  fmt.Sprintf("Перший видимий елемент з'являється через %s після запиту.", grades.FCP.Formatted),
	}
	if diag.RenderBlockingCount > 0 {
		fcpDiag.Details = append(fcpDiag.Details, fmt.Sprintf("Виявлено %d стилів/скриптів, що блокують рендеринг", diag.RenderBlockingCount))
		for _, f := range diag.RenderBlockingFiles {
			fcpDiag.Details = append(fcpDiag.Details, fmt.Sprintf("Блокуючий ресурс: %s", f))
		}
	}
	if grades.FCP.Status != "good" {
		fcpDiag.Fixes = append(fcpDiag.Fixes,
			"Додайте атрибути 'defer' або 'async' для всіх некритичних скриптів JavaScript.",
			"Скоротіть обсяг критичного CSS або додайте inline critical CSS для швидкого відображення першого екрану.",
		)
	} else {
		fcpDiag.Fixes = append(fcpDiag.Fixes, "Перше малювання відбувається без затримок.")
	}
	cats["fcp"] = fcpDiag

	// 3. LCP Category
	lcpDiag := CategoryDiagnostic{
		Category: "lcp",
		Title:    "LCP (Найбільший елемент макету)",
		Status:   grades.LCP.Status,
		Summary:  fmt.Sprintf("Найбільший контентний елемент макету відмальовується за %s.", grades.LCP.Formatted),
	}
	if diag.LCPElement != "" {
		lcpDiag.Details = append(lcpDiag.Details, fmt.Sprintf("Головний LCP елемент: %s", diag.LCPElement))
	}
	if diag.LCPUrl != "" {
		lcpDiag.Details = append(lcpDiag.Details, fmt.Sprintf("Ресурс зображення/шрифту: %s", diag.LCPUrl))
	}
	if grades.LCP.Status != "good" {
		lcpDiag.Fixes = append(lcpDiag.Fixes,
			"Конвертуйте зображення у сучасні формати WebP/AVIF та стисніть розмір.",
			"Додайте `<link rel='preload' as='image'>` або атрибут `fetchpriority='high'` для LCP зображення.",
		)
	} else {
		lcpDiag.Fixes = append(fcpDiag.Fixes, "Головний елемент сторінки завантажується оптимально.")
	}
	cats["lcp"] = lcpDiag

	// 4. CLS Category
	clsDiag := CategoryDiagnostic{
		Category: "cls",
		Title:    "CLS (Зсув макету сторінки)",
		Status:   grades.CLS.Status,
		Summary:  fmt.Sprintf("Сумарний зсув елементів макету становить %s.", grades.CLS.Formatted),
	}
	if diag.ShiftCount > 0 {
		clsDiag.Details = append(clsDiag.Details, fmt.Sprintf("Фіксовано %d зафіксованих зсувів елементів під час рендерингу", diag.ShiftCount))
	}
	for _, cause := range diag.ShiftCauses {
		clsDiag.Details = append(clsDiag.Details, cause)
	}
	if grades.CLS.Status != "good" {
		clsDiag.Fixes = append(clsDiag.Fixes,
			"Вкажіть точні атрибути `width` та `height` для всіх <img> та <iframe> елементів.",
			"Зарезервуйте фіксоване місце під динамічні баннери чи віджети через CSS `min-height`.",
		)
	} else {
		clsDiag.Fixes = append(clsDiag.Fixes, "Макет сторінки залишається стабільним без небажаних зсувів.")
	}
	cats["cls"] = clsDiag

	// 5. TBT Category
	tbtDiag := CategoryDiagnostic{
		Category: "tbt",
		Title:    "TBT (Сумарне блокування потоку)",
		Status:   grades.TBT.Status,
		Summary:  fmt.Sprintf("Головний потік JS заблоковано на %s.", grades.TBT.Formatted),
	}
	if diag.LongTasksCount > 0 {
		tbtDiag.Details = append(tbtDiag.Details, fmt.Sprintf("Кількість довготривалих задач (>50ms): %d", diag.LongTasksCount))
	}
	if diag.MaxLongTaskMs > 0 {
		tbtDiag.Details = append(tbtDiag.Details, fmt.Sprintf("Найдовша задача на головному потоці: %.0fms", diag.MaxLongTaskMs))
	}
	if grades.TBT.Status != "good" {
		tbtDiag.Fixes = append(tbtDiag.Fixes,
			"Оптимізуйте та розбийте важкі функції JavaScript за допомогою `requestIdleCallback` або `setTimeout`.",
			"Видаліть або відкладіть завантаження некритичних трекерів та важких JS-бібліотек.",
		)
	} else {
		tbtDiag.Fixes = append(tbtDiag.Fixes, "Головний потік JavaScript вільний для взаємодії з користувачем.")
	}
	cats["tbt"] = tbtDiag

	return cats
}

func GenerateRecommendations(m WebVitals, grades DetailedGrades, diag PageDiagnostics) []string {
	var recs []string

	if grades.TTFB.Status != "good" {
		diagText := ""
		if diag.ServerProcessing > 0 {
			diagText = fmt.Sprintf(" Час обробки сервером: %.0fms, DNS: %.0fms, TCP: %.0fms.", diag.ServerProcessing, diag.DNSTime, diag.TCPTime)
		}
		recs = append(recs, fmt.Sprintf("Високий TTFB (%s): Повільна відповідь сервера.%s Рекомендація: оптимізуйте серверні запити, використайте CDN або HTTP кешування.", grades.TTFB.Formatted, diagText))
	}

	if grades.FCP.Status != "good" {
		recs = append(recs, fmt.Sprintf("Повільний FCP (%s): Перший вміст з'являється із затримкою. Рекомендація: приберіть стилі/скрипти, що блокують рендеринг.", grades.FCP.Formatted))
	}

	if grades.LCP.Status != "good" {
		lcpInfo := ""
		if diag.LCPElement != "" {
			lcpInfo = fmt.Sprintf(" Головний LCP елемент: %s.", diag.LCPElement)
		}
		if diag.LCPUrl != "" {
			lcpInfo += fmt.Sprintf(" Зображення/Ресурс: %s.", diag.LCPUrl)
		}
		recs = append(recs, fmt.Sprintf("Критичний LCP (%s): Найбільший елемент завантажується занадто довго.%s Рекомендація: стисніть зображення (WebP/AVIF), додайте `fetchpriority='high'` або зробіть preload.", grades.LCP.Formatted, lcpInfo))
	}

	if grades.CLS.Status != "good" {
		recs = append(recs, fmt.Sprintf("Високий CLS (%s): Зміщення макету під час завантаження. Рекомендація: додайте чіткі width/height для зображень та баннерів.", grades.CLS.Formatted))
	}

	if grades.TBT.Status != "good" {
		tasksText := ""
		if diag.LongTasksCount > 0 {
			tasksText = fmt.Sprintf(" Виявлено %d довготривалих задач (>50ms) на головному потоці.", diag.LongTasksCount)
		}
		recs = append(recs, fmt.Sprintf("Високий TBT (%s): Заблоковано головний потік JavaScript.%s Рекомендація: розбийте важкі JS-функції та видаліть невикористовуваний JS.", grades.TBT.Formatted, tasksText))
	}

	if len(diag.SlowestResources) > 0 {
		slowText := "Повільні ресурси сторінки: "
		for i, res := range diag.SlowestResources {
			if i > 0 {
				slowText += ", "
			}
			slowText += fmt.Sprintf("%s (%s, %.0fms)", res.Name, res.Type, res.Duration)
		}
		recs = append(recs, slowText)
	}

	if len(recs) == 0 {
		recs = append(recs, "Чудова продуктивність! Усі метрикі Core Web Vitals у межах зеленої зони Google.")
	}

	return recs
}
