package wpexport

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

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
		Format                  string   `json:"format,omitempty"`
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
		optName := "optimized.webp"
		if strings.ToLower(im.Format) == "svg" && !bytes.HasPrefix(im.WebPData, []byte("RIFF")) {
			optName = "optimized.svg"
		}
		optPath := fmt.Sprintf("images/%s/%s", id, optName)

		if err := writeZipFile(zw, optPath, im.WebPData); err != nil {
			_ = zw.Close()
			return nil, err
		}

		origPreviewURL := im.SourceURL
		if origPreviewURL == "" {
			origPreviewURL = im.Basename
		}

		entries = append(entries, zipEntry{
			ID:                      id,
			SourceURL:               im.SourceURL,
			PathHint:                im.PathHint,
			WebpRel:                 im.WebpRel,
			Basename:                im.Basename,
			Format:                  im.Format,
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
			OriginalPath:            origPreviewURL,
			OptimizedPath:           optPath,
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

	htmlBuf.WriteString("<div style=\"display:flex;align-items:center;gap:12px;margin-bottom:24px;position:sticky;top:0;background:#fafafa;padding:12px 0;z-index:100;border-bottom:1px solid #e2e8f0;\">")
	htmlBuf.WriteString("<input type=\"text\" id=\"compare-search\" placeholder=\"🔍 Пошук по назві або URL (наприклад, waves-1, img-21, hero)...\" oninput=\"filterCompare(this.value)\" style=\"flex:1;max-width:500px;padding:8px 14px;border:1px solid #cbd5e1;border-radius:8px;font-size:13px;outline:none;font-family:inherit;\" />")
	htmlBuf.WriteString(fmt.Sprintf("<span id=\"compare-count\" style=\"font-size:13px;font-weight:bold;color:#475569;\">Показано: %d з %d</span>", len(entries), len(entries)))
	htmlBuf.WriteString("</div>")

	for idx, e := range entries {
		afterFmt := "WebP"
		if strings.ToLower(e.Format) == "svg" {
			afterFmt = "SVG"
		}
		htmlBuf.WriteString("<section class=\"pair\">")
		htmlBuf.WriteString("<h2>")
		htmlBuf.WriteString(esc(e.Basename))
		htmlBuf.WriteString("</h2>")
		writeReviewContextHTML(&htmlBuf, e.SourceURL, e.Pages)
		htmlBuf.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"original\" loading=\"lazy\" onerror=\"this.onerror=null;this.style.opacity='0.4';\" onclick=\"openGallery(%d, 'orig')\"><figcaption>Before · %s · %s</figcaption></figure>", esc(e.OriginalPath), idx, esc(e.Basename), esc(e.OriginalFormatted)))
		htmlBuf.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"optimized\" loading=\"lazy\" onclick=\"openGallery(%d, 'after')\"><figcaption>After · %s · %s · <span class=\"sav\">−%.1f%%</span></figcaption></figure>", esc(e.OptimizedPath), idx, afterFmt, esc(e.OptimizedFormatted), e.SavingsPercent))
		htmlBuf.WriteString("</section>")
	}

	// Fullscreen Gallery Modal DOM
	htmlBuf.WriteString("<div id=\"lightbox\" onclick=\"onBackdropClick(event)\">")
	htmlBuf.WriteString("<div class=\"lb-header\">")
	htmlBuf.WriteString("<div class=\"lb-title-wrap\"><span id=\"lb-count\" style=\"color:#38bdf8;font-weight:bold;font-family:monospace;\"></span><span id=\"lb-title\" class=\"lb-title\"></span><span id=\"lb-mode-badge\" class=\"badge-webp\"></span></div>")
	htmlBuf.WriteString("<div class=\"lb-mode-toggle\">")
	htmlBuf.WriteString("<button id=\"btn-before\" class=\"lb-tab\" onclick=\"toggleMode('orig')\">🔴 Before (Original)</button>")
	htmlBuf.WriteString("<button id=\"btn-after\" class=\"lb-tab lb-tab-active-webp\" onclick=\"toggleMode('after')\">🟢 After (Optimized)</button>")
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
	const isSvg = item.format === 'svg';
	const afterLabel = isSvg ? 'SVG' : 'WebP';
	badgeEl.textContent = isOrig ? '🔴 Before (Original)' : ('🟢 After (' + afterLabel + ')');
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

function filterCompare(val) {
	const q = val.toLowerCase().trim();
	const pairs = document.querySelectorAll('.pair');
	let count = 0;
	pairs.forEach(p => {
		const match = !q || p.innerText.toLowerCase().includes(q);
		p.style.display = match ? 'flex' : 'none';
		if (match) count++;
	});
	document.getElementById('compare-count').innerText = 'Показано: ' + count + ' з ' + pairs.length;
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

// ExportRecord tracks a generated WordPress WebP deploy package on disk
