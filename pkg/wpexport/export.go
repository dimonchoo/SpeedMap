package wpexport

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
)

//go:embed apply_template.php
var applyTemplate string

//go:embed rollback_template.php
var rollbackTemplate string

type ManifestImage struct {
	SourceURL string   `json:"sourceUrl"`
	PathHint  string   `json:"pathHint"`
	WebpRel   string   `json:"webpRel"`
	Basename  string   `json:"basename"`
	Format    string   `json:"format"`
	IsHeavy   bool     `json:"isHeavy"`
	Bytes     int64    `json:"bytes"`
	Pages     []string `json:"pages"`
}

type Manifest struct {
	Domain        string          `json:"domain"`
	Generated     string          `json:"generated"`
	Quality       int             `json:"quality"`
	WordPressPath string          `json:"wordpressPath"`
	Images        []ManifestImage `json:"images"`
}

// WrittenImage is a converted heavy image with payloads for disk + review ZIP.
type WrittenImage struct {
	ManifestImage
	OrigExt          string
	OrigData         []byte
	WebPData         []byte
	OriginalBytes    int64
	OptimizedBytes   int64
	SavingsPercent   float64
	OriginalFormatted  string
	OptimizedFormatted string
}

// ExportResult paths written under wordpressPath (no SaveFileDialog).
type ExportResult struct {
	ApplyPHP      string `json:"applyPHP"`
	RollbackPHP   string `json:"rollbackPHP"`
	ReviewZIP     string `json:"reviewZIP"`
	WebPCount     int    `json:"webpCount"`
	WordPressPath string `json:"wordpressPath"`
}

var rasterFormats = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "bmp": true,
}

// WP resized file: hero-1024x768.png / hero-465x203.png
var sizeSuffixRE = regexp.MustCompile(`-\d+x\d+(\.[A-Za-z0-9]+)(\?.*)?$`)

// CollectHeavyImages dedupes by lowercase webpRel (same stem .png+.jpg → one entry,
// prefer larger bytes) and keeps only IsHeavy.
func CollectHeavyImages(images []analytics.AggregatedImage) []ManifestImage {
	byKey := make(map[string]int)
	out := make([]ManifestImage, 0)

	for _, img := range images {
		if img.URL == "" {
			continue
		}
		format := strings.ToLower(img.Format)
		if format == "" {
			format = guessFormat(img.URL)
		}
		if format == "svg" || format == "avif" || format == "webp" {
			continue
		}
		if !rasterFormats[format] {
			continue
		}

		sourceURL := preferOriginalURL(img.URL)
		basename, pathHint := basenameAndHint(sourceURL)
		webpRel := webpRelFromHint(pathHint, basename)
		key := strings.ToLower(webpRel)
		if key == "" || key == ".webp" {
			key = strings.ToLower(strings.Split(sourceURL, "?")[0])
		}

		pages := img.Pages
		if pages == nil {
			pages = []string{}
		}

		if idx, ok := byKey[key]; ok {
			ex := &out[idx]
			ex.Pages = mergePages(ex.Pages, pages)
			if img.IsHeavy {
				ex.IsHeavy = true
			}
			// Prefer larger source when same webpRel (png vs jpg collision)
			if img.MaxTransferSize > ex.Bytes {
				ex.Bytes = img.MaxTransferSize
				ex.SourceURL = sourceURL
				ex.PathHint = pathHint
				ex.Basename = basename
				ex.Format = format
				ex.WebpRel = webpRel
			} else if hasSizeSuffix(ex.SourceURL) && !hasSizeSuffix(sourceURL) {
				ex.SourceURL = sourceURL
			}
			continue
		}

		byKey[key] = len(out)
		out = append(out, ManifestImage{
			SourceURL: sourceURL,
			PathHint:  pathHint,
			WebpRel:   webpRel,
			Basename:  basename,
			Format:    format,
			IsHeavy:   img.IsHeavy,
			Bytes:     img.MaxTransferSize,
			Pages:     append([]string(nil), pages...),
		})
	}

	heavy := make([]ManifestImage, 0, len(out))
	for _, im := range out {
		if im.IsHeavy {
			heavy = append(heavy, im)
		}
	}
	return heavy
}

