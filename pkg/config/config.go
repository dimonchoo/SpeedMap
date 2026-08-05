package config

type CustomHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ScanConfig struct {
	SitemapURL            string         `json:"sitemapUrl"`
	Concurrency           int            `json:"concurrency"`           // 1 - 10, default 1
	HeavyImageThresholdKB int            `json:"heavyImageThresholdKB"` // KB, default 100
	AuthUser              string         `json:"authUser"`
	AuthPass              string         `json:"authPass"`
	Headers               []CustomHeader `json:"headers"`
	IsMobile              bool           `json:"isMobile"`
	AutoScroll            bool           `json:"autoScroll"` // Disabled by default
	TimeoutSec            int            `json:"timeoutSec"`
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

func (c *ScanConfig) NormalizedTimeout() int {
	if c.TimeoutSec <= 0 {
		return 30
	}
	return c.TimeoutSec
}
