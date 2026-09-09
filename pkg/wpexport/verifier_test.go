package wpexport

import (
	"testing"
)

func TestMapURLToTargetDomain(t *testing.T) {
	origURL := "https://infuse.com/wp-content/uploads/2021/11/head-1-1.jpg"
	targetScheme := "https"
	targetDomain := "uat.infuse.com"

	webpURL := mapURLToTargetDomain(origURL, targetScheme, targetDomain, "jpg", true)
	expectedWebp := "https://uat.infuse.com/wp-content/uploads/2021/11/head-1-1.webp"
	if webpURL != expectedWebp {
		t.Fatalf("expected %s, got %s", expectedWebp, webpURL)
	}

	oldTargetURL := mapURLToTargetDomain(origURL, targetScheme, targetDomain, "jpg", false)
	expectedOld := "https://uat.infuse.com/wp-content/uploads/2021/11/head-1-1.jpg"
	if oldTargetURL != expectedOld {
		t.Fatalf("expected %s, got %s", expectedOld, oldTargetURL)
	}

	// SVG test
	origSVG := "https://infuse.com/wp-content/themes/theme/assets/logo.svg"
	svgURL := mapURLToTargetDomain(origSVG, targetScheme, targetDomain, "svg", true)
	expectedSVG := "https://uat.infuse.com/wp-content/themes/theme/assets/logo.svg"
	if svgURL != expectedSVG {
		t.Fatalf("expected %s, got %s", expectedSVG, svgURL)
	}
}

func TestMapPageURLToTargetDomain(t *testing.T) {
	origPage := "https://infuse.com/insight/definitive-guides/"
	targetPage := mapPageURLToTargetDomain(origPage, "https", "uat.infuse.com")
	expected := "https://uat.infuse.com/insight/definitive-guides/"
	if targetPage != expected {
		t.Fatalf("expected %s, got %s", expected, targetPage)
	}
}
