package wpexport

import (
	"encoding/base64"
	"fmt"
	"html"
	"net/url"
	"path"
	"strings"
)

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
	} else if idx := strings.Index(u.Path, "/wp-content/"); idx >= 0 {
		rel := stripSizeSuffix(u.Path[idx+1:])
		pathHint = strings.TrimPrefix(rel, "/")
	}
	return base, pathHint
}

func targetRelFromHint(pathHint, basename, format string) string {
	base := basename
	if base == "" && pathHint != "" {
		base = path.Base(pathHint)
	}
	ext := strings.ToLower(path.Ext(base))
	if strings.ToLower(format) == "svg" || ext == ".svg" {
		if pathHint != "" {
			return pathHint
		}
		if ext == ".svg" {
			return base
		}
		return base + ".svg"
	}
	return webpRelFromHint(pathHint, basename)
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
