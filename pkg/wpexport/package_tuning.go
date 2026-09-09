package wpexport

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"SpeedMap/pkg/config"
	"SpeedMap/pkg/optimizer"
	_ "golang.org/x/image/webp"
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
	IsCustomReplaced        bool     `json:"isCustomReplaced,omitempty"`
	SourceType              string   `json:"sourceType,omitempty"`
	ReplacedAt              string   `json:"replacedAt,omitempty"`
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

		isCustom := getMapBool(m, "isCustomReplaced")
		srcType := getMapString(m, "sourceType")
		if isCustom || srcType == "custom_file" {
			isCustom = true
			if srcType == "" {
				srcType = "custom_file"
			}
		} else {
			srcType = "remote_url"
		}
		replacedAt := getMapString(m, "replacedAt")

		// If local file exists on disk and its size differs from manifest's optimizedBytes, sync it
		if fi, err := os.Stat(localWebPAbsPath); err == nil && fi.Size() > 0 {
			if optBytes > 0 && fi.Size() != optBytes {
				optBytes = fi.Size()
				optFormatted = formatBytes(optBytes)
				if origBytes > 0 {
					savingsPct = float64(origBytes-optBytes) / float64(origBytes) * 100
				}
				isCustom = true
				srcType = "custom_file"
			}
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
			IsCustomReplaced:        isCustom,
			SourceType:              srcType,
			ReplacedAt:              replacedAt,
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
	updatePackageCompareHTML(packageDir, rawManifest.Images)

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

// OpenPackageCompareHTML opens compare.html in the user's default browser, optionally jumping to a specific item anchor.
func OpenPackageCompareHTML(dirOrManifest string, targetAnchor ...string) error {
	packageDir, _, err := ResolvePackagePaths(dirOrManifest)
	if err != nil {
		return err
	}
	comparePath := filepath.Join(packageDir, "compare.html")
	if _, err := os.Stat(comparePath); err != nil {
		return fmt.Errorf("compare.html не знайдено в папці %s", packageDir)
	}

	target := comparePath
	if len(targetAnchor) > 0 && strings.TrimSpace(targetAnchor[0]) != "" {
		anchor := strings.TrimPrefix(strings.TrimSpace(targetAnchor[0]), "#")
		if !strings.HasPrefix(anchor, "item-") && !strings.HasPrefix(anchor, "row-") {
			anchor = "item-" + anchor
		}
		target = "file://" + filepath.ToSlash(comparePath) + "#" + anchor
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("explorer", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}

// RegeneratePackageCompareHTML reads manifest.json in packageDir and rewrites compare.html with up-to-date markup and scripts
func RegeneratePackageCompareHTML(dirOrManifest string) error {
	packageDir, manifestPath, err := ResolvePackagePaths(dirOrManifest)
	if err != nil {
		return err
	}

	maniData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("помилка читання manifest.json: %w", err)
	}

	var mani struct {
		Domain string           `json:"domain"`
		Images []ReviewZipEntry `json:"images"`
	}
	if err := json.Unmarshal(maniData, &mani); err != nil {
		return fmt.Errorf("помилка парсингу manifest.json: %w", err)
	}

	compareHTML := GenerateCompareHTML(mani.Domain, mani.Images)
	comparePath := filepath.Join(packageDir, "compare.html")
	if err := os.WriteFile(comparePath, []byte(compareHTML), 0644); err != nil {
		return fmt.Errorf("помилка запису compare.html: %w", err)
	}
	return nil
}

func updatePackageCompareHTML(packageDir string, rawImages []map[string]interface{}) {
	if err := RegeneratePackageCompareHTML(packageDir); err != nil {
		comparePath := filepath.Join(packageDir, "compare.html")
		if compareHTMLData, err := os.ReadFile(comparePath); err == nil {
			updatedHTML := syncCompareHTMLItems(string(compareHTMLData), rawImages)
			_ = os.WriteFile(comparePath, []byte(updatedHTML), 0644)
		}
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

// ReplacePackageImageWithFile replaces the package's optimized image with an external file provided by user.
// Supported formats: WebP, PNG, JPG/JPEG, SVG.
// If PNG/JPG is provided, it is converted to high-quality WebP.
func ReplacePackageImageWithFile(packageDir, imageID, sourceFilePath string) (*TunedSaveResult, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(packageDir)
	if err != nil {
		return nil, err
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("imageID не може бути порожнім")
	}
	sourceFilePath = strings.TrimSpace(sourceFilePath)
	if sourceFilePath == "" {
		return nil, fmt.Errorf("шлях до файлу не може бути порожнім")
	}

	sourceData, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("не вдалося прочитати вибраний файл: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(sourceFilePath))
	var optFilename string
	var optData []byte
	var optW, optH int

	if ext == ".svg" || (len(sourceData) > 5 && bytes.Contains(sourceData[:min(len(sourceData), 512)], []byte("<svg"))) {
		optFilename = "optimized.svg"
		optData = sourceData
	} else if ext == ".webp" {
		optFilename = "optimized.webp"
		optData = sourceData
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(optData)); err == nil {
			optW = cfg.Width
			optH = cfg.Height
		}
	} else if ext == ".png" || ext == ".jpg" || ext == ".jpeg" {
		optFilename = "optimized.webp"
		isPng := (ext == ".png")
		convRes, err := optimizer.ConvertImageBytesTuned(filepath.Base(sourceFilePath), sourceData, optimizer.ImageTuneOptions{
			Quality:  85.0,
			Lossless: isPng,
		})
		if err != nil {
			return nil, fmt.Errorf("помилка конвертації файлу в WebP: %w", err)
		}
		idx := strings.Index(convRes.OptimizedWebPBase64, ",")
		if idx == -1 {
			return nil, fmt.Errorf("некоректні base64 дані після конвертації")
		}
		decoded, err := base64.StdEncoding.DecodeString(convRes.OptimizedWebPBase64[idx+1:])
		if err != nil {
			return nil, fmt.Errorf("помилка декодування WebP: %w", err)
		}
		optData = decoded
		optW = convRes.OptimizedWidth
		optH = convRes.OptimizedHeight
	} else {
		return nil, fmt.Errorf("непідтримуваний формат файлу: %s (дозволені webp, png, jpg, svg)", ext)
	}

	targetDir := filepath.Join(packageDir, "images", imageID)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("помилка створення директорії %s: %w", targetDir, err)
	}

	targetPath := filepath.Join(targetDir, optFilename)
	if err := os.WriteFile(targetPath, optData, 0644); err != nil {
		return nil, fmt.Errorf("помилка запису файлу %s: %w", targetPath, err)
	}

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

	optBytes := int64(len(optData))
	optFormatted := formatBytes(optBytes)
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	var savingsPct float64

	for _, im := range rawManifest.Images {
		if getMapString(im, "id") == imageID {
			origBytes := getMapInt64(im, "originalBytes")
			if origBytes == 0 {
				origBytes = getMapInt64(im, "bytes")
			}
			if origBytes > 0 {
				savingsPct = float64(origBytes-optBytes) / float64(origBytes) * 100
			}
			im["optimizedBytes"] = optBytes
			im["optimizedFormatted"] = optFormatted
			im["savingsPercent"] = savingsPct
			if optW > 0 {
				im["optimizedWidth"] = optW
			}
			if optH > 0 {
				im["optimizedHeight"] = optH
			}
			im["optimizedPath"] = fmt.Sprintf("images/%s/%s", imageID, optFilename)
			im["isOverridden"] = true
			im["isCustomReplaced"] = true
			im["sourceType"] = "custom_file"
			im["replacedAt"] = nowStr
			break
		}
	}

	updatedJSON, err := json.MarshalIndent(rawManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("помилка серіалізації manifest.json: %w", err)
	}
	if err := os.WriteFile(manifestPath, updatedJSON, 0644); err != nil {
		return nil, fmt.Errorf("помилка збереження manifest.json: %w", err)
	}

	updatePackageCompareHTML(packageDir, rawManifest.Images)

	return &TunedSaveResult{
		ID:                 imageID,
		OptimizedBytes:     optBytes,
		OptimizedFormatted: optFormatted,
		SavingsPercent:     savingsPct,
		OptimizedWidth:     optW,
		OptimizedHeight:    optH,
		SavedPath:          targetPath,
	}, nil
}

// ReloadPackageImageFromDisk syncs metadata from the file in images/<id>/ if modified outside SpeedMap.
func ReloadPackageImageFromDisk(packageDir, imageID string) (*TunedSaveResult, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(packageDir)
	if err != nil {
		return nil, err
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("imageID не може бути порожнім")
	}

	targetDir := filepath.Join(packageDir, "images", imageID)
	optPath := filepath.Join(targetDir, "optimized.webp")
	if _, err := os.Stat(optPath); os.IsNotExist(err) {
		optPath = filepath.Join(targetDir, "optimized.svg")
		if _, err := os.Stat(optPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("файл optimized.webp/svg не знайдено в папці %s", targetDir)
		}
	}

	data, err := os.ReadFile(optPath)
	if err != nil {
		return nil, fmt.Errorf("не вдалося прочитати %s: %w", optPath, err)
	}

	optBytes := int64(len(data))
	optFormatted := formatBytes(optBytes)
	var optW, optH int
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		optW = cfg.Width
		optH = cfg.Height
	}

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

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	var savingsPct float64

	for _, im := range rawManifest.Images {
		if getMapString(im, "id") == imageID {
			origBytes := getMapInt64(im, "originalBytes")
			if origBytes == 0 {
				origBytes = getMapInt64(im, "bytes")
			}
			if origBytes > 0 {
				savingsPct = float64(origBytes-optBytes) / float64(origBytes) * 100
			}
			im["optimizedBytes"] = optBytes
			im["optimizedFormatted"] = optFormatted
			im["savingsPercent"] = savingsPct
			if optW > 0 {
				im["optimizedWidth"] = optW
			}
			if optH > 0 {
				im["optimizedHeight"] = optH
			}
			im["isOverridden"] = true
			im["isCustomReplaced"] = true
			im["sourceType"] = "custom_file"
			if getMapString(im, "replacedAt") == "" {
				im["replacedAt"] = nowStr
			}
			break
		}
	}

	updatedJSON, err := json.MarshalIndent(rawManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("помилка серіалізації manifest.json: %w", err)
	}
	if err := os.WriteFile(manifestPath, updatedJSON, 0644); err != nil {
		return nil, fmt.Errorf("помилка збереження manifest.json: %w", err)
	}

	updatePackageCompareHTML(packageDir, rawManifest.Images)

	return &TunedSaveResult{
		ID:                 imageID,
		OptimizedBytes:     optBytes,
		OptimizedFormatted: optFormatted,
		SavingsPercent:     savingsPct,
		OptimizedWidth:     optW,
		OptimizedHeight:    optH,
		SavedPath:          optPath,
	}, nil
}

// RevertPackageImageToRemote downloads the original image from the remote URL and re-optimizes it,
// resetting isCustomReplaced and restoring standard remote tracking.
func RevertPackageImageToRemote(packageDir, imageID string, cfg config.ScanConfig) (*TunedSaveResult, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(packageDir)
	if err != nil {
		return nil, err
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("imageID не може бути порожнім")
	}

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

	var targetEntry map[string]interface{}
	for _, im := range rawManifest.Images {
		if getMapString(im, "id") == imageID {
			targetEntry = im
			break
		}
	}
	if targetEntry == nil {
		return nil, fmt.Errorf("зображення #%s не знайдено в manifest.json", imageID)
	}

	rawURL := getMapString(targetEntry, "sourceUrl")
	if rawURL == "" {
		rawURL = getMapString(targetEntry, "originalPath")
	}
	if rawURL == "" {
		return nil, fmt.Errorf("не знайдено sourceUrl для зображення #%s", imageID)
	}

	origBytes, err := optimizer.FetchImageBytes(rawURL, cfg.AuthUser, cfg.AuthPass, cfg.GetUserAgent())
	if err != nil {
		return nil, fmt.Errorf("помилка завантаження оригіналу з %s: %w", rawURL, err)
	}

	q := float32(rawManifest.Quality)
	if q <= 0 {
		q = 80
	}
	convRes, err := optimizer.ConvertImageBytesTuned(rawURL, origBytes, optimizer.ImageTuneOptions{
		Quality: q,
	})
	if err != nil {
		return nil, fmt.Errorf("помилка оптимізації: %w", err)
	}

	isVectorSVG := strings.HasPrefix(convRes.OptimizedWebPBase64, "data:image/svg+xml")
	optFilename := "optimized.webp"
	if isVectorSVG {
		optFilename = "optimized.svg"
	}
	targetDir := filepath.Join(packageDir, "images", imageID)
	_ = os.MkdirAll(targetDir, 0755)

	idx := strings.Index(convRes.OptimizedWebPBase64, ",")
	if idx == -1 {
		return nil, fmt.Errorf("некоректні base64 дані")
	}
	optData, err := base64.StdEncoding.DecodeString(convRes.OptimizedWebPBase64[idx+1:])
	if err != nil {
		return nil, fmt.Errorf("помилка декодування base64: %w", err)
	}
	targetPath := filepath.Join(targetDir, optFilename)
	if err := os.WriteFile(targetPath, optData, 0644); err != nil {
		return nil, fmt.Errorf("помилка запису файлу %s: %w", targetPath, err)
	}

	targetEntry["optimizedBytes"] = convRes.OptimizedBytes
	targetEntry["optimizedFormatted"] = convRes.OptimizedFormatted
	targetEntry["savingsPercent"] = convRes.SavingsPercent
	targetEntry["optimizedWidth"] = convRes.OptimizedWidth
	targetEntry["optimizedHeight"] = convRes.OptimizedHeight
	targetEntry["isOverridden"] = false
	targetEntry["isCustomReplaced"] = false
	targetEntry["sourceType"] = "remote_url"
	delete(targetEntry, "replacedAt")

	updatedJSON, err := json.MarshalIndent(rawManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("помилка серіалізації manifest.json: %w", err)
	}
	if err := os.WriteFile(manifestPath, updatedJSON, 0644); err != nil {
		return nil, fmt.Errorf("помилка збереження manifest.json: %w", err)
	}

	updatePackageCompareHTML(packageDir, rawManifest.Images)

	return &TunedSaveResult{
		ID:                 imageID,
		OptimizedBytes:     convRes.OptimizedBytes,
		OptimizedFormatted: convRes.OptimizedFormatted,
		SavingsPercent:     convRes.SavingsPercent,
		OptimizedWidth:     convRes.OptimizedWidth,
		OptimizedHeight:    convRes.OptimizedHeight,
		SavedPath:          targetPath,
	}, nil
}

// GetPackageImagePreview generates a full ConversionResult for Studio preview using the local package file on disk.
func GetPackageImagePreview(packageDir, imageID string) (*optimizer.ConversionResult, error) {
	packageDir, manifestPath, err := ResolvePackagePaths(packageDir)
	if err != nil {
		return nil, err
	}
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return nil, fmt.Errorf("imageID не може бути порожнім")
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("помилка читання %s: %w", manifestPath, err)
	}

	var rawManifest struct {
		Domain string                   `json:"domain"`
		Images []map[string]interface{} `json:"images"`
	}
	if err := json.Unmarshal(manifestData, &rawManifest); err != nil {
		return nil, fmt.Errorf("помилка парсингу %s: %w", manifestPath, err)
	}

	var targetEntry map[string]interface{}
	for _, im := range rawManifest.Images {
		if getMapString(im, "id") == imageID {
			targetEntry = im
			break
		}
	}
	if targetEntry == nil {
		return nil, fmt.Errorf("зображення #%s не знайдено в manifest.json", imageID)
	}

	targetDir := filepath.Join(packageDir, "images", imageID)
	optPath := filepath.Join(targetDir, "optimized.webp")
	isSVG := false
	if _, err := os.Stat(optPath); os.IsNotExist(err) {
		optPath = filepath.Join(targetDir, "optimized.svg")
		if _, err := os.Stat(optPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("локальний файл зображення не знайдено в %s", targetDir)
		}
		isSVG = true
	}

	data, err := os.ReadFile(optPath)
	if err != nil {
		return nil, fmt.Errorf("помилка читання %s: %w", optPath, err)
	}

	var optW, optH int
	if isSVG {
		// SVG size can be taken from manifest
	} else if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		optW = cfg.Width
		optH = cfg.Height
	}
	if optW == 0 {
		optW = getMapInt(targetEntry, "optimizedWidth")
		if optW == 0 {
			optW = getMapInt(targetEntry, "naturalWidth")
		}
	}
	if optH == 0 {
		optH = getMapInt(targetEntry, "optimizedHeight")
		if optH == 0 {
			optH = getMapInt(targetEntry, "naturalHeight")
		}
	}

	var optBase64 string
	if isSVG {
		optBase64 = fmt.Sprintf("data:image/svg+xml;base64,%s", base64.StdEncoding.EncodeToString(data))
	} else {
		optBase64 = fmt.Sprintf("data:image/webp;base64,%s", base64.StdEncoding.EncodeToString(data))
	}

	origBytes := getMapInt64(targetEntry, "originalBytes")
	if origBytes == 0 {
		origBytes = getMapInt64(targetEntry, "bytes")
	}
	origFormatted := getMapString(targetEntry, "originalFormatted")
	if origFormatted == "" && origBytes > 0 {
		origFormatted = formatBytes(origBytes)
	}
	optBytes := int64(len(data))
	optFormatted := formatBytes(optBytes)
	savingsBytes := origBytes - optBytes
	var savingsPct float64
	if origBytes > 0 {
		savingsPct = float64(savingsBytes) / float64(origBytes) * 100
	}

	rawURL := getMapString(targetEntry, "sourceUrl")
	if rawURL == "" {
		rawURL = getMapString(targetEntry, "originalPath")
	}
	basename := getMapString(targetEntry, "basename")

	return &optimizer.ConversionResult{
		URL:                 rawURL,
		Filename:            basename,
		OriginalWidth:       getMapInt(targetEntry, "naturalWidth"),
		OriginalHeight:      getMapInt(targetEntry, "naturalHeight"),
		OptimizedWidth:      optW,
		OptimizedHeight:     optH,
		OriginalBytes:       origBytes,
		OriginalFormatted:   origFormatted,
		OptimizedBytes:      optBytes,
		OptimizedFormatted:  optFormatted,
		SavingsBytes:        savingsBytes,
		SavingsFormatted:    formatBytes(savingsBytes),
		SavingsPercent:      savingsPct,
		QualityUsed:         float32(getMapFloat64(targetEntry, "quality")),
		IsLossless:          getMapBool(targetEntry, "isLossless"),
		OptimizedWebPBase64: optBase64,
	}, nil
}