// WriteWebPFiles converts each image via pkg/optimizer and writes under
// {wordpressPath}/wp-content/uploads/{webpRel}. Returns successful writes with payloads.
func WriteWebPFiles(wordpressPath string, images []ManifestImage, quality float32, authUser, authPass string) ([]WrittenImage, int, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return nil, 0, fmt.Errorf("wordpress path is required")
	}
	info, err := os.Stat(wpPath)
	if err != nil || !info.IsDir() {
		return nil, 0, fmt.Errorf("wordpress path must be an existing directory: %s", wpPath)
	}

	uploads := filepath.Join(wpPath, "wp-content", "uploads")
	ok := make([]WrittenImage, 0, len(images))
	written := 0

	for _, img := range images {
		res, err := optimizer.ConvertImageURLToWebPAuth(img.SourceURL, quality, authUser, authPass)
		if err != nil {
			fmt.Printf("[wpexport] skip %s: %v\n", img.SourceURL, err)
			continue
		}
		webpData, err := decodeDataURL(res.OptimizedWebPBase64)
		if err != nil {
			fmt.Printf("[wpexport] skip %s: %v\n", img.SourceURL, err)
			continue
		}
		origData, err := decodeDataURL(res.OriginalDataBase64)
		if err != nil {
			fmt.Printf("[wpexport] skip %s (orig): %v\n", img.SourceURL, err)
			continue
		}

		rel := img.WebpRel
		if rel == "" {
			rel = webpRelFromHint(img.PathHint, img.Basename)
		}
		rel = filepath.ToSlash(rel)
		abs := filepath.Join(uploads, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return ok, written, fmt.Errorf("mkdir %s: %w", filepath.Dir(abs), err)
		}
		if err := os.WriteFile(abs, webpData, 0644); err != nil {
			return ok, written, fmt.Errorf("write %s: %w", abs, err)
		}
		img.WebpRel = rel
		ext := path.Ext(img.Basename)
		if ext == "" {
			ext = "." + img.Format
		}
		ok = append(ok, WrittenImage{
			ManifestImage:      img,
			OrigExt:            strings.TrimPrefix(strings.ToLower(ext), "."),
			OrigData:           origData,
			WebPData:           webpData,
			OriginalBytes:      res.OriginalBytes,
			OptimizedBytes:     res.OptimizedBytes,
			SavingsPercent:     res.SavingsPercent,
			OriginalFormatted:  res.OriginalFormatted,
			OptimizedFormatted: res.OptimizedFormatted,
		})
		written++
	}

	if written == 0 {
		return nil, 0, fmt.Errorf("no WebP files written (check auth / image URLs)")
	}
	return ok, written, nil
}

