package wpexport

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
)

// PackageStudioContext holds metadata and images loaded from an existing export package.
type PackageStudioContext struct {
	PackageDir   string                   `json:"packageDir"`
	ManifestPath string                   `json:"manifestPath"`
	Domain       string                   `json:"domain"`
	Generated    string                   `json:"generated"`
	Count        int                      `json:"count"`
	Images       []StudioPackageImageItem `json:"images"`
}

// StudioPackageImageItem represents an image entry from a package's manifest,
// shaped to seamlessly match the frontend Image Studio item structure.
type StudioPackageImageItem struct {
	ID                      string   `json:"id"`
	URL                     string   `json:"url"` // for studio preview loading (maps to sourceUrl / originalPath)
	SourceURL               string   `json:"sourceUrl"`
	Basename                string   `json:"basename"`
	Format                  string   `json:"format"`
	PathHint                string   `json:"pathHint"`
	WebpRel                 string   `json:"webpRel"`
	Pages                   []string `json:"pages"`
	NaturalWidth            int      `json:"naturalWidth"`
	NaturalHeight           int      `json:"naturalHeight"`
	OptimizedWidth          int      `json:"optimizedWidth"`
	OptimizedHeight         int      `json:"optimizedHeight"`
	MaxRenderedWidth        int      `json:"maxRenderedWidth"`
	MaxRenderedHeight       int      `json:"maxRenderedHeight"`
	RecommendedRetinaWidth  int      `json:"recommendedRetinaWidth"`
	RecommendedRetinaHeight int      `json:"recommendedRetinaHeight"`
	OriginalBytes           int64    `json:"originalBytes"`
	OptimizedBytes          int64    `json:"optimizedBytes"`
	Bytes                   int64    `json:"bytes"`          // alias for originalBytes so studio can read img.bytes
	FormattedBytes          string   `json:"formattedBytes"` // alias for originalFormatted
	OriginalFormatted       string   `json:"originalFormatted"`
	OptimizedFormatted      string   `json:"optimizedFormatted"`
	SavingsPercent          float64  `json:"savingsPercent"`
	OptimizedPath           string   `json:"optimizedPath"` // e.g. "images/001/optimized.webp"
	LocalWebPAbsPath        string   `json:"localWebPAbsPath"`
	IsHeavy                 bool     `json:"isHeavy"`
	IsModified              bool     `json:"isModified"`
	Quality                 float32  `json:"quality,omitempty"`
	IsLossless              bool     `json:"isLossless,omitempty"`
}

// TunedSaveResult returns the updated stats after overwriting an image in the package.
type TunedSaveResult struct {
	ID                 string  `json:"id"`
	OptimizedBytes     int64   `json:"optimizedBytes"`
	OptimizedFormatted string  `json:"optimizedFormatted"`
	SavingsPercent     float64 `json:"savingsPercent"`
	OptimizedWidth     int     `json:"optimizedWidth"`
	OptimizedHeight    int     `json:"optimizedHeight"`
	SavedPath          string  `json:"savedPath"`
}

// ResolvePackagePaths resolves packageDir and manifest.json path from an input string.
func ResolvePackagePaths(dirOrManifest string) (string, string, error) {
	p := strings.TrimSpace(dirOrManifest)
	if p == "" {
		return "", "", fmt.Errorf("шлях до папки або маніфесту не може бути порожнім")
	}
	p = filepath.Clean(p)

	var packageDir, manifestPath string
	fi, err := os.Stat(p)
	if err != nil {
		return "", "", fmt.Errorf("помилка доступу до шляху %s: %w", p, err)
	}

	if fi.IsDir() {
		packageDir = p
		manifestPath = filepath.Join(p, "manifest.json")
	} else {
		if strings.ToLower(filepath.Base(p)) == "manifest.json" {
			manifestPath = p
			packageDir = filepath.Dir(p)
		} else {
			return "", "", fmt.Errorf("обраний файл не є manifest.json: %s", p)
		}
	}

	if _, err := os.Stat(manifestPath); err != nil {
		return "", "", fmt.Errorf("manifest.json не знайдено за шляхом %s: %w", manifestPath, err)
	}

	return packageDir, manifestPath, nil
}

