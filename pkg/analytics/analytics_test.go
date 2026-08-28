package analytics

import (
	"testing"

	"SpeedMap/pkg/scanner"
)

func TestComputeSiteAnalytics(t *testing.T) {
	pages := []scanner.PageResult{
		{
			ID:            1,
			URL:           "https://example.com/page1",
			StatusCode:    200,
			OverallStatus: "good",
			Metrics: scanner.WebVitals{
				TTFB: 300,
				FCP:  800,
				LCP:  1200,
				CLS:  0.01,
				TBT:  50,
			},
			Diagnostics: scanner.PageDiagnostics{
				SlowestResources: []scanner.ResourceTiming{
					{Name: "https://example.com/js/app.js", Type: "script", Duration: 350},
				},
				LargestImages: []scanner.ImageDetail{
					{URL: "https://example.com/img/hero.jpg", TransferSize: 524288, Duration: 250, Width: 1200, Height: 800, FormattedSize: "512 KB"},
				},
				Fonts: []scanner.FontDetail{
					{Family: "Inter", URL: "https://example.com/fonts/inter.woff2", Type: "woff2", TransferSize: 25000, Duration: 60},
				},
				Iframes: []scanner.IframeDetail{
					{Src: "https://www.youtube.com/embed/abc", Title: "Video", Width: 560, Height: 315, LoadedDuringScan: true, Duration: 120},
					{Src: "https://maps.example.com/embed", Title: "Map", Width: 600, Height: 400, IsLazy: true, LoadedDuringScan: false},
				},
			},
		},
		{
			ID:            2,
			URL:           "https://example.com/page2",
			StatusCode:    200,
			OverallStatus: "needs-improvement",
			Metrics: scanner.WebVitals{
				TTFB: 900,
				FCP:  1800,
				LCP:  3200,
				CLS:  0.15,
				TBT:  250,
			},
			Diagnostics: scanner.PageDiagnostics{
				SlowestResources: []scanner.ResourceTiming{
					{Name: "https://example.com/js/app.js", Type: "script", Duration: 400},
					{Name: "https://example.com/fonts/inter.woff2", Type: "font", Duration: 200},
				},
				LargestImages: []scanner.ImageDetail{
					{URL: "https://example.com/img/hero.jpg", TransferSize: 524288, Duration: 280, Width: 1200, Height: 800, FormattedSize: "512 KB"},
					{URL: "https://example.com/img/banner.png", TransferSize: 1048576, Duration: 450, Width: 1920, Height: 1080, FormattedSize: "1.0 MB"},
				},
				Fonts: []scanner.FontDetail{
					{Family: "Inter", URL: "https://example.com/fonts/inter.woff2", Type: "woff2", TransferSize: 25000, Duration: 80},
					{Family: "Roboto", URL: "https://example.com/fonts/roboto.woff2", Type: "woff2", TransferSize: 30000, Duration: 90},
				},
				Iframes: []scanner.IframeDetail{
					{Src: "https://www.youtube.com/embed/abc", Title: "Video", Width: 560, Height: 315, LoadedDuringScan: false, IsLazy: true},
				},
				Forms: []scanner.FormDetail{
					{
						ID:     "wpcf7-f34972-o1",
						Title:  "Footer Form",
						Engine: "contact-form-7",
						Method: "POST",
						Action: "/#wpcf7-f34972-o1",
						Fields: []scanner.FormFieldDetail{
							{Name: "firstname", Label: "First Name", Type: "text", IsRequired: true},
							{Name: "email", Label: "Company Email", Type: "email", IsRequired: true},
						},
						FieldCount: 2,
						Captcha: scanner.CaptchaDetail{
							Type:     "recaptcha-v3",
							SiteKey:  "6Le-wvkSAAAAAPBMRT0X3nBDyd2h4BPn64qStZZL",
							IsActive: true,
						},
					},
					{
						ID:               "application_form",
						Title:            "Job Application",
						Engine:           "greenhouse",
						Method:           "POST",
						HasFileUpload:    true,
						AllowedFileTypes: ".pdf, .doc, .docx",
						Fields: []scanner.FormFieldDetail{
							{Name: "resume", Label: "Resume/CV", Type: "file", IsRequired: true, Accept: ".pdf,.doc,.docx"},
						},
						FieldCount: 1,
						Captcha:    scanner.CaptchaDetail{Type: "none", IsActive: false},
					},
				},
			},
		},
	}

	analytics := ComputeSiteAnalytics(pages)

	if analytics.TotalPages != 2 {
		t.Errorf("Expected TotalPages = 2, got %d", analytics.TotalPages)
	}

	if analytics.HealthScore != 75 { // (1 * 1.0 + 1 * 0.5) / 2 * 100 = 75
		t.Errorf("Expected HealthScore = 75, got %d", analytics.HealthScore)
	}

	if len(analytics.TopResourceBottlenecks) != 2 {
		t.Fatalf("Expected 2 bottleneck resources, got %d", len(analytics.TopResourceBottlenecks))
	}

	// app.js should be first with occurrences = 2
	if analytics.TopResourceBottlenecks[0].Name != "https://example.com/js/app.js" {
		t.Errorf("Expected top resource to be app.js, got %s", analytics.TopResourceBottlenecks[0].Name)
	}
	if analytics.TopResourceBottlenecks[0].Occurrences != 2 {
		t.Errorf("Expected app.js occurrences = 2, got %d", analytics.TopResourceBottlenecks[0].Occurrences)
	}

	// Check Largest Images
	if len(analytics.LargestImages) != 2 {
		t.Fatalf("Expected 2 aggregated largest images, got %d", len(analytics.LargestImages))
	}
	if analytics.LargestImages[0].URL != "https://example.com/img/banner.png" {
		t.Errorf("Expected top image to be banner.png, got %s", analytics.LargestImages[0].URL)
	}
	if analytics.LargestImages[1].PageCount != 2 {
		t.Errorf("Expected hero.jpg page count = 2, got %d", analytics.LargestImages[1].PageCount)
	}
	if len(analytics.LargestImages[1].Pages) != 2 {
		t.Fatalf("Expected hero.jpg on 2 pages, got %v", analytics.LargestImages[1].Pages)
	}
	wantPages := map[string]bool{"https://example.com/page1": true, "https://example.com/page2": true}
	for _, pg := range analytics.LargestImages[1].Pages {
		if !wantPages[pg] {
			t.Errorf("unexpected page %s", pg)
		}
	}

	// Check Font Usage
	if len(analytics.FontUsage) != 2 {
		t.Fatalf("Expected 2 fonts aggregated, got %d", len(analytics.FontUsage))
	}
	if analytics.FontUsage[0].Family != "Inter" {
		t.Errorf("Expected top font to be Inter, got %s", analytics.FontUsage[0].Family)
	}
	if analytics.FontUsage[0].Occurrences != 2 || analytics.FontUsage[0].Percentage != 100 {
		t.Errorf("Expected Inter font occurrences=2, percentage=100%%, got %d (%.1f%%)", analytics.FontUsage[0].Occurrences, analytics.FontUsage[0].Percentage)
	}

	if analytics.TotalIframeCount != 2 {
		t.Errorf("Expected TotalIframeCount = 2, got %d", analytics.TotalIframeCount)
	}
	if analytics.MissedIframeCount != 2 {
		t.Errorf("Expected MissedIframeCount = 2, got %d", analytics.MissedIframeCount)
	}
	if analytics.LoadedIframeCount != 1 {
		t.Errorf("Expected LoadedIframeCount = 1, got %d", analytics.LoadedIframeCount)
	}
	if len(analytics.Iframes) != 2 {
		t.Fatalf("Expected 2 aggregated iframes, got %d", len(analytics.Iframes))
	}

	if len(analytics.GlobalFixes) == 0 {
		t.Errorf("Expected global fixes to be generated")
	}

	// Verify SEOAEO-235 Image Optimization Analytics
	if analytics.TotalImageCount != 2 {
		t.Errorf("Expected TotalImageCount = 2, got %d", analytics.TotalImageCount)
	}
	if analytics.HeavyImagesCount != 2 {
		t.Errorf("Expected HeavyImagesCount = 2 (>100KB), got %d", analytics.HeavyImagesCount)
	}
	if analytics.TotalWebPSavingsBytes <= 0 {
		t.Errorf("Expected positive WebP savings, got %d", analytics.TotalWebPSavingsBytes)
	}

	// Verify Form Analytics
	if analytics.TotalFormsCount != 2 {
		t.Errorf("Expected TotalFormsCount = 2, got %d", analytics.TotalFormsCount)
	}
	if analytics.PagesWithFormsCount != 1 {
		t.Errorf("Expected PagesWithFormsCount = 1, got %d", analytics.PagesWithFormsCount)
	}
	if analytics.CaptchaProtectedCount != 1 {
		t.Errorf("Expected CaptchaProtectedCount = 1, got %d", analytics.CaptchaProtectedCount)
	}
	if analytics.UnprotectedFormsCount != 1 {
		t.Errorf("Expected UnprotectedFormsCount = 1, got %d", analytics.UnprotectedFormsCount)
	}
	if analytics.FileUploadFormsCount != 1 {
		t.Errorf("Expected FileUploadFormsCount = 1, got %d", analytics.FileUploadFormsCount)
	}
	if len(analytics.Forms) != 2 {
		t.Fatalf("Expected 2 aggregated forms, got %d", len(analytics.Forms))
	}

	// Test GenerateImageComparisonHTML
	htmlReport := GenerateImageComparisonHTML(analytics, "example.com")
	if len(htmlReport) == 0 {
		t.Fatalf("Expected non-empty HTML report string")
	}
	if !testing.Short() && len(htmlReport) < 100 {
		t.Errorf("Report HTML seems unexpectedly small: %d chars", len(htmlReport))
	}
}

