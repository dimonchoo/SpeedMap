package config

type CustomHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ScanConfig struct {
	SitemapURL  string         `json:"sitemapUrl"`
	Concurrency int            `json:"concurrency"` // 1 - 10
	AuthUser    string         `json:"authUser"`
	AuthPass    string         `json:"authPass"`
	Headers     []CustomHeader `json:"headers"`
	IsMobile    bool           `json:"isMobile"`
	TimeoutSec  int            `json:"timeoutSec"`
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

func (c *ScanConfig) NormalizedTimeout() int {
	if c.TimeoutSec <= 0 {
		return 30
	}
	return c.TimeoutSec
}
