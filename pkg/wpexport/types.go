package wpexport

import (
	"regexp"
	"time"
)

type ManifestImage struct {
	SourceURL              string   `json:"sourceUrl"`
	PathHint               string   `json:"pathHint"`
	WebpRel                string   `json:"webpRel"`
	Basename               string   `json:"basename"`
	Format                 string   `json:"format"`
	IsHeavy                bool     `json:"isHeavy"`
	Bytes                  int64    `json:"bytes"`
	Pages                  []string `json:"pages"`
	ID                     string   `json:"id,omitempty"`
	PackageWebP            string   `json:"packageWebp,omitempty"` // relative to apply.php, e.g. images/001/optimized.webp
	NaturalWidth           int      `json:"naturalWidth,omitempty"`
	NaturalHeight          int      `json:"naturalHeight,omitempty"`
	MaxRenderedWidth        int      `json:"maxRenderedWidth,omitempty"`
	MaxRenderedHeight       int      `json:"maxRenderedHeight,omitempty"`
	RecommendedRetinaWidth  int      `json:"recommendedRetinaWidth,omitempty"`
	RecommendedRetinaHeight int      `json:"recommendedRetinaHeight,omitempty"`
	Quality                 float32  `json:"quality,omitempty"`
	IsLossless              bool     `json:"isLossless,omitempty"`
	IsOverridden            bool     `json:"isOverridden,omitempty"`
	IsCustomReplaced        bool     `json:"isCustomReplaced,omitempty"`
	SourceType              string   `json:"sourceType,omitempty"`
	ReplacedAt              string   `json:"replacedAt,omitempty"`
}

// ImageOverride specifies per-image customization applied in Image Studio.
type ImageOverride struct {
	Quality  float32 `json:"quality"`
	Lossless bool    `json:"lossless"`
	Exact    bool    `json:"exact"`
	Retina   bool    `json:"retina"`
	MaxW     int     `json:"maxW"`
	MaxH     int     `json:"maxH"`
	Dither   bool    `json:"dither"`
	Skip     bool    `json:"skip"`
	Approved bool    `json:"approved"`
}

type Manifest struct {
	Domain        string          `json:"domain"`
	Generated     string          `json:"generated"`
	Quality       int             `json:"quality"`
	WordPressPath string          `json:"wordpressPath"`
	Images        []ManifestImage `json:"images"`
}

// WrittenImage is a converted heavy image with payloads for package + review ZIP.
type WrittenImage struct {
	ManifestImage
	OrigExt            string
	OrigData           []byte
	WebPData           []byte
	OptimizedWidth     int
	OptimizedHeight    int
	OriginalBytes      int64
	OptimizedBytes     int64
	SavingsPercent     float64
	OriginalFormatted  string
	OptimizedFormatted string
}

// ExportResult paths for the deploy package (not WP uploads).
type ExportResult struct {
	ApplyPHP      string `json:"applyPHP"`
	RollbackPHP   string `json:"rollbackPHP"`
	ReviewZIP     string `json:"reviewZIP"`
	PackageDir    string `json:"packageDir"`
	WebPCount     int    `json:"webpCount"`
	WordPressPath string `json:"wordpressPath"`
}

var rasterFormats = map[string]bool{
	"png": true, "jpg": true, "jpeg": true, "gif": true, "bmp": true,
}

// WP resized file: hero-1024x768.png / hero-465x203.png
var sizeSuffixRE = regexp.MustCompile(`-\d+x\d+(\.[A-Za-z0-9]+)(\?.*)?$`)

type ExportRecord struct {
	ID             string    `json:"id"`
	Domain         string    `json:"domain"`
	Timestamp      time.Time `json:"timestamp"`
	FormattedTime  string    `json:"formattedTime"`
	PackageDir     string    `json:"packageDir"`
	ManifestPath   string    `json:"manifestPath"`
	ReviewZIP      string    `json:"reviewZip"`
	ApplyPHP       string    `json:"applyPhp"`
	RollbackPHP    string    `json:"rollbackPhp"`
	CompareHTML    string    `json:"compareHtml"`
	RenderReport   string    `json:"renderReport"`
	ImageCount     int       `json:"imageCount"`
	OriginalBytes  int64     `json:"originalBytes"`
	OptimizedBytes int64     `json:"optimizedBytes"`
	SavingsPercent float64   `json:"savingsPercent"`
	ExistsOnDisk   bool      `json:"existsOnDisk"`
}

type ExportFileDiff struct {
	SourceURL         string   `json:"sourceUrl"`
	Basename          string   `json:"basename"`
	OriginalBytes     int64    `json:"originalBytes"`
	OriginalFormatted string   `json:"originalFormatted"`
	BaseWebPBytes     int64    `json:"baseWebpBytes"`
	BaseWebPFormatted string   `json:"baseWebpFormatted"`
	CurrWebPBytes     int64    `json:"currWebpBytes"`
	CurrWebPFormatted string   `json:"currWebpFormatted"`
	DeltaBytes        int64    `json:"deltaBytes"`
	DeltaFormatted    string   `json:"deltaFormatted"`
	Status            string   `json:"status"` // "degraded", "improved", "same", "new", "removed"
	Pages             []string `json:"pages,omitempty"`
}

type ExportDiffReport struct {
	BasePackageDir    string           `json:"basePackageDir"`
	BaseTime          string           `json:"baseTime"`
	CurrentPackageDir string           `json:"currentPackageDir"`
	CurrentTime       string           `json:"currentTime"`
	TotalFiles        int              `json:"totalFiles"`
	DegradedCount     int              `json:"degradedCount"`
	ImprovedCount     int              `json:"improvedCount"`
	SameCount         int              `json:"sameCount"`
	NewCount          int              `json:"newCount"`
	RemovedCount      int              `json:"removedCount"`
	BaseTotalWebP     int64            `json:"baseTotalWebp"`
	CurrentTotalWebP  int64            `json:"currentTotalWebp"`
	DeltaTotalWebP    int64            `json:"deltaTotalWebp"`
	Files             []ExportFileDiff `json:"files"`
}

