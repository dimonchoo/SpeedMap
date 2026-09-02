package config

type CustomHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ScanConfig struct {
	SitemapURL            string         `json:"sitemapUrl"`
	Concurrency           int            `json:"concurrency"`           // 1 - 10, default 1
	HeavyImageThresholdKB int            `json:"heavyImageThresholdKB"` // KB, default 100
	WebPQuality           float32        `json:"webpQuality"`           // 1 - 100, default 80
	PngWebPRatio          int            `json:"pngWebPRatio"`          // % of original, default 30 (70% savings)
	JpgWebPRatio          int            `json:"jpgWebPRatio"`          // % of original, default 60 (40% savings)
	GifWebPRatio          int            `json:"gifWebPRatio"`          // % of original, default 50 (50% savings)
	AuthUser              string         `json:"authUser"`
	AuthPass              string         `json:"authPass"`
	Headers               []CustomHeader `json:"headers"`
	IsMobile              bool           `json:"isMobile"`
	AutoScroll            bool           `json:"autoScroll"` // Disabled by default
	TimeoutSec            int            `json:"timeoutSec"`
	GDriveClientID        string         `json:"gdriveClientID"`
	GDriveClientSecret    string         `json:"gdriveClientSecret"`
	MinWebPQuality        float32        `json:"minWebPQuality"`        // Minimum quality floor (default 80)
	SkipIfNoWebPSavings   *bool          `json:"skipIfNoWebPSavings"`   // true by default: skip images if WebP size >= original size
	AdaptiveQuality       *bool          `json:"adaptiveQuality"`       // true by default: automatically adjusts quality for gradients/transparency
	ResizeToRetina        *bool          `json:"resizeToRetina"`        // true by default: resize oversized images to max rendered Retina 2x bounds
	AutoPruneHistory      *bool          `json:"autoPruneHistory"`      // true by default: auto-prunes old history runs
	HistoryRetentionRuns  int            `json:"historyRetentionRuns"`  // default 20 (max runs per domain, 0 = unlimited)
	HistoryRetentionDays  int            `json:"historyRetentionDays"`  // default 30 (max age in days, 0 = unlimited)
	FilterTrackingBeacons *bool          `json:"filterTrackingBeacons"` // true by default: filter 1x1 ad & tracking beacons
	ExcludedImagePatterns []string       `json:"excludedImagePatterns"` // custom substrings/patterns to exclude from image discovery
}

func (c *ScanConfig) IsAdaptiveQualityEnabled() bool {
	if c.AdaptiveQuality == nil {
		return true // Default true
	}
	return *c.AdaptiveQuality
}

func (c *ScanConfig) NormalizedMinWebPQuality() float32 {
	if c.MinWebPQuality <= 0 || c.MinWebPQuality > 100 {
		return 75.0 // 75% default floor (allows step-down fallback from base 80%)
	}
	return c.MinWebPQuality
}

func (c *ScanConfig) IsSkipIfNoWebPSavingsEnabled() bool {
	if c.SkipIfNoWebPSavings == nil {
		return true // Default true
	}
	return *c.SkipIfNoWebPSavings
}

func (c *ScanConfig) IsResizeToRetinaEnabled() bool {
	if c.ResizeToRetina == nil {
		return true // Default true
	}
	return *c.ResizeToRetina
}

func (c *ScanConfig) IsAutoPruneEnabled() bool {
	if c.AutoPruneHistory == nil {
		return true // Default true
	}
	return *c.AutoPruneHistory
}

func (c *ScanConfig) NormalizedRetentionRuns() int {
	if c.HistoryRetentionRuns < 0 {
		return 0
	}
	if c.HistoryRetentionRuns == 0 {
		return 20
	}
	return c.HistoryRetentionRuns
}

func (c *ScanConfig) NormalizedRetentionDays() int {
	if c.HistoryRetentionDays < 0 {
		return 0
	}
	if c.HistoryRetentionDays == 0 {
		return 30
	}
	return c.HistoryRetentionDays
}


func (c *ScanConfig) NormalizedConcurrency() int {
	if c.Concurrency < 1 {
		return 1
	}
	if c.Concurrency > 10 {
		return 10
	}
	return c.Concurrency
}

func (c *ScanConfig) NormalizedHeavyThresholdBytes() int64 {
	if c.HeavyImageThresholdKB <= 0 {
		return 100 * 1024 // 100 KB default
	}
	return int64(c.HeavyImageThresholdKB) * 1024
}

func (c *ScanConfig) NormalizedWebPQuality() float32 {
	if c.WebPQuality <= 0 || c.WebPQuality > 100 {
		return 80.0 // 80% quality default
	}
	return c.WebPQuality
}

func (c *ScanConfig) NormalizedPngRatio() float64 {
	if c.PngWebPRatio <= 0 || c.PngWebPRatio > 100 {
		return 0.30 // 30% ratio default
	}
	return float64(c.PngWebPRatio) / 100.0
}

func (c *ScanConfig) NormalizedJpgRatio() float64 {
	if c.JpgWebPRatio <= 0 || c.JpgWebPRatio > 100 {
		return 0.60 // 60% ratio default
	}
	return float64(c.JpgWebPRatio) / 100.0
}

func (c *ScanConfig) NormalizedGifRatio() float64 {
	if c.GifWebPRatio <= 0 || c.GifWebPRatio > 100 {
		return 0.50 // 50% ratio default
	}
	return float64(c.GifWebPRatio) / 100.0
}

func (c *ScanConfig) NormalizedTimeout() int {
	if c.TimeoutSec <= 0 {
		return 30
	}
	return c.TimeoutSec
}

func (c *ScanConfig) IsFilterTrackingBeaconsEnabled() bool {
	if c.FilterTrackingBeacons == nil {
		return true // Default true
	}
	return *c.FilterTrackingBeacons
}

func (c *ScanConfig) GetExcludedImagePatterns() []string {
	if c.FilterTrackingBeacons != nil && !*c.FilterTrackingBeacons {
		if c.ExcludedImagePatterns == nil {
			return []string{}
		}
		return c.ExcludedImagePatterns
	}
	if len(c.ExcludedImagePatterns) > 0 {
		return c.ExcludedImagePatterns
	}
	return []string{
		"googleadservices.com",
		"doubleclick.net",
		"facebook.com/tr",
		"bat.bing.com",
		"clarity.ms",
		"px.ads.linkedin.com",
		"analytics.google.com",
		"google-analytics.com",
		"t.co/1/i/adsct",
		"stats.wp.com",
		"/pagead/",
	}
}