// BuildReviewZIP packs orig + webp + compare.html + manifest.json for task handoff.
func BuildReviewZIP(domain string, images []WrittenImage) ([]byte, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images for review ZIP")
	}

	type zipEntry struct {
		ID                 string   `json:"id"`
		SourceURL          string   `json:"sourceUrl"`
		PathHint           string   `json:"pathHint"`
		WebpRel            string   `json:"webpRel"`
		Basename           string   `json:"basename"`
		Pages              []string `json:"pages"`
		OriginalBytes      int64    `json:"originalBytes"`
		OptimizedBytes     int64    `json:"optimizedBytes"`
		SavingsPercent     float64  `json:"savingsPercent"`
		OriginalFormatted  string   `json:"originalFormatted"`
		OptimizedFormatted string   `json:"optimizedFormatted"`
		OriginalPath       string   `json:"originalPath"`
		OptimizedPath      string   `json:"optimizedPath"`
	}

	entries := make([]zipEntry, 0, len(images))
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for i, im := range images {
		id := fmt.Sprintf("%03d", i+1)
		ext := im.OrigExt
		if ext == "" {
			ext = "bin"
		}
		origPath := fmt.Sprintf("images/%s/original.%s", id, ext)
		webpPath := fmt.Sprintf("images/%s/optimized.webp", id)

		if err := writeZipFile(zw, origPath, im.OrigData); err != nil {
			_ = zw.Close()
			return nil, err
		}
		if err := writeZipFile(zw, webpPath, im.WebPData); err != nil {
			_ = zw.Close()
			return nil, err
		}

		entries = append(entries, zipEntry{
			ID:                 id,
			SourceURL:          im.SourceURL,
			PathHint:           im.PathHint,
			WebpRel:            im.WebpRel,
			Basename:           im.Basename,
			Pages:              im.Pages,
			OriginalBytes:      im.OriginalBytes,
			OptimizedBytes:     im.OptimizedBytes,
			SavingsPercent:     im.SavingsPercent,
			OriginalFormatted:  im.OriginalFormatted,
			OptimizedFormatted: im.OptimizedFormatted,
			OriginalPath:       origPath,
			OptimizedPath:      webpPath,
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

	var htmlBuf strings.Builder
	htmlBuf.WriteString("<!DOCTYPE html><html><head><meta charset=\"utf-8\"><title>SpeedMap WebP review — ")
	htmlBuf.WriteString(esc(domain))
	htmlBuf.WriteString("</title><style>")
	htmlBuf.WriteString("body{font-family:system-ui,sans-serif;margin:24px;color:#111;background:#fafafa}")
	htmlBuf.WriteString("h1{font-size:1.25rem;margin:0 0 8px}p.meta{color:#555;font-size:13px;margin:0 0 24px}")
	// Stack Before/After vertically so reviewers see each image large, same asset compared top→bottom.
	htmlBuf.WriteString(".pair{display:flex;flex-direction:column;gap:16px;max-width:1200px;margin:0 0 48px;padding:0 0 32px;border-bottom:1px solid #ddd}")
	htmlBuf.WriteString(".pair h2{font-size:1rem;font-weight:600;margin:0;color:#222}")
	htmlBuf.WriteString(".ctx{font-size:13px;color:#444;line-height:1.5}")
	htmlBuf.WriteString(".ctx a{color:#0645ad;word-break:break-all}")
	htmlBuf.WriteString(".ctx .more{color:#666}")
	htmlBuf.WriteString("figure{margin:0}img{display:block;width:100%;max-width:100%;height:auto;background:#eee}")
	htmlBuf.WriteString("figcaption{font-size:13px;color:#444;margin-top:8px} .sav{color:#066}")
	htmlBuf.WriteString("</style></head><body>")
	htmlBuf.WriteString("<h1>SpeedMap WebP review</h1>")
	htmlBuf.WriteString("<p class=\"meta\">")
	htmlBuf.WriteString(esc(domain))
	htmlBuf.WriteString(" · ")
	htmlBuf.WriteString(fmt.Sprintf("%d", len(entries)))
	htmlBuf.WriteString(" images · open this file from the unzipped archive</p>")
	for _, e := range entries {
		htmlBuf.WriteString("<section class=\"pair\">")
		htmlBuf.WriteString("<h2>")
		htmlBuf.WriteString(esc(e.Basename))
		htmlBuf.WriteString("</h2>")
		writeReviewContextHTML(&htmlBuf, e.SourceURL, e.Pages)
		htmlBuf.WriteString("<figure><img src=\"")
		htmlBuf.WriteString(esc(e.OriginalPath))
		htmlBuf.WriteString("\" alt=\"original\"><figcaption>Before · ")
		htmlBuf.WriteString(esc(e.Basename))
		htmlBuf.WriteString(" · ")
		htmlBuf.WriteString(esc(e.OriginalFormatted))
		htmlBuf.WriteString("</figcaption></figure>")
		htmlBuf.WriteString("<figure><img src=\"")
		htmlBuf.WriteString(esc(e.OptimizedPath))
		htmlBuf.WriteString("\" alt=\"webp\"><figcaption>After · WebP · ")
		htmlBuf.WriteString(esc(e.OptimizedFormatted))
		htmlBuf.WriteString(" · <span class=\"sav\">−")
		htmlBuf.WriteString(fmt.Sprintf("%.1f", e.SavingsPercent))
		htmlBuf.WriteString("%</span></figcaption></figure>")
		htmlBuf.WriteString("</section>")
	}
	htmlBuf.WriteString("</body></html>")

	if err := writeZipFile(zw, "compare.html", []byte(htmlBuf.String())); err != nil {
		_ = zw.Close()
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// BuildApplyPHP creates a self-contained WP-CLI eval-file script with embedded manifest.
func BuildApplyPHP(domain string, cfg config.ScanConfig, images []analytics.AggregatedImage, wordpressPath string) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}

	heavy := CollectHeavyImages(images)
	if len(heavy) == 0 {
		return "", fmt.Errorf("no heavy convertible images in scan (raise threshold or rescan)")
	}
	return BuildApplyPHPFromManifest(domain, cfg, wpPath, heavy)
}

// BuildApplyPHPFromManifest embeds an already-filtered image list.
func BuildApplyPHPFromManifest(domain string, cfg config.ScanConfig, wordpressPath string, images []ManifestImage) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}
	if len(images) == 0 {
		return "", fmt.Errorf("no images for WP apply PHP")
	}

	q := int(cfg.NormalizedWebPQuality())
	m := Manifest{
		Domain:        domain,
		Generated:     time.Now().UTC().Format(time.RFC3339),
		Quality:       q,
		WordPressPath: wpPath,
		Images:        images,
	}

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}

	php := strings.Replace(applyTemplate, "{{SPEEDMAP_MANIFEST_JSON}}", string(raw), 1)
	if php == applyTemplate {
		return "", fmt.Errorf("template placeholder {{SPEEDMAP_MANIFEST_JSON}} not found")
	}
	php = strings.ReplaceAll(php, "{{WORDPRESS_PATH}}", wpPath)
	return php, nil
}

