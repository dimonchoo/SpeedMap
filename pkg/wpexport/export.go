package wpexport

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
	"time"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
)

//go:embed apply_template.php
var applyTemplate string

type ManifestImage struct {
	SourceURL string   `json:"sourceUrl"`
	PathHint  string   `json:"pathHint"`
	Basename  string   `json:"basename"`
	Format    string   `json:"format"`
	IsHeavy   bool     `json:"isHeavy"`
	Bytes     int64    `json:"bytes"`
	Pages     []string `json:"pages"`
}

type Manifest struct {
	Domain         string          `json:"domain"`
	Generated      string          `json:"generated"`
	Quality        int             `json:"quality"`
	WordPressPath  string          `json:"wordpressPath"`
	Images         []ManifestImage `json:"images"`
}

var rasterFormats = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "bmp": true,
}

// BuildApplyPHP creates a self-contained WP-CLI eval-file script with embedded manifest.
// Same set as images tab "non-webp": raster formats that are not webp/svg/avif.
// wordpressPath is required (e.g. /var/www/site) — baked into header + JSON for --path=.
func BuildApplyPHP(domain string, cfg config.ScanConfig, images []analytics.AggregatedImage, wordpressPath string) (string, error) {
	wpPath := strings.TrimSpace(wordpressPath)
	if wpPath == "" {
		return "", fmt.Errorf("wordpress path is required (e.g. /var/www/site)")
	}

	q := int(cfg.NormalizedWebPQuality())
	m := Manifest{
		Domain:        domain,
		Generated:     time.Now().UTC().Format(time.RFC3339),
		Quality:       q,
		WordPressPath: wpPath,
		Images:        make([]ManifestImage, 0, len(images)),
	}

	seen := make(map[string]bool)
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

		key := strings.Split(img.URL, "?")[0]
		if seen[key] {
			continue
		}
		seen[key] = true

		pages := img.Pages
		if pages == nil {
			pages = []string{}
		}

		basename, pathHint := basenameAndHint(img.URL)
		m.Images = append(m.Images, ManifestImage{
			SourceURL: img.URL,
			PathHint:  pathHint,
			Basename:  basename,
			Format:    format,
			IsHeavy:   img.IsHeavy,
			Bytes:     img.MaxTransferSize,
			Pages:     pages,
		})
	}

	if len(m.Images) == 0 {
		return "", fmt.Errorf("no convertible images in scan (need non-webp raster assets)")
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
