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

type ReviewZipEntry struct {
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

func BuildReviewZIP(domain string, images []WrittenImage) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images for review ZIP")
	}

	entries := make([]ReviewZipEntry, 0, len(images))
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

		entries = append(entries, ReviewZipEntry{
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

		renderReportBuf.WriteString(fmt.Sprintf("<tr id=\"row-%d\"><td><a href=\"#row-%d\" style=\"color:#38bdf8;text-decoration:none;font-weight:bold;\">%d</a></td><td><strong style=\"color:#f1f5f9;font-size:14px;\">%s</strong><br><a href=\"%s\" target=\"_blank\" style=\"color:#64748b;font-size:11px;word-break:break-all;\">%s</a>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><strong style=\"color:#10b981;\">%s</strong>%s</td><td><span class=\"badge-opt\">-%.1f%%</span></td></tr>",
			idx+1, idx+1, idx+1, esc(e.Basename), esc(e.SourceURL), esc(e.SourceURL), previewHTML, pagesHTML.String(), origDimText, rendText, retinaText, esc(e.OriginalFormatted), esc(e.OptimizedFormatted), optDimText, e.SavingsPercent))
	}
	renderReportBuf.WriteString("</tbody></table></div>")

	// Vanilla JS filter & search script
	renderReportBuf.WriteString("<script>")
	renderReportBuf.WriteString("let currentFilter='all';")
	renderReportBuf.WriteString("function setFilter(f,btn){currentFilter=f;document.querySelectorAll('.filter-btn').forEach(b=>b.classList.remove('active'));btn.classList.add('active');filterReport();}")
	renderReportBuf.WriteString("function filterReport(){const q=document.getElementById('searchInput').value.toLowerCase().trim();const rows=document.querySelectorAll('tbody tr');let visible=0;rows.forEach(tr=>{const text=tr.innerText.toLowerCase();const isResized=tr.querySelector('.badge-resized')!==null;const origSizeText=tr.children[6]?.innerText||'';const isHeavy=origSizeText.includes('MB');let mf=true;if(currentFilter==='resized')mf=isResized;else if(currentFilter==='native')mf=!isResized;else if(currentFilter==='heavy')mf=isHeavy;const mq=!q||text.includes(q);if(mf&&mq){tr.style.display='';visible++;}else{tr.style.display='none';}});document.getElementById('visibleCount').innerText=`Відображено: ${visible} з ${rows.length}`;}")
	renderReportBuf.WriteString("function jumpToRow(val){const num=parseInt(val,10);const row=document.getElementById('row-'+num);if(row){row.style.display='';row.scrollIntoView({behavior:'smooth',block:'center'});row.style.outline='2px solid #38bdf8';setTimeout(()=>row.style.outline='',2000);try{history.replaceState(null,'','#row-'+num);}catch(e){}}}")
	renderReportBuf.WriteString("window.addEventListener('DOMContentLoaded',()=>{const rows=document.querySelectorAll('tbody tr');let r=0,n=0,h=0;rows.forEach(tr=>{if(tr.querySelector('.badge-resized'))r++;else n++;if((tr.children[6]?.innerText||'').includes('MB'))h++;});document.getElementById('countAll').innerText=rows.length;document.getElementById('countResized').innerText=r;document.getElementById('countNative').innerText=n;document.getElementById('countHeavy').innerText=h;document.getElementById('visibleCount').innerText=`Відображено: ${rows.length} з ${rows.length}`;const hash=window.location.hash.replace('#','').replace('row-','');if(hash){const n=parseInt(hash,10);if(!isNaN(n)){setTimeout(()=>jumpToRow(n),150);}}});")
	renderReportBuf.WriteString("</script></body></html>")

	if err := writeZipFile(zw, "render-report.html", []byte(renderReportBuf.String())); err != nil {
		_ = zw.Close()
		return nil, err
	}

	compareHTMLStr := GenerateCompareHTML(domain, entries)
	if err := writeZipFile(zw, "compare.html", []byte(compareHTMLStr)); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// formatReviewBytes formats a byte count into a readable string for the review report
func formatReviewBytes(b int64) string {
	if b >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	}
	if b >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%d B", b)
}