// BuildRollbackPHP returns a standalone rollback eval-file script.
func BuildRollbackPHP(wordpressPath string) string {
	return strings.ReplaceAll(rollbackTemplate, "{{WORDPRESS_PATH}}", strings.TrimSpace(wordpressPath))
}

// ManifestImagesFromWritten strips payloads for PHP embedding.
func ManifestImagesFromWritten(written []WrittenImage) []ManifestImage {
	out := make([]ManifestImage, len(written))
	for i, w := range written {
		out[i] = w.ManifestImage
	}
	return out
}

func webpRelFromHint(pathHint, basename string) string {
	base := basename
	if base == "" && pathHint != "" {
		base = path.Base(pathHint)
	}
	name := strings.TrimSuffix(base, path.Ext(base))
	if name == "" {
		name = "image"
	}
	webpName := name + ".webp"
	if pathHint == "" {
		return webpName
	}
	dir := path.Dir(pathHint)
	if dir == "." || dir == "/" {
		return webpName
	}
	return path.Join(dir, webpName)
}

func decodeDataURL(dataURL string) ([]byte, error) {
	idx := strings.Index(dataURL, ",")
	if idx == -1 {
		return nil, fmt.Errorf("invalid data URL")
	}
	return base64.StdEncoding.DecodeString(dataURL[idx+1:])
}

func preferOriginalURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return sizeSuffixRE.ReplaceAllString(rawURL, "$1$2")
	}
	u.Path = sizeSuffixRE.ReplaceAllString(u.Path, "$1")
	return u.String()
}

func hasSizeSuffix(rawURL string) bool {
	pathOnly := strings.Split(rawURL, "?")[0]
	return sizeSuffixRE.MatchString(pathOnly)
}

func mergePages(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, p := range append(a, b...) {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func guessFormat(rawURL string) string {
	u := strings.Split(rawURL, "?")[0]
	ext := strings.ToLower(path.Ext(u))
	return strings.TrimPrefix(ext, ".")
}

func basenameAndHint(rawURL string) (basename, pathHint string) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		base := path.Base(strings.Split(rawURL, "?")[0])
		return stripSizeSuffix(base), ""
	}
	base := stripSizeSuffix(path.Base(u.Path))
	marker := "/wp-content/uploads/"
	if idx := strings.Index(u.Path, marker); idx >= 0 {
		rel := stripSizeSuffix(u.Path[idx+len(marker):])
		pathHint = strings.TrimPrefix(rel, "/")
	}
	return base, pathHint
}

func stripSizeSuffix(name string) string {
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	parts := strings.Split(stem, "-")
	if len(parts) < 2 {
		return name
	}
	last := parts[len(parts)-1]
	if looksLikeWxH(last) {
		return strings.Join(parts[:len(parts)-1], "-") + ext
	}
	return name
}

func looksLikeWxH(s string) bool {
	segs := strings.Split(s, "x")
	if len(segs) != 2 {
		return false
	}
	return isDigits(segs[0]) && isDigits(segs[1])
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func esc(s string) string {
	return html.EscapeString(s)
}

// writeReviewContextHTML adds source-file + sample page links so reviewers can open the live context.
// Caps visible page links (sitewide assets can appear on hundreds of URLs).
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
