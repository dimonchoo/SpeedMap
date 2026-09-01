package scanner

import (
	"fmt"
)

type WebVitals struct {
	TTFB float64 `json:"ttfb"` // ms
	FCP  float64 `json:"fcp"`  // ms
	LCP  float64 `json:"lcp"`  // ms
	CLS  float64 `json:"cls"`  // unitless score
	TBT  float64 `json:"tbt"`  // ms
}

type MetricGrade struct {
	Value     float64 `json:"value"`
	Formatted string  `json:"formatted"`
	Status    string  `json:"status"` // "good", "needs-improvement", "poor"
}

type DetailedGrades struct {
	TTFB MetricGrade `json:"ttfb"`
	FCP  MetricGrade `json:"fcp"`
	LCP  MetricGrade `json:"lcp"`
	CLS  MetricGrade `json:"cls"`
	TBT  MetricGrade `json:"tbt"`
}

type ResourceTiming struct {
	Name     string  `json:"name"`
	Duration float64 `json:"duration"` // ms
	Type     string  `json:"type"`     // script, img, css, fetch
}

type ImageDetail struct {
	URL            string  `json:"url"`
	TransferSize   int64   `json:"transferSize"`   // bytes
	EncodedSize    int64   `json:"encodedSize"`    // bytes
	Duration       float64 `json:"duration"`       // ms
	Width          int     `json:"width"`          // px (intrinsic / natural width)
	Height         int     `json:"height"`         // px (intrinsic / natural height)
	NaturalWidth   int     `json:"naturalWidth"`   // px (original image width)
	NaturalHeight  int     `json:"naturalHeight"`  // px (original image height)
	RenderedWidth  int     `json:"renderedWidth"`  // px (DOM display width via getBoundingClientRect / clientWidth)
	RenderedHeight int     `json:"renderedHeight"` // px (DOM display height via getBoundingClientRect / clientHeight)
	FormattedSize  string  `json:"formattedSize"`  // e.g. "450 KB"
	Format         string  `json:"format"`         // e.g. "png", "jpg", "webp", "svg", "avif"
	IsLazy         bool    `json:"isLazy"`         // whether loading="lazy" is set
	Alt            string  `json:"alt"`            // image alt text if available
	IsLCP          bool    `json:"isLCP"`          // whether this image is the LCP element
}

type FontDetail struct {
	Family       string  `json:"family"`       // e.g. "Inter", "Roboto"
	URL          string  `json:"url"`          // font asset URL if available
	Type         string  `json:"type"`         // woff2, woff, ttf, google-font, etc.
	TransferSize int64   `json:"transferSize"` // bytes
	Duration     float64 `json:"duration"`     // ms
}

// IframeDetail is a DOM <iframe> seen during page load collection.
// LoadedDuringScan=false means iframe was in DOM but had no network timing
// in the observation window (lazy / below-fold / deferred) — i.e. "missed".
type IframeDetail struct {
	Src              string  `json:"src"`
	Title            string  `json:"title"`
	Width            int     `json:"width"`
	Height           int     `json:"height"`
	IsLazy           bool    `json:"isLazy"`
	LoadedDuringScan bool    `json:"loadedDuringScan"`
	InViewport       bool    `json:"inViewport"`
	Sandbox          bool    `json:"sandbox"`
	Duration         float64 `json:"duration"`     // ms
	TransferSize     int64   `json:"transferSize"` // bytes
	FormattedSize    string  `json:"formattedSize"`
}

type CategoryDiagnostic struct {
	Category string   `json:"category"` // "ttfb", "fcp", "lcp", "cls", "tbt"
	Title    string   `json:"title"`
	Status   string   `json:"status"`  // "good", "needs-improvement", "poor"
	Summary  string   `json:"summary"` // What caused the score
	Details  []string `json:"details"` // Specific technical breakdown items
	Fixes    []string `json:"fixes"`   // Actionable step-by-step solutions
}

type PageDiagnostics struct {
	DNSTime             float64                        `json:"dnsTime"`
	TCPTime             float64                        `json:"tcpTime"`
	TLSTime             float64                        `json:"tlsTime"`
	ServerProcessing    float64                        `json:"serverProcessing"`
	RenderBlockingCount int                            `json:"renderBlockingCount"`
	RenderBlockingFiles []string                       `json:"renderBlockingFiles"`
	LCPElement          string                         `json:"lcpElement"`
	LCPUrl              string                         `json:"lcpUrl"`
	ShiftCount          int                            `json:"shiftCount"`
	ShiftCauses         []string                       `json:"shiftCauses"`
	LongTasksCount      int                            `json:"longTasksCount"`
	MaxLongTaskMs       float64                        `json:"maxLongTaskMs"`
	SlowestResources    []ResourceTiming               `json:"slowestResources"`
	LargestImages       []ImageDetail                  `json:"largestImages"`
	Fonts               []FontDetail                   `json:"fonts"`
	Iframes             []IframeDetail                 `json:"iframes"`
	Forms               []FormDetail                   `json:"forms"`
	Categories          map[string]CategoryDiagnostic `json:"categories"`
	W3C                 interface{}                    `json:"w3c,omitempty"`
}