// GenerateCompareHTML generates the complete interactive compare.html document
func GenerateCompareHTML(domain string, entries []ReviewZipEntry) string {
	entriesJSON, _ := json.Marshal(entries)
	totalCount := len(entries)

	var totalOrigBytes int64
	var totalOptBytes int64
	for _, e := range entries {
		totalOrigBytes += e.OriginalBytes
		totalOptBytes += e.OptimizedBytes
	}
	savedBytes := totalOrigBytes - totalOptBytes
	overallPct := 0.0
	if totalOrigBytes > 0 {
		overallPct = float64(savedBytes) / float64(totalOrigBytes) * 100.0
	}
	savedFormatted := formatReviewBytes(savedBytes)

	var htmlBuf strings.Builder
	htmlBuf.WriteString("<!DOCTYPE html><html lang=\"uk\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>SpeedMap WebP review — ")
	htmlBuf.WriteString(esc(domain))
	htmlBuf.WriteString("</title><style>")
	htmlBuf.WriteString(`:root {
  --bg-page: #f8fafc;
  --card-bg: #ffffff;
  --border-color: #e2e8f0;
  --text-primary: #0f172a;
  --text-secondary: #475569;
  --text-muted: #94a3b8;
  --primary: #0284c7;
  --primary-hover: #0369a1;
  --success: #10b981;
}
* { box-sizing: border-box; }
body {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  margin: 0; padding: 0; background: var(--bg-page); color: var(--text-primary); -webkit-font-smoothing: antialiased;
}
.page-wrap { width: 100%; max-width: 100%; margin: 0; padding: 14px 16px 80px; }
.page-header { margin-bottom: 16px; padding-bottom: 14px; border-bottom: 1px solid var(--border-color); }
.page-title { font-size: 22px; font-weight: 800; color: var(--text-primary); margin: 0 0 6px; letter-spacing: -0.02em; display: flex; align-items: center; gap: 10px; }
.page-subtitle { font-size: 13px; color: var(--text-secondary); margin: 0; display: flex; align-items: center; flex-wrap: wrap; gap: 10px; }
.stat-pill { display: inline-flex; align-items: center; gap: 5px; background: #ffffff; border: 1px solid var(--border-color); padding: 3px 10px; border-radius: 20px; font-size: 12px; font-weight: 600; color: var(--text-secondary); }
.stat-pill.success { background: #ecfdf5; border-color: #a7f3d0; color: #059669; }

.top-bar-sticky { position: sticky; top: 8px; z-index: 100; background: rgba(255, 255, 255, 0.92); backdrop-filter: blur(14px); -webkit-backdrop-filter: blur(14px); border: 1px solid rgba(226, 232, 240, 0.9); border-radius: 12px; box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.06), 0 2px 6px -1px rgba(0, 0, 0, 0.03); padding: 8px 14px; margin-bottom: 18px; display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; }
.toolbar-group { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.search-input-wrap { position: relative; width: 380px; max-width: 100%; }
.search-input { width: 100%; padding: 7px 12px 7px 32px; background: #ffffff; border: 1px solid var(--border-color); border-radius: 8px; font-size: 13px; outline: none; font-family: inherit; transition: all 0.2s; }
.search-input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgba(2, 132, 199, 0.15); }
.search-icon { position: absolute; left: 10px; top: 50%; transform: translateY(-50%); font-size: 13px; color: var(--text-muted); pointer-events: none; }

.jump-widget { display: inline-flex; align-items: center; gap: 6px; background: #ffffff; border: 1px solid var(--border-color); border-radius: 8px; padding: 3px 6px; }
.jump-label { color: var(--text-secondary); font-weight: 700; font-size: 12px; padding-left: 4px; }
.jump-input { width: 54px; padding: 4px 6px; border: 1px solid var(--border-color); border-radius: 6px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-weight: 700; text-align: center; font-size: 13px; outline: none; transition: border-color 0.2s; }
.jump-input:focus { border-color: var(--primary); }
.btn-action { padding: 5px 12px; border-radius: 6px; font-size: 12px; font-weight: 700; cursor: pointer; transition: all 0.18s; border: 1px solid transparent; display: inline-flex; align-items: center; gap: 4px; }
.btn-preview { background: #0f172a; color: #38bdf8; border-color: #334155; }
.btn-preview:hover { background: #1e293b; border-color: #60a5fa; }
.btn-scroll { background: var(--primary); color: #ffffff; }
.btn-scroll:hover { background: var(--primary-hover); }

.layout-toggle { display: inline-flex; background: #f1f5f9; border: 1px solid var(--border-color); border-radius: 8px; padding: 3px; gap: 2px; }
.layout-btn { background: transparent; border: none; padding: 4px 10px; border-radius: 6px; font-size: 12px; font-weight: 600; color: var(--text-secondary); cursor: pointer; transition: all 0.15s; display: inline-flex; align-items: center; gap: 5px; }
.layout-btn.active { background: #ffffff; color: var(--text-primary); box-shadow: 0 1px 3px rgba(0,0,0,0.08); font-weight: 700; }
.counter-badge { font-size: 12px; font-weight: 700; color: var(--text-secondary); background: #f1f5f9; border: 1px solid var(--border-color); padding: 4px 10px; border-radius: 8px; }

.card-item { background: var(--card-bg); border: 1px solid var(--border-color); border-radius: 12px; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.03); margin-bottom: 18px; overflow: hidden; scroll-margin-top: 80px; transition: box-shadow 0.25s, border-color 0.25s; }
.card-item:hover { border-color: #cbd5e1; box-shadow: 0 6px 16px rgba(0, 0, 0, 0.05); }
.card-item.highlighted { outline: 3px solid var(--primary); border-radius: 12px; box-shadow: 0 0 24px rgba(2, 132, 199, 0.4); transition: all 0.3s; }

.card-header { padding: 10px 16px; background: #ffffff; display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; border-bottom: 1px solid #f1f5f9; }
.card-meta-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.item-badge { display: inline-flex; align-items: center; background: var(--primary); color: #ffffff; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-weight: 700; padding: 2px 8px; border-radius: 6px; text-decoration: none; transition: background 0.18s; }
.item-badge:hover { background: var(--primary-hover); }
.id-badge { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; font-weight: 600; color: #64748b; background: #f1f5f9; padding: 2px 6px; border-radius: 5px; border: 1px solid #e2e8f0; }
.filename { font-size: 13px; font-weight: 700; color: var(--text-primary); word-break: break-all; }
.dim-pill { font-size: 11px; font-weight: 600; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #0369a1; background: #f0f9ff; border: 1px solid #bae6fd; padding: 2px 6px; border-radius: 5px; }

.card-meta-right { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.savings-pill { display: inline-flex; align-items: center; background: #ecfdf5; color: #059669; border: 1px solid #a7f3d0; padding: 2px 8px; border-radius: 20px; font-size: 12px; font-weight: 700; }
.bytes-transition { font-size: 12px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #64748b; }
.btn-card-preview { background: #f8fafc; border: 1px solid var(--border-color); color: var(--primary); font-size: 12px; font-weight: 700; padding: 3px 8px; border-radius: 6px; cursor: pointer; transition: all 0.15s; }
.btn-card-preview:hover { background: #f0f9ff; border-color: #bae6fd; color: var(--primary-hover); }

.card-context { padding: 6px 16px; background: #f8fafc; border-bottom: 1px solid var(--border-color); font-size: 12px; color: #64748b; display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; }
.ctx-url { color: #0284c7; text-decoration: none; word-break: break-all; }
.ctx-url:hover { text-decoration: underline; }
.pages-summary { display: inline-flex; align-items: center; gap: 6px; }
.pages-dropdown-toggle { background: #e2e8f0; border: none; color: #475569; font-size: 11px; font-weight: 600; cursor: pointer; padding: 2px 6px; border-radius: 4px; }
.pages-dropdown-toggle:hover { background: #cbd5e1; }
.pages-expanded { width: 100%; margin-top: 6px; padding-top: 6px; border-top: 1px dashed #cbd5e1; font-size: 11px; line-height: 1.6; display: none; }
.pages-expanded.open { display: block; }

.card-body { padding: 12px 14px 14px; }
.grid-2col { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
@media (max-width: 920px) { .grid-2col { grid-template-columns: 1fr; gap: 14px; } }
.grid-1col { display: grid; grid-template-columns: 1fr; gap: 18px; }

.image-column { display: flex; flex-direction: column; gap: 6px; }
.col-banner { display: flex; align-items: center; justify-content: space-between; font-size: 12px; font-weight: 700; padding: 2px 2px; }
.col-banner-before { color: #dc2626; }
.col-banner-after { color: #059669; }

.img-frame { border: 1px solid var(--border-color); border-radius: 8px; overflow: hidden; background-color: #ffffff; background-image: linear-gradient(45deg, #f1f5f9 25%, transparent 25%), linear-gradient(-45deg, #f1f5f9 25%, transparent 25%), linear-gradient(45deg, transparent 75%, #f1f5f9 75%), linear-gradient(-45deg, transparent 75%, #f1f5f9 75%); background-size: 16px 16px; background-position: 0 0, 0 8px, 8px -8px, -8px 0px; position: relative; cursor: zoom-in; transition: border-color 0.2s, box-shadow 0.2s; }
.img-frame:hover { border-color: #94a3b8; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08); }
.img-frame img { display: block; width: 100%; max-width: 100%; height: auto; transition: transform 0.2s ease; }
.img-frame:hover img { transform: scale(1.006); }

/* Lightbox Styles */
#lightbox { display: none; position: fixed; inset: 0; background: rgba(5, 10, 20, 0.95); backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px); z-index: 99999; flex-direction: column; justify-content: space-between; padding: 16px 24px; box-sizing: border-box; overflow: hidden; }
.lb-header { display: flex; align-items: center; justify-content: space-between; width: 100%; gap: 16px; color: #f1f5f9; }
.lb-title-wrap { display: flex; align-items: center; gap: 10px; min-width: 0; }
.lb-title { font-size: 15px; font-weight: 700; color: #ffffff; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 35vw; }
.lb-jump-wrap { display: inline-flex; align-items: center; gap: 4px; background: #0f172a; border: 1px solid #334155; border-radius: 8px; padding: 3px 8px; font-family: monospace; font-size: 13px; color: #38bdf8; }
.lb-jump-wrap input { width: 54px; background: #020617; border: 1px solid #1e293b; border-radius: 6px; color: #38bdf8; font-weight: 700; font-family: monospace; text-align: center; padding: 2px 4px; font-size: 12px; outline: none; }
.badge-orig { background: rgba(239,68,68,0.2); color: #f87171; border: 1px solid rgba(239,68,68,0.4); padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 700; }
.badge-webp { background: rgba(16,185,129,0.2); color: #34d399; border: 1px solid rgba(16,185,129,0.4); padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 700; }
.lb-mode-toggle { display: flex; background: #1e293b; border-radius: 24px; padding: 3px; border: 1px solid #334155; gap: 4px; }
.lb-tab { background: transparent; border: none; color: #94a3b8; padding: 6px 16px; border-radius: 20px; font-size: 12px; font-weight: 700; cursor: pointer; transition: all 0.2s; }
.lb-tab:hover { color: #ffffff; }
.lb-tab-active-orig { background: #dc2626 !important; color: #ffffff !important; box-shadow: 0 2px 8px rgba(220,38,38,0.4); }
.lb-tab-active-webp { background: #059669 !important; color: #ffffff !important; box-shadow: 0 2px 8px rgba(5,150,105,0.4); }
.lb-goto-btn { background: #1e293b; color: #38bdf8; border: 1px solid #334155; padding: 6px 12px; border-radius: 10px; font-size: 12px; font-weight: 700; cursor: pointer; transition: all 0.2s; }
.lb-goto-btn:hover { background: #0284c7; color: #ffffff; border-color: #38bdf8; }
.lb-close-btn { background: #334155; color: #f1f5f9; border: none; padding: 6px 14px; border-radius: 10px; font-size: 12px; font-weight: bold; cursor: pointer; }
.lb-close-btn:hover { background: #475569; }
.lb-body { position: relative; flex: 1; display: flex; align-items: center; justify-content: center; width: 100%; height: calc(100vh - 130px); margin: 8px 0; }
.lb-main-img { max-height: calc(100vh - 140px); max-width: 88vw; object-fit: contain; border-radius: 8px; box-shadow: 0 20px 40px rgba(0,0,0,0.8); background: transparent; cursor: pointer; }
.lb-nav-btn { position: absolute; top: 50%; transform: translateY(-50%); background: rgba(30,41,59,0.85); color: #ffffff; border: 1px solid #475569; width: 48px; height: 48px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 22px; font-weight: bold; cursor: pointer; transition: all 0.2s; user-select: none; z-index: 10; }
.lb-nav-btn:hover { background: #2563eb; border-color: #60a5fa; transform: translateY(-50%) scale(1.08); }
.lb-prev { left: 16px; }
.lb-next { right: 16px; }
.lb-footer { display: flex; align-items: center; justify-content: space-between; width: 100%; color: #94a3b8; font-size: 12px; font-family: monospace; border-top: 1px solid #1e293b; padding-top: 8px; }
.lb-shortcuts { color: #64748b; font-size: 11px; }
</style></head><body>`)

	htmlBuf.WriteString("<div class=\"page-wrap\">")
	htmlBuf.WriteString("<header class=\"page-header\">")
	htmlBuf.WriteString("<h1 class=\"page-title\"><span>SpeedMap WebP Review</span></h1>")
	htmlBuf.WriteString(fmt.Sprintf("<div class=\"page-subtitle\"><span>%s</span><span class=\"stat-pill\">🖼 %d зображень</span><span class=\"stat-pill success\">⚡ Заощаджено: %s (−%.1f%%)</span><span style=\"color:#94a3b8;font-size:12px;\">(Клацніть на будь-яке зображення для швидкого Before ↔ After)</span></div>", esc(domain), totalCount, esc(savedFormatted), overallPct))
	htmlBuf.WriteString("</header>")

	// Sticky Top Toolbar
	htmlBuf.WriteString("<div class=\"top-bar-sticky\">")
	htmlBuf.WriteString("<div class=\"toolbar-group\"><div class=\"search-input-wrap\"><span class=\"search-icon\">🔍</span>")
	htmlBuf.WriteString("<input type=\"text\" id=\"compare-search\" class=\"search-input\" placeholder=\"Пошук по назві, URL або #номеру (наприклад, #25, hero)...\" oninput=\"filterCompare(this.value)\" />")
	htmlBuf.WriteString("</div></div>")

	htmlBuf.WriteString(fmt.Sprintf("<div class=\"toolbar-group\"><div class=\"jump-widget\" title=\"Швидкий перехід до картки або прев'ю за порядковим номером\"><span class=\"jump-label\">№</span><input type=\"number\" id=\"jump-num-input\" class=\"jump-input\" min=\"1\" max=\"%d\" placeholder=\"1\" onkeydown=\"if(event.key==='Enter'){event.preventDefault();jumpToItem(this.value, true);}\" /><button type=\"button\" class=\"btn-action btn-preview\" onclick=\"jumpToItem(document.getElementById('jump-num-input').value, true)\" title=\"Відкрити інтерактивне повноекранне прев'ю\">🔍 Прев'ю</button><button type=\"button\" class=\"btn-action btn-scroll\" onclick=\"jumpToItem(document.getElementById('jump-num-input').value, false)\" title=\"Прокрутити до картки на сторінці\">⬇ До картки</button></div></div>", totalCount))

	htmlBuf.WriteString("<div class=\"toolbar-group\">")
	htmlBuf.WriteString("<div class=\"layout-toggle\" title=\"Перемикання розташування Before і After\"><button type=\"button\" id=\"btn-layout-2col\" class=\"layout-btn active\" onclick=\"setLayout('2col')\">⫿ Пліч-о-пліч</button><button type=\"button\" id=\"btn-layout-1col\" class=\"layout-btn\" onclick=\"setLayout('1col')\">☰ Стовпчик</button></div>")
	htmlBuf.WriteString(fmt.Sprintf("<span id=\"compare-count\" class=\"counter-badge\">Показано: %d з %d</span>", totalCount, totalCount))
	htmlBuf.WriteString("</div></div>")

	htmlBuf.WriteString("<main id=\"cards-container\">")

	for idx, e := range entries {
		afterFmt := "WebP"
		if strings.ToLower(e.Format) == "svg" {
			afterFmt = "SVG"
		}
		num := idx + 1
		origDimAttr := ""
		if e.NaturalWidth > 0 && e.NaturalHeight > 0 {
			origDimAttr = fmt.Sprintf(" width=\"%d\" height=\"%d\" style=\"aspect-ratio:%d/%d;\"", e.NaturalWidth, e.NaturalHeight, e.NaturalWidth, e.NaturalHeight)
		}
		optDimAttr := origDimAttr
		if e.OptimizedWidth > 0 && e.OptimizedHeight > 0 {
			optDimAttr = fmt.Sprintf(" width=\"%d\" height=\"%d\" style=\"aspect-ratio:%d/%d;\"", e.OptimizedWidth, e.OptimizedHeight, e.OptimizedWidth, e.OptimizedHeight)
		}

		dimText := ""
		if e.NaturalWidth > 0 && e.NaturalHeight > 0 {
			dimText = fmt.Sprintf("<span class=\"dim-pill\">%d × %d px</span>", e.NaturalWidth, e.NaturalHeight)
		}

		htmlBuf.WriteString(fmt.Sprintf("<section class=\"card-item\" id=\"item-%d\" data-index=\"%d\" data-id=\"%s\">", num, idx, esc(e.ID)))
		htmlBuf.WriteString("<div class=\"card-header\">")
		htmlBuf.WriteString(fmt.Sprintf("<div class=\"card-meta-left\"><a href=\"#item-%d\" class=\"item-badge\" title=\"Зображення #%d (клікніть для копіювання посилання)\">#%d</a><span class=\"id-badge\">ID: %s</span><span class=\"filename\">%s</span>%s</div>", num, num, num, esc(e.ID), esc(e.Basename), dimText))
		htmlBuf.WriteString(fmt.Sprintf("<div class=\"card-meta-right\"><span class=\"savings-pill\">−%.1f%%</span><span class=\"bytes-transition\">%s → %s</span><button type=\"button\" class=\"btn-card-preview\" onclick=\"openGallery(%d, 'after')\">🔍 Прев'ю Before/After</button></div>", e.SavingsPercent, esc(e.OriginalFormatted), esc(e.OptimizedFormatted), idx))
		htmlBuf.WriteString("</div>")

		writeReviewContextHTML(&htmlBuf, e.SourceURL, e.Pages, num)

		htmlBuf.WriteString("<div class=\"card-body\"><div class=\"card-images grid-2col\">")
		htmlBuf.WriteString(fmt.Sprintf("<div class=\"image-column\"><div class=\"col-banner col-banner-before\"><span>🔴 Before · Оригінал</span><span style=\"font-family:monospace;font-weight:600;\">%s</span></div><div class=\"img-frame\" onclick=\"openGallery(%d, 'orig')\" title=\"Клацніть для повноекранного перегляду та перемикання\"><img src=\"%s\" alt=\"original\"%s loading=\"lazy\" onerror=\"this.onerror=null;this.style.opacity='0.4';\"></div></div>", esc(e.OriginalFormatted), idx, esc(e.OriginalPath), origDimAttr))
		htmlBuf.WriteString(fmt.Sprintf("<div class=\"image-column\"><div class=\"col-banner col-banner-after\"><span>🟢 After · %s</span><span style=\"font-family:monospace;font-weight:600;\">%s (−%.1f%%)</span></div><div class=\"img-frame\" onclick=\"openGallery(%d, 'after')\" title=\"Клацніть для повноекранного перегляду та перемикання\"><img src=\"%s\" alt=\"optimized\"%s loading=\"lazy\"></div></div>", afterFmt, esc(e.OptimizedFormatted), e.SavingsPercent, idx, esc(e.OptimizedPath), optDimAttr))
		htmlBuf.WriteString("</div></div>")
		htmlBuf.WriteString("</section>")
	}

	htmlBuf.WriteString("</main></div>")

	// Fullscreen Gallery Modal DOM
	htmlBuf.WriteString("<div id=\"lightbox\" onclick=\"onBackdropClick(event)\">")
	htmlBuf.WriteString("<div class=\"lb-header\">")
	htmlBuf.WriteString("<div class=\"lb-title-wrap\">")
	htmlBuf.WriteString(fmt.Sprintf("<div class=\"lb-jump-wrap\" title=\"Введіть номер зображення та натисніть Enter (або клавішу G)\"><span style=\"opacity:0.7;\">#</span><input type=\"number\" id=\"lb-jump-input\" min=\"1\" max=\"%d\" value=\"1\" onchange=\"onLbJumpChange(this.value)\" onkeydown=\"if(event.key==='Enter'){this.blur();}\" /><span style=\"opacity:0.6;\">/</span><span id=\"lb-total-count\">%d</span></div>", totalCount, totalCount))
	htmlBuf.WriteString("<span id=\"lb-id-badge\" style=\"background:#1e293b;color:#94a3b8;border:1px solid #334155;padding:2px 8px;border-radius:6px;font-size:11px;font-family:monospace;font-weight:600;\"></span>")
	htmlBuf.WriteString("<span id=\"lb-title\" class=\"lb-title\"></span>")
	htmlBuf.WriteString("<span id=\"lb-mode-badge\" class=\"badge-webp\"></span>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("<div class=\"lb-mode-toggle\">")
	htmlBuf.WriteString("<button id=\"btn-before\" class=\"lb-tab\" onclick=\"toggleMode('orig')\">🔴 Before</button>")
	htmlBuf.WriteString("<button id=\"btn-after\" class=\"lb-tab lb-tab-active-webp\" onclick=\"toggleMode('after')\">🟢 After</button>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("<div style=\"display:flex;align-items:center;gap:8px;\">")
	htmlBuf.WriteString("<button type=\"button\" class=\"lb-goto-btn\" onclick=\"goToCardFromLb()\" title=\"Закрити прев'ю і перейти до цієї картки\">📍 До картки</button>")
	htmlBuf.WriteString("<button type=\"button\" class=\"lb-close-btn\" onclick=\"closeLb()\">✕ Закрити (Esc)</button>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("</div>")

	htmlBuf.WriteString("<div class=\"lb-body\">")
	htmlBuf.WriteString("<div class=\"lb-nav-btn lb-prev\" onclick=\"prevItem()\" title=\"Попереднє (ArrowLeft)\">‹</div>")
	htmlBuf.WriteString("<img id=\"lb-img\" class=\"lb-main-img\" src=\"\" alt=\"preview\" onclick=\"toggleMode()\" title=\"Клацніть для перемикання Before ↔ After (або Пробіл)\">")
	htmlBuf.WriteString("<div class=\"lb-nav-btn lb-next\" onclick=\"nextItem()\" title=\"Наступне (ArrowRight)\">›</div>")
	htmlBuf.WriteString("</div>")

	htmlBuf.WriteString("<div class=\"lb-footer\">")
	htmlBuf.WriteString("<div id=\"lb-stats\" style=\"color:#e2e8f0;font-weight:bold;\"></div>")
	htmlBuf.WriteString("<div class=\"lb-shortcuts\">Гарячі клавіші: ⬅ ➡ (перегортання) | G (перейти до №) | Пробіл / 1 / 2 (Before ↔ After) | Esc (вихід)</div>")
	htmlBuf.WriteString("</div>")
	htmlBuf.WriteString("</div>")

	// Scripts
	htmlBuf.WriteString("<script>")
	htmlBuf.WriteString("const items = ")
	htmlBuf.Write(entriesJSON)
	htmlBuf.WriteString(";\n")
	htmlBuf.WriteString("let currentIndex = 0;\n")
	htmlBuf.WriteString("let currentMode = 'webp';\n")
	htmlBuf.WriteString("let currentLayout = '2col';\n")
	htmlBuf.WriteString(`
function setLayout(layout) {
	currentLayout = layout;
	const containers = document.querySelectorAll('.card-images');
	const btn2 = document.getElementById('btn-layout-2col');
	const btn1 = document.getElementById('btn-layout-1col');
	if (layout === '1col') {
		containers.forEach(c => {
			c.classList.remove('grid-2col');
			c.classList.add('grid-1col');
		});
		if (btn1) btn1.classList.add('active');
		if (btn2) btn2.classList.remove('active');
	} else {
		containers.forEach(c => {
			c.classList.remove('grid-1col');
			c.classList.add('grid-2col');
		});
		if (btn2) btn2.classList.add('active');
		if (btn1) btn1.classList.remove('active');
	}
}

function togglePages(id) {
	const el = document.getElementById(id);
	if (el) el.classList.toggle('open');
}

function openGallery(idx, mode) {
	currentIndex = idx;
	currentMode = mode || 'webp';
	renderGallery();
	document.getElementById('lightbox').style.display = 'flex';
	document.body.style.overflow = 'hidden';
}

function renderGallery() {
	if (currentIndex < 0) currentIndex = 0;
	if (currentIndex >= items.length) currentIndex = items.length - 1;
	const item = items[currentIndex];
	const isOrig = currentMode === 'orig';
	const imgEl = document.getElementById('lb-img');
	imgEl.src = isOrig ? item.originalPath : item.optimizedPath;
	
	document.getElementById('lb-title').textContent = item.basename;
	const jumpInp = document.getElementById('lb-jump-input');
	if (jumpInp && document.activeElement !== jumpInp) jumpInp.value = currentIndex + 1;
	
	document.getElementById('lb-total-count').textContent = items.length;
	document.getElementById('lb-id-badge').textContent = 'ID: ' + item.id;
	
	const badgeEl = document.getElementById('lb-mode-badge');
	const afterLabel = item.format === 'svg' ? 'SVG' : 'WebP';
	badgeEl.textContent = isOrig ? '🔴 Before' : ('🟢 After (' + afterLabel + ')');
	badgeEl.className = isOrig ? 'badge-orig' : 'badge-webp';
	
	document.getElementById('btn-before').className = isOrig ? 'lb-tab lb-tab-active-orig' : 'lb-tab';
	document.getElementById('btn-after').className = !isOrig ? 'lb-tab lb-tab-active-webp' : 'lb-tab';
	
	document.getElementById('lb-stats').textContent = 'Before: ' + item.originalFormatted + ' ➜ After: ' + item.optimizedFormatted + ' (' + item.savingsPercent.toFixed(1) + '%)';
}

function onLbJumpChange(val) {
	const num = parseInt(val, 10);
	if (!isNaN(num) && num >= 1 && num <= items.length) { currentIndex = num - 1; renderGallery(); }
}

function nextItem() { if (currentIndex < items.length - 1) { currentIndex++; renderGallery(); } }
function prevItem() { if (currentIndex > 0) { currentIndex--; renderGallery(); } }
function toggleMode(mode) { currentMode = mode || (currentMode === 'orig' ? 'webp' : 'orig'); renderGallery(); }
function closeLb() { document.getElementById('lightbox').style.display = 'none'; document.body.style.overflow = ''; }
function goToCardFromLb() { closeLb(); jumpToItem(currentIndex + 1, false); }

function jumpToItem(val, openPreview) {
	const num = parseInt(val, 10);
	if (isNaN(num) || num < 1 || num > items.length) return;
	if (openPreview) { openGallery(num - 1, currentMode); return; }
	
	const target = document.getElementById('item-' + num);
	if (!target) return;
	if (target.style.display === 'none') {
		const s = document.getElementById('compare-search');
		if (s) { s.value = ''; filterCompare(''); }
	}
	const topBar = document.querySelector('.top-bar-sticky');
	const navHeight = topBar ? topBar.offsetHeight : 60;
	window.scrollTo({ top: target.getBoundingClientRect().top + window.pageYOffset - navHeight - 16, behavior: 'smooth' });
	target.classList.add('highlighted');
	setTimeout(() => target.classList.remove('highlighted'), 2200);
}

function onBackdropClick(e) { if (e.target.id === 'lightbox') closeLb(); }

function filterCompare(val) {
	const term = (val || '').toLowerCase().trim();
	const sections = document.querySelectorAll('.card-item');
	let shown = 0;
	let numSearch = null;
	if (term.startsWith('#')) numSearch = parseInt(term.slice(1), 10);
	else if (/^\d+$/.test(term)) numSearch = parseInt(term, 10);

	sections.forEach((sec, idx) => {
		const num = idx + 1;
		let match = (numSearch !== null) ? (num === numSearch) : (!term || sec.innerText.toLowerCase().includes(term) || (sec.getAttribute('data-id') || '').toLowerCase().includes(term));
		sec.style.display = match ? '' : 'none';
		if (match) shown++;
	});
	const countEl = document.getElementById('compare-count');
	if (countEl) countEl.innerText = 'Показано: ' + shown + ' з ' + sections.length;
}

window.addEventListener('keydown', (e) => {
	const lb = document.getElementById('lightbox');
	const lbOpen = lb && lb.style.display === 'flex';
	const tag = (e.target.tagName || '').toLowerCase();
	if (tag === 'input' || tag === 'textarea') return;

	if (lbOpen) {
		if (e.key === 'Escape') closeLb();
		else if (e.key === 'ArrowLeft') prevItem();
		else if (e.key === 'ArrowRight') nextItem();
		else if (e.key === ' ' || e.code === 'Space' || e.key === '1' || e.key === '2') { e.preventDefault(); toggleMode(); }
		else if (e.key === 'g' || e.key === 'G') { e.preventDefault(); const inp = document.getElementById('lb-jump-input'); if (inp) { inp.focus(); inp.select(); } }
	} else {
		if (e.key === 'g' || e.key === 'G') { e.preventDefault(); const inp = document.getElementById('jump-num-input'); if (inp) { inp.focus(); inp.select(); } }
	}
});

window.addEventListener('DOMContentLoaded', () => {
	const hash = window.location.hash.replace('#', '').replace('item-', '');
	if (hash) {
		const num = parseInt(hash, 10);
		if (!isNaN(num)) setTimeout(() => jumpToItem(num, false), 150);
	}
});
</script>
</body></html>`)

	return htmlBuf.String()
}

func writeReviewContextHTML(b *strings.Builder, sourceURL string, pages []string, num int) {
	b.WriteString(`<div class="card-context ctx">`)
	b.WriteString(`<div style="min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;max-width:60%;">`)
	if strings.TrimSpace(sourceURL) != "" {
		b.WriteString(`<span>File: </span><a href="`)
		b.WriteString(esc(sourceURL))
		b.WriteString(`" target="_blank" rel="noopener" class="ctx-url">`)
		b.WriteString(esc(sourceURL))
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)

	ordered := prioritizeReviewPages(pages)
	if len(ordered) > 0 {
		b.WriteString(`<div class="pages-summary"><span>Зустрічається на `)
		b.WriteString(fmt.Sprintf("%d", len(ordered)))
		b.WriteString(` стор.:</span> `)
		firstCount := 2
		if len(ordered) < firstCount {
			firstCount = len(ordered)
		}
		for i := 0; i < firstCount; i++ {
			if i > 0 {
				b.WriteString(` · `)
			}
			b.WriteString(`<a href="`)
			b.WriteString(esc(ordered[i]))
			b.WriteString(`" target="_blank" rel="noopener" class="ctx-url">`)
			b.WriteString(esc(shortPageLabel(ordered[i])))
			b.WriteString(`</a>`)
		}
		if len(ordered) > firstCount {
			b.WriteString(fmt.Sprintf(` <button type="button" class="pages-dropdown-toggle" onclick="togglePages('pages-%d')">+ ще %d ▾</button></div>`, num, len(ordered)-firstCount))
			b.WriteString(fmt.Sprintf(`<div id="pages-%d" class="pages-expanded">`, num))
			maxDrop := 20
			if len(ordered) < maxDrop {
				maxDrop = len(ordered)
			}
			for i := firstCount; i < maxDrop; i++ {
				if i > firstCount {
					b.WriteString(`<br>`)
				}
				b.WriteString(`<a href="`)
				b.WriteString(esc(ordered[i]))
				b.WriteString(`" target="_blank" rel="noopener" class="ctx-url">`)
				b.WriteString(esc(ordered[i]))
				b.WriteString(`</a>`)
			}
			if len(ordered) > maxDrop {
				b.WriteString(fmt.Sprintf(`<br><span style="color:#94a3b8;">... та ще %d сторінок</span>`, len(ordered)-maxDrop))
			}
			b.WriteString(`</div>`)
		} else {
			b.WriteString(`</div>`)
		}
	}
	b.WriteString(`</div>`)
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
