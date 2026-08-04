package scanner

import (
	"fmt"
	"testing"

	"SpeedMap/pkg/config"
)

func TestScannerInfuse(t *testing.T) {
	cfg := config.ScanConfig{
		Concurrency: 1,
		TimeoutSec:  30,
	}

	sc := NewScanner(cfg)
	results, err := sc.ScanURLs([]string{"https://infuse.com/"}, nil)
	if err != nil {
		t.Fatalf("scan error: %v", err)
	}

	for _, res := range results {
		fmt.Printf("TEST RESULT INFUSE [%s]: Status=%d, TTFB=%.2fms, FCP=%.2fms, LCP=%.2fms, CLS=%.3f, TBT=%.2fms, OverallStatus=%s, Error='%s'\n",
			res.URL, res.StatusCode, res.Metrics.TTFB, res.Metrics.FCP, res.Metrics.LCP, res.Metrics.CLS, res.Metrics.TBT, res.OverallStatus, res.Error)
	}
}