// FormFieldDetail describes a single input/field inside a form
type FormFieldDetail struct {
	Name              string `json:"name"`
	Label             string `json:"label"`
	Type              string `json:"type"` // text, email, tel, select, checkbox, radio, file, textarea, hidden, submit
	IsRequired        bool   `json:"isRequired"`
	ValidationPattern string `json:"validationPattern,omitempty"`
	Accept            string `json:"accept,omitempty"`
	Value             string `json:"value,omitempty"`
}

// CaptchaDetail captures anti-spam and captcha integration state
type CaptchaDetail struct {
	Type     string `json:"type"` // "recaptcha-v3", "recaptcha-v2", "turnstile", "hcaptcha", "honeypot", "none"
	SiteKey  string `json:"siteKey,omitempty"`
	Action   string `json:"action,omitempty"`
	IsActive bool   `json:"isActive"`
}

// FormDetail captures complete DOM and metadata for a detected web form
type FormDetail struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Engine           string            `json:"engine"` // "contact-form-7", "pardot", "greenhouse", "hubspot", "native-html", "custom"
	Method           string            `json:"method"` // "POST", "GET"
	Action           string            `json:"action"`
	Fields           []FormFieldDetail `json:"fields"`
	FieldCount       int               `json:"fieldCount"`
	HasFileUpload    bool              `json:"hasFileUpload"`
	AllowedFileTypes string            `json:"allowedFileTypes,omitempty"`
	Captcha          CaptchaDetail     `json:"captcha"`
	HiddenTokens     map[string]string `json:"hiddenTokens,omitempty"`
	InViewport       bool              `json:"inViewport"`
}

type PageResult struct {
	ID                    int             `json:"id"`
	URL                   string          `json:"url"`
	StatusCode            int             `json:"statusCode"`
	PluginCacheStatus     string          `json:"pluginCacheStatus"`     // e.g. "HIT", "MISS", "BYPASS", "NONE"
	CloudflareCacheStatus string          `json:"cloudflareCacheStatus"` // e.g. "HIT", "MISS", "DYNAMIC", "REVALIDATED", "BYPASS", "NONE"
	CloudflarePop         string          `json:"cloudflarePop,omitempty"`         // e.g. "KBP", "WAW", "FRA", "LHR"
	CloudflareRay         string          `json:"cloudflareRay,omitempty"`         // e.g. "92837264871638-WAW"
	Metrics               WebVitals       `json:"metrics"`
	Grades                DetailedGrades  `json:"grades"`
	Diagnostics           PageDiagnostics `json:"diagnostics"`
	OverallStatus         string          `json:"overallStatus"` // "good", "needs-improvement", "poor", "error"
	Error                 string          `json:"error"`
	Recommendations       []string        `json:"recommendations"`
	DurationMs            int64           `json:"durationMs"`
}

type ScanProgress struct {
	TotalProcessed int         `json:"totalProcessed"`
	TotalURLs      int         `json:"totalUrls"`
	CurrentURL     string      `json:"currentUrl"`
	LatestResult   *PageResult `json:"latestResult,omitempty"`
	IsFinished     bool        `json:"isFinished"`
}

func CalculateGrade(metricName string, val float64) MetricGrade {
	status := "good"
	formatted := ""

	switch metricName {
	case "TTFB":
		formatted = formatMs(val)
		if val > 1800 {
			status = "poor"
		} else if val > 800 {
			status = "needs-improvement"
		}
	case "FCP":
		formatted = formatMs(val)
		if val > 3000 {
			status = "poor"
		} else if val > 1800 {
			status = "needs-improvement"
		}
	case "LCP":
		formatted = formatMs(val)
		if val > 4000 {
			status = "poor"
		} else if val > 2500 {
			status = "needs-improvement"
		}
	case "CLS":
		formatted = formatDecimal(val, 3)
		if val > 0.25 {
			status = "poor"
		} else if val > 0.1 {
			status = "needs-improvement"
		}
	case "TBT":
		formatted = formatMs(val)
		if val > 600 {
			status = "poor"
		} else if val > 200 {
			status = "needs-improvement"
		}
	}

	return MetricGrade{
		Value:     val,
		Formatted: formatted,
		Status:    status,
	}
}

func CalculateOverallStatus(g DetailedGrades) string {
	statuses := []string{g.TTFB.Status, g.FCP.Status, g.LCP.Status, g.CLS.Status, g.TBT.Status}
	hasPoor := false
	hasNeedsImprovement := false

	for _, s := range statuses {
		if s == "poor" {
			hasPoor = true
		} else if s == "needs-improvement" {
			hasNeedsImprovement = true
		}
	}

	if hasPoor {
		return "poor"
	}
	if hasNeedsImprovement {
		return "needs-improvement"
	}
	return "good"
}

func formatMs(val float64) string {
	if val >= 1000 {
		return formatDecimal(val/1000.0, 2) + "s"
	}
	return formatDecimal(val, 0) + "ms"
}

func formatDecimal(val float64, decimals int) string {
	if decimals == 0 {
		return fmt.Sprintf("%.0f", val)
	} else if decimals == 2 {
		return fmt.Sprintf("%.2f", val)
	}
	return fmt.Sprintf("%.3f", val)
}
