package analytics

import (
	"SpeedMap/pkg/scanner"
)

type ResourceImpact struct {
	Name            string  `json:"name"`
	Type            string  `json:"type"`
	Occurrences     int     `json:"occurrences"`
	AvgDurationMs   float64 `json:"avgDurationMs"`
	TotalDurationMs float64 `json:"totalDurationMs"`
}

type AggregatedImage struct {
	URL                       string   `json:"url"`
	Basename                  string   `json:"basename"`        // e.g. "sas-logo.svg"
	MaxTransferSize           int64    `json:"maxTransferSize"` // bytes
	FormattedSize             string   `json:"formattedSize"`   // e.g. "1.2 MB"
	AvgDurationMs             float64  `json:"avgDurationMs"`   // ms
	PageCount                 int      `json:"pageCount"` // how many pages use this image
	Pages                     []string `json:"pages"`     // page URLs where this image was seen
	Width                     int      `json:"width"`     // intrinsic / natural width
	Height                    int      `json:"height"`    // intrinsic / natural height
	NaturalWidth              int      `json:"naturalWidth"`
	NaturalHeight             int      `json:"naturalHeight"`
	MaxRenderedWidth          int      `json:"maxRenderedWidth"`        // max CSS display width across all pages
	MaxRenderedHeight         int      `json:"maxRenderedHeight"`       // max CSS display height across all pages
	RecommendedRetinaWidth    int      `json:"recommendedRetinaWidth"`  // MaxRenderedWidth * 2 (optimal for Retina)
	RecommendedRetinaHeight   int      `json:"recommendedRetinaHeight"` // MaxRenderedHeight * 2
	IsOversized               bool     `json:"isOversized"`             // NaturalWidth > MaxRenderedWidth * 2
	Format                    string   `json:"format"`                  // png, jpg, webp, svg, avif, gif
	IsHeavy                   bool     `json:"isHeavy"`                 // > threshold
	IsLazy                    bool     `json:"isLazy"`
	IsLCP                     bool     `json:"isLCP"`
	EstimatedWebPSize         int64    `json:"estimatedWebPSize"`
	EstimatedWebPFormatted    string   `json:"estimatedWebPFormatted"`
	EstimatedSavingsBytes     int64    `json:"estimatedSavingsBytes"`
	EstimatedSavingsFormatted string   `json:"estimatedSavingsFormatted"`
	EstimatedSavingsPercent   float64  `json:"estimatedSavingsPercent"`
}

type AggregatedFont struct {
	Family        string   `json:"family"`
	URL           string   `json:"url"`
	Type          string   `json:"type"`
	Occurrences   int      `json:"occurrences"`   // how many pages use this font
	Percentage    float64  `json:"percentage"`    // % of total scanned pages
	AvgDurationMs float64  `json:"avgDurationMs"` // ms
	TransferSize  int64    `json:"transferSize"`
	FormattedSize string   `json:"formattedSize"`
	PageURLs      []string `json:"pageUrls"`
}

// AggregatedIframe groups the same iframe src across scanned pages.
// MissedCount = pages where iframe was in DOM but did not load during the scan window.
type AggregatedIframe struct {
	Src             string   `json:"src"`
	Title           string   `json:"title"`
	PageCount       int      `json:"pageCount"`
	Pages           []string `json:"pages"`
	Occurrences     int      `json:"occurrences"`
	LoadedCount     int      `json:"loadedCount"`
	MissedCount     int      `json:"missedCount"`
	IsLazy          bool     `json:"isLazy"`
	AvgDurationMs   float64  `json:"avgDurationMs"`
	MaxTransferSize int64    `json:"maxTransferSize"`
	FormattedSize   string   `json:"formattedSize"`
	Width           int      `json:"width"`
	Height          int      `json:"height"`
}

// AggregatedForm groups unique forms across scanned pages
type AggregatedForm struct {
	ID               string                    `json:"id"`
	Title            string                    `json:"title"`
	Engine           string                    `json:"engine"`
	Method           string                    `json:"method"`
	Action           string                    `json:"action"`
	PageCount        int                       `json:"pageCount"`
	Pages            []string                  `json:"pages"`
	Fields           []scanner.FormFieldDetail `json:"fields"`
	FieldCount       int                       `json:"fieldCount"`
	HasFileUpload    bool                      `json:"hasFileUpload"`
	AllowedFileTypes string                    `json:"allowedFileTypes,omitempty"`
	Captcha          scanner.CaptchaDetail     `json:"captcha"`
	HiddenTokens     map[string]string         `json:"hiddenTokens,omitempty"`
}

type SiteAnalytics struct {
	TotalPages  scannedCount `json:"totalPages"`
	HealthScore int          `json:"healthScore"` // 0 - 100

	StatusCounts map[string]int `json:"statusCounts"` // good, needs-improvement, poor, error

	AverageMetrics map[string]float64 `json:"averageMetrics"` // TTFB, FCP, LCP, CLS, TBT

	TopResourceBottlenecks []ResourceImpact   `json:"topResourceBottlenecks"`
	LargestImages          []AggregatedImage  `json:"largestImages"`
	AllImages              []AggregatedImage  `json:"allImages"`
	FontUsage              []AggregatedFont   `json:"fontUsage"`
	Iframes                []AggregatedIframe `json:"iframes"`
	Forms                  []AggregatedForm   `json:"forms"`
	GlobalFixes            []string           `json:"globalFixes"`

	// Image Optimization Analytics (SEOAEO-235)
	TotalImagePayloadBytes     int64          `json:"totalImagePayloadBytes"`
	TotalImagePayloadFormatted string         `json:"totalImagePayloadFormatted"`
	TotalImageCount            int            `json:"totalImageCount"`
	HeavyImagesCount           int            `json:"heavyImagesCount"`
	OversizedImagesCount       int            `json:"oversizedImagesCount"`
	NonWebPCount               int            `json:"nonWebPCount"`
	SVGCount                   int            `json:"svgCount"`
	MissingLazyCount           int            `json:"missingLazyCount"`
	TotalWebPSavingsBytes      int64          `json:"totalWebPSavingsBytes"`
	TotalWebPSavingsFormatted  string         `json:"totalWebPSavingsFormatted"`
	FormatBreakdown            map[string]int `json:"formatBreakdown"`

	// Iframe audit
	TotalIframeCount  int `json:"totalIframeCount"`
	MissedIframeCount int `json:"missedIframeCount"`
	LoadedIframeCount int `json:"loadedIframeCount"`

	// Form audit
	TotalFormsCount       int            `json:"totalFormsCount"`
	PagesWithFormsCount   int            `json:"pagesWithFormsCount"`
	CaptchaProtectedCount int            `json:"captchaProtectedCount"`
	UnprotectedFormsCount int            `json:"unprotectedFormsCount"`
	FileUploadFormsCount  int            `json:"fileUploadFormsCount"`
	FormEngineBreakdown   map[string]int `json:"formEngineBreakdown"`

	// DOM Virtualization Global Audit
	TotalDOMNodesAcrossPages int      `json:"totalDomNodesAcrossPages"`
	AverageDOMNodesPerPage   int      `json:"averageDomNodesPerPage"`
	HeavyDOMPagesCount       int      `json:"heavyDomPagesCount"`
	GlobalCandidateSelectors []string `json:"globalCandidateSelectors"`
	GlobalVirtualizationCSS  string   `json:"globalVirtualizationCss"`
	GlobalVirtualizationPHP  string   `json:"globalVirtualizationPhp"`
}

type scannedCount = int
