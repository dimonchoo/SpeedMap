package scanner

import (
	"fmt"
	"testing"

	"SpeedMap/pkg/config"
)

func TestSelectivePagesQA(t *testing.T) {
	qaURLs := []string{
		"https://uat.infuse.com/academy/",
		"https://uat.infuse.com/insight/demand-generation-anti-trends/",
		"https://uat.infuse.com/insight/why-your-buyers-have-ghosted-you/",
		"https://uat.infuse.com/insight/how-to-develop-a-compelling-uvp-for-competitive-displacement-strategies/",
		"https://uat.infuse.com/insight/hubspot-infuse-pr-partnership-announcement/",
	}

	cfg := config.ScanConfig{
		Concurrency: 2,
		TimeoutSec:  30,
		AutoScroll:  true,
	}

	sc := NewScanner(cfg)
	results, err := sc.ScanURLs(qaURLs, nil)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}

	fmt.Printf("\n=== QA SELECTIVE SCAN COMPLETED (%d pages) ===\n", len(results))
	for _, res := range results {
		fmt.Printf("Page: %s (Status %d, LCP: %.1fms)\n", res.URL, res.StatusCode, res.Metrics.LCP)
		for _, img := range res.Diagnostics.LargestImages {
			if img.RenderedWidth > 0 || img.Width > 500 {
				fmt.Printf("  Image: %s | Natural: %dx%d | Rendered: %dx%d | Size: %s\n",
					img.URL, img.NaturalWidth, img.NaturalHeight, img.RenderedWidth, img.RenderedHeight, img.FormattedSize)
			}
		}
	}
}