func TestImageMultiPageRenderAggregationAndRetina(t *testing.T) {
	// An image used across 3 different pages with different CSS render sizes:
	// Page 1: Avatar render (83x83)
	// Page 2: Bio card render (350x350)
	// Page 3: Team grid render (200x200)
	// Original file is 1200x1200 px
	pages := []scanner.PageResult{
		{
			ID:  1,
			URL: "https://example.com/page-avatar",
			Diagnostics: scanner.PageDiagnostics{
				LargestImages: []scanner.ImageDetail{
					{
						URL:            "https://example.com/img/author.png",
						TransferSize:   800 * 1024,
						Width:          1200,
						Height:         1200,
						NaturalWidth:   1200,
						NaturalHeight:  1200,
						RenderedWidth:  83,
						RenderedHeight: 83,
						Format:         "png",
					},
				},
			},
		},
		{
			ID:  2,
			URL: "https://example.com/page-bio",
			Diagnostics: scanner.PageDiagnostics{
				LargestImages: []scanner.ImageDetail{
					{
						URL:            "https://example.com/img/author.png",
						TransferSize:   800 * 1024,
						Width:          1200,
						Height:         1200,
						NaturalWidth:   1200,
						NaturalHeight:  1200,
						RenderedWidth:  350,
						RenderedHeight: 350,
						Format:         "png",
					},
				},
			},
		},
		{
			ID:  3,
			URL: "https://example.com/page-team",
			Diagnostics: scanner.PageDiagnostics{
				LargestImages: []scanner.ImageDetail{
					{
						URL:            "https://example.com/img/author.png",
						TransferSize:   800 * 1024,
						Width:          1200,
						Height:         1200,
						NaturalWidth:   1200,
						NaturalHeight:  1200,
						RenderedWidth:  200,
						RenderedHeight: 200,
						Format:         "png",
					},
				},
			},
		},
	}

	siteAnalytics := ComputeSiteAnalytics(pages)

	if len(siteAnalytics.AllImages) != 1 {
		t.Fatalf("Expected 1 aggregated image, got %d", len(siteAnalytics.AllImages))
	}

	img := siteAnalytics.AllImages[0]
	if img.PageCount != 3 {
		t.Errorf("Expected PageCount = 3, got %d", img.PageCount)
	}
	if img.MaxRenderedWidth != 350 {
		t.Errorf("Expected MaxRenderedWidth = 350 (max across 83, 350, 200), got %d", img.MaxRenderedWidth)
	}
	if img.MaxRenderedHeight != 350 {
		t.Errorf("Expected MaxRenderedHeight = 350, got %d", img.MaxRenderedHeight)
	}
	if img.RecommendedRetinaWidth != 700 { // 350 * 2
		t.Errorf("Expected RecommendedRetinaWidth = 700 (2x Retina), got %d", img.RecommendedRetinaWidth)
	}
	if img.RecommendedRetinaHeight != 700 {
		t.Errorf("Expected RecommendedRetinaHeight = 700, got %d", img.RecommendedRetinaHeight)
	}
	if !img.IsOversized { // 1200 > 700
		t.Errorf("Expected IsOversized = true (1200 > 700)")
	}
	if siteAnalytics.OversizedImagesCount != 1 {
		t.Errorf("Expected OversizedImagesCount = 1, got %d", siteAnalytics.OversizedImagesCount)
	}
}
