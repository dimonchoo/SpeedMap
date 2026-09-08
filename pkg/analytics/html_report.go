package analytics

import (
	"fmt"
	"strings"
)

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

