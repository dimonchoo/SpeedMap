package optimizer

import (
	"strings"
	"testing"
)

func TestOptimizeSVG_SpriteSafe(t *testing.T) {
	rawSVG := `<?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 25.0.0, SVG Export Plug-In -->
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100">
  <metadata>
    <custom:meta>Unused metadata</custom:meta>
  </metadata>
  <g id="useless-group">
    <symbol id="icon-search" viewBox="0 0 24 24">
      <path d="M10.123456 10.789012 L20.555555 20.666666" />
    </symbol>
  </g>
</svg>`

	opt, err := OptimizeSVG([]byte(rawSVG))
	if err != nil {
		t.Fatalf("OptimizeSVG failed: %v", err)
	}

	optStr := string(opt)
	if len(opt) >= len(rawSVG) {
		t.Fatalf("expected optimized size (%d) < original size (%d)", len(opt), len(rawSVG))
	}

	// Verify symbol and symbol id are preserved!
	if !strings.Contains(optStr, "icon-search") {
		t.Fatalf("critical error: symbol id 'icon-search' was stripped: %s", optStr)
	}
	if !strings.Contains(optStr, "symbol") {
		t.Fatalf("critical error: <symbol> tag was removed: %s", optStr)
	}
	// Verify XML comments & metadata are stripped
	if strings.Contains(optStr, "Adobe Illustrator") {
		t.Fatalf("XML comments should be stripped: %s", optStr)
	}
	if strings.Contains(optStr, "custom:meta") {
		t.Fatalf("metadata should be stripped: %s", optStr)
	}
}

func TestMinifySVGNative(t *testing.T) {
	rawSVG := `<!-- Comment -->
<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="http://www.inkscape.org">
  <symbol id="icon-test" viewBox="0 0 20 20">
    <path d="M12.34567 15.6789" />
  </symbol>
</svg>`

	min := MinifySVGNative([]byte(rawSVG))
	minStr := string(min)

	if strings.Contains(minStr, "Comment") {
		t.Fatalf("native minifier did not strip comment")
	}
	if strings.Contains(minStr, "xmlns:inkscape") {
		t.Fatalf("native minifier did not strip editor ns")
	}
	if !strings.Contains(minStr, "icon-test") {
		t.Fatalf("native minifier lost symbol id")
	}
	if len(min) >= len(rawSVG) {
		t.Fatalf("expected minified len < raw len")
	}
}