// LoadPackageForStudio parses manifest.json from an export directory and returns Studio-ready items.
func LoadPackageForStudio(dirOrManifest string) (*PackageStudioContext, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(dirOrManifest)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("помилка читання %s: %w", manifestPath, err)
	}

	var raw struct {
		Domain    string                   `json:"domain"`
		Generated string                   `json:"generated"`
		Count     int                      `json:"count"`
		Images    []map[string]interface{} `json:"images"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("помилка розбору JSON маніфесту %s: %w", manifestPath, err)
	}

	items := make([]StudioPackageImageItem, 0, len(raw.Images))
	for idx, m := range raw.Images {
		id := getMapString(m, "id")
		if id == "" {
			id = fmt.Sprintf("%03d", idx+1)
		}
		sourceURL := getMapString(m, "sourceUrl")
		if sourceURL == "" {
			sourceURL = getMapString(m, "originalPath")
		}
		basename := getMapString(m, "basename")
		if basename == "" && sourceURL != "" {
			basename = filepath.Base(sourceURL)
		}
		format := strings.ToLower(getMapString(m, "format"))
		if format == "" && basename != "" {
			format = strings.TrimPrefix(filepath.Ext(basename), ".")
		}
		optPath := getMapString(m, "optimizedPath")
		if optPath == "" {
			optFile := "optimized.webp"
			if format == "svg" {
				optFile = "optimized.svg"
			}
			optPath = fmt.Sprintf("images/%s/%s", id, optFile)
		}

		localWebPAbsPath := filepath.Join(packageDir, optPath)
		origBytes := getMapInt64(m, "originalBytes")
		if origBytes == 0 {
			origBytes = getMapInt64(m, "bytes")
		}
		optBytes := getMapInt64(m, "optimizedBytes")
		if optBytes == 0 {
			if fi, err := os.Stat(localWebPAbsPath); err == nil {
				optBytes = fi.Size()
			}
		}

		origFormatted := getMapString(m, "originalFormatted")
		if origFormatted == "" && origBytes > 0 {
			origFormatted = formatBytes(origBytes)
		}
		optFormatted := getMapString(m, "optimizedFormatted")
		if optFormatted == "" && optBytes > 0 {
			optFormatted = formatBytes(optBytes)
		}

		savingsPct := getMapFloat64(m, "savingsPercent")
		if savingsPct == 0 && origBytes > 0 && optBytes > 0 {
			savingsPct = float64(origBytes-optBytes) / float64(origBytes) * 100
		}

		pages := getMapStringSlice(m, "pages")

		items = append(items, StudioPackageImageItem{
			ID:                      id,
			URL:                     sourceURL,
			SourceURL:               sourceURL,
			Basename:                basename,
			Format:                  format,
			PathHint:                getMapString(m, "pathHint"),
			WebpRel:                 getMapString(m, "webpRel"),
			Pages:                   pages,
			NaturalWidth:            getMapInt(m, "naturalWidth"),
			NaturalHeight:           getMapInt(m, "naturalHeight"),
			OptimizedWidth:          getMapInt(m, "optimizedWidth"),
			OptimizedHeight:         getMapInt(m, "optimizedHeight"),
			MaxRenderedWidth:        getMapInt(m, "maxRenderedWidth"),
			MaxRenderedHeight:       getMapInt(m, "maxRenderedHeight"),
			RecommendedRetinaWidth:  getMapInt(m, "recommendedRetinaWidth"),
			RecommendedRetinaHeight: getMapInt(m, "recommendedRetinaHeight"),
			OriginalBytes:           origBytes,
			OptimizedBytes:          optBytes,
			Bytes:                   origBytes,
			FormattedBytes:          origFormatted,
			OriginalFormatted:       origFormatted,
			OptimizedFormatted:      optFormatted,
			SavingsPercent:          savingsPct,
			OptimizedPath:           optPath,
			LocalWebPAbsPath:        localWebPAbsPath,
			IsHeavy:                 origBytes >= 100*1024,
			Quality:                 float32(getMapFloat64(m, "quality")),
			IsLossless:              getMapBool(m, "isLossless"),
			IsModified:              getMapBool(m, "isOverridden"),
		})
	}

	return &PackageStudioContext{
		PackageDir:   packageDir,
		ManifestPath: manifestPath,
		Domain:       raw.Domain,
		Generated:    raw.Generated,
		Count:        len(items),
		Images:       items,
	}, nil
}

// SaveTunedImageToPackage converts the image using tuned options, overwrites the file in
// images/<id>/optimized.webp on disk, and updates manifest.json and compare.html.
func SaveTunedImageToPackage(packageDir, imageID, rawURL string, opts optimizer.ImageTuneOptions, cfg config.ScanConfig) (*TunedSaveResult, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(packageDir)
	if err != nil {
		return nil, err
	}

	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("imageID не може бути порожнім")
	}

	// 1. Fetch original image bytes
	origBytes, err := optimizer.FetchImageBytes(rawURL, cfg.AuthUser, cfg.AuthPass, cfg.GetUserAgent())
	if err != nil {
		return nil, fmt.Errorf("помилка завантаження оригіналу %s: %w", rawURL, err)
	}

	// 2. Convert to WebP / optimized format using tuned options
	convRes, err := optimizer.ConvertImageBytesTuned(rawURL, origBytes, opts)
	if err != nil {
		return nil, fmt.Errorf("помилка оптимізації зображення: %w", err)
	}

	// 3. Determine target filename in images/<id>/
	isVectorSVG := strings.HasPrefix(convRes.OptimizedWebPBase64, "data:image/svg+xml")
	optFilename := "optimized.webp"
	if isVectorSVG {
		optFilename = "optimized.svg"
	}

	targetDir := filepath.Join(packageDir, "images", imageID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("помилка створення директорії %s: %w", targetDir, err)
	}

	idx := strings.Index(convRes.OptimizedWebPBase64, ",")
	if idx == -1 {
		return nil, fmt.Errorf("некоректні base64 дані зображення")
	}
	optData, err := base64.StdEncoding.DecodeString(convRes.OptimizedWebPBase64[idx+1:])
	if err != nil {
		return nil, fmt.Errorf("помилка декодування base64: %w", err)
	}

	targetPath := filepath.Join(targetDir, optFilename)
	if err := os.WriteFile(targetPath, optData, 0644); err != nil {
		return nil, fmt.Errorf("помилка запису файлу %s: %w", targetPath, err)
	}

	// 4. Update manifest.json
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("помилка читання %s: %w", manifestPath, err)
	}

	var rawManifest struct {
		Domain        string                   `json:"domain"`
		Generated     string                   `json:"generated"`
		Count         int                      `json:"count"`
		Quality       int                      `json:"quality,omitempty"`
		WordPressPath string                   `json:"wordpressPath,omitempty"`
		Images        []map[string]interface{} `json:"images"`
	}

	if err := json.Unmarshal(manifestData, &rawManifest); err != nil {
		return nil, fmt.Errorf("помилка парсингу %s: %w", manifestPath, err)
	}

	var updatedEntry map[string]interface{}
	for _, im := range rawManifest.Images {
		if getMapString(im, "id") == imageID || (imageID == "" && getMapString(im, "sourceUrl") == rawURL) {
			im["optimizedBytes"] = convRes.OptimizedBytes
			im["optimizedFormatted"] = convRes.OptimizedFormatted
			im["savingsPercent"] = convRes.SavingsPercent
			im["optimizedWidth"] = convRes.OptimizedWidth
			im["optimizedHeight"] = convRes.OptimizedHeight
			im["quality"] = opts.Quality
			im["isLossless"] = opts.Lossless
			im["isOverridden"] = true
			updatedEntry = im
			break
		}
	}

	updatedManifestJSON, err := json.MarshalIndent(rawManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("помилка серіалізації manifest.json: %w", err)
	}
	if err := os.WriteFile(manifestPath, updatedManifestJSON, 0644); err != nil {
		return nil, fmt.Errorf("помилка збереження manifest.json: %w", err)
	}

	// 5. Update compare.html if present
	comparePath := filepath.Join(packageDir, "compare.html")
	if compareHTMLData, err := os.ReadFile(comparePath); err == nil {
		updatedHTML := syncCompareHTMLItems(string(compareHTMLData), rawManifest.Images)
		_ = os.WriteFile(comparePath, []byte(updatedHTML), 0644)
	}

	res := &TunedSaveResult{
		ID:                 imageID,
		OptimizedBytes:     convRes.OptimizedBytes,
		OptimizedFormatted: convRes.OptimizedFormatted,
		SavingsPercent:     convRes.SavingsPercent,
		OptimizedWidth:     convRes.OptimizedWidth,
		OptimizedHeight:    convRes.OptimizedHeight,
		SavedPath:          targetPath,
	}
	if updatedEntry != nil {
		fmt.Printf("[GO LOG] Tuned image #%s saved to %s (%s, savings: %.1f%%)\n", imageID, targetPath, convRes.OptimizedFormatted, convRes.SavingsPercent)
	}
	return res, nil
}

// OpenPackageCompareHTML opens compare.html in the user's default browser.
func OpenPackageCompareHTML(dirOrManifest string) error {
	packageDir, _, err := ResolvePackagePaths(dirOrManifest)
	if err != nil {
		return err
	}
	comparePath := filepath.Join(packageDir, "compare.html")
	if _, err := os.Stat(comparePath); err != nil {
		return fmt.Errorf("compare.html не знайдено в папці %s", packageDir)
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", comparePath).Start()
	case "windows":
		return exec.Command("explorer", comparePath).Start()
	default:
		return exec.Command("xdg-open", comparePath).Start()
	}
}

// syncCompareHTMLItems replaces the JavaScript `const items = [ ... ];` block in compare.html
// with the freshly updated images array.
func syncCompareHTMLItems(htmlContent string, images []map[string]interface{}) string {
	imagesJSON, err := json.Marshal(images)
	if err != nil {
		return htmlContent
	}

	re := regexp.MustCompile(`(?s)const\s+items\s*=\s*\[.*?\];`)
	replacement := fmt.Sprintf("const items = %s;", string(imagesJSON))
	if re.MatchString(htmlContent) {
		return re.ReplaceAllString(htmlContent, replacement)
	}
	return htmlContent
}

// Helpers for extracting values from map[string]interface{}
func getMapString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getMapInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return 0
}

func getMapInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return 0
}

func getMapFloat64(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok && v != nil {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

func getMapBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok && v != nil {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMapStringSlice(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok && v != nil {
		if arr, ok := v.([]interface{}); ok {
			res := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		}
		if arr, ok := v.([]string); ok {
			return arr
		}
	}
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), units[exp])
}
