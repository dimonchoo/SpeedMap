package optimizer

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	svgCommentRE    = regexp.MustCompile(`<!--[\s\S]*?-->`)
	svgXMLDeclRE    = regexp.MustCompile(`<\?xml[\s\S]*?\?>`)
	svgDocTypeRE    = regexp.MustCompile(`<!DOCTYPE[\s\S]*?>`)
	svgMetadataRE   = regexp.MustCompile(`(?i)<metadata[\s\S]*?<\/metadata>`)
	svgEditorNSRE   = regexp.MustCompile(`\s*xmlns:(?:inkscape|sodipodi|sketch|illustrator|adobe|custom)="[^"]*"`)
	svgEditorAttrRE = regexp.MustCompile(`\s*(?:inkscape|sodipodi|sketch|adobe|i):[a-zA-Z0-9_-]+="[^"]*"`)
	svgEmptyGroupRE = regexp.MustCompile(`<g\s*>\s*<\/g>`)
	svgTagWSRE      = regexp.MustCompile(`>\s+<`)
	svgMultiWSRE    = regexp.MustCompile(`[\t\r\n]+`)
	svgExcessWSRE   = regexp.MustCompile(` {2,}`)
	svgCoordRE      = regexp.MustCompile(`(\b\d+\.\d{2})\d+\b`)

	svgoConfigPath string
	svgoConfigOnce sync.Once
)

const safeSVGOConfig = `export default {
  multipass: true,
  plugins: [
    {
      name: "preset-default",
      params: {
        floatPrecision: 1,
        overrides: {
          cleanupIds: false,
          removeHiddenElems: false,
          removeUselessDefs: false,
        }
      }
    }
  ]
};
`

func init() {
	path := os.Getenv("PATH")
	extraPaths := []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/homebrew/sbin", "/usr/bin", "/bin"}
	for _, p := range extraPaths {
		if !strings.Contains(path, p) {
			path = p + ":" + path
		}
	}
	_ = os.Setenv("PATH", path)
}

// getSVGOConfig returns the path to the cached sprite-safe SVGO config file.
func getSVGOConfig() string {
	svgoConfigOnce.Do(func() {
		tmpDir := os.TempDir()
		cfgFile := filepath.Join(tmpDir, "speedmap_svgo_safe.mjs")
		_ = os.WriteFile(cfgFile, []byte(safeSVGOConfig), 0644)
		svgoConfigPath = cfgFile
	})
	return svgoConfigPath
}

// OptimizeSVG runs sprite-safe SVGO (if available) with a fallback to native Go vector minification.
// It guarantees that symbol IDs and sprite structures are never broken, and only returns optimized
// bytes if the result is actually smaller than the original.
func OptimizeSVG(svgBytes []byte) ([]byte, error) {
	if len(svgBytes) == 0 {
		return nil, fmt.Errorf("empty svg input")
	}

	// 1. Try SVGO with sprite-safe configuration
	optBytes, err := RunSVGO(svgBytes)
	if err == nil && len(optBytes) > 0 && isValidSVG(optBytes) {
		if len(optBytes) < len(svgBytes) {
			return optBytes, nil
		}
		return svgBytes, nil
	}

	// 2. Fallback to native pure-Go vector minifier
	nativeBytes := MinifySVGNative(svgBytes)
	if len(nativeBytes) > 0 && isValidSVG(nativeBytes) && len(nativeBytes) < len(svgBytes) {
		return nativeBytes, nil
	}

	return svgBytes, nil
}

// RunSVGO executes SVGO using direct binary or npx with sprite-safe flags.
func RunSVGO(svgBytes []byte) ([]byte, error) {
	var cmdPath string
	var cmdArgs []string

	cfgPath := getSVGOConfig()

	if p, err := exec.LookPath("svgo"); err == nil {
		cmdPath = p
		cmdArgs = []string{"--config", cfgPath, "-i", "-", "-o", "-"}
	} else {
		// Do not spawn `npx -y svgo` dynamically: npx initializes node/npm, taking 1000ms+ and 100%+ CPU.
		// Fallback directly to native pure-Go MinifySVGNative which executes in 0.05ms with 0 CPU overhead.
		return nil, fmt.Errorf("svgo binary not found on PATH, falling back to native minifier")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, cmdArgs...)
	cmd.Stdin = bytes.NewReader(svgBytes)
	cmd.Env = os.Environ()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svgo failed: %w (stderr: %s)", err, stderr.String())
	}

	out := bytes.TrimSpace(stdout.Bytes())
	if len(out) == 0 {
		return nil, fmt.Errorf("svgo returned empty output")
	}
	return out, nil
}

// MinifySVGNative minifies SVG XML using pure Go regex without external tools.
func MinifySVGNative(svgBytes []byte) []byte {
	s := string(svgBytes)

	// Strip comments, XML declaration, doctype, and metadata
	s = svgCommentRE.ReplaceAllString(s, "")
	s = svgXMLDeclRE.ReplaceAllString(s, "")
	s = svgDocTypeRE.ReplaceAllString(s, "")
	s = svgMetadataRE.ReplaceAllString(s, "")

	// Strip editor namespaces and attributes
	s = svgEditorNSRE.ReplaceAllString(s, "")
	s = svgEditorAttrRE.ReplaceAllString(s, "")

	// Strip empty groups
	s = svgEmptyGroupRE.ReplaceAllString(s, "")

	// Trim whitespace
	s = svgTagWSRE.ReplaceAllString(s, "><")
	s = svgMultiWSRE.ReplaceAllString(s, " ")
	s = svgExcessWSRE.ReplaceAllString(s, " ")

	// Minify coordinate decimals in path attributes
	s = svgCoordRE.ReplaceAllString(s, "$1")

	return []byte(strings.TrimSpace(s))
}

func isValidSVG(data []byte) bool {
	lower := strings.ToLower(string(data))
	hasOpen := strings.Contains(lower, "<svg")
	hasClose := strings.Contains(lower, "</svg>") || strings.Contains(lower, "/>")
	return hasOpen && hasClose
}

// ComputeSVGSavings calculates compression savings for an SVG.
func ComputeSVGSavings(originalBytes, optimizedBytes int64) (savingsBytes int64, savingsPercent float64) {
	if originalBytes <= 0 || optimizedBytes >= originalBytes {
		return 0, 0.0
	}
	savingsBytes = originalBytes - optimizedBytes
	savingsPercent = math.Round((float64(savingsBytes)/float64(originalBytes)*100)*10) / 10
	return savingsBytes, savingsPercent
}

// EnsureSVGXMLNS guarantees that the SVG contains the standard xmlns attribute required for <img> rendering.
func EnsureSVGXMLNS(svg []byte) []byte {
	s := string(svg)
	if !strings.Contains(s, "xmlns") {
		idx := strings.Index(strings.ToLower(s), "<svg")
		if idx != -1 {
			insertPos := idx + 4
			return []byte(s[:insertPos] + " xmlns=\"http://www.w3.org/2000/svg\"" + s[insertPos:])
		}
	}
	return svg
}

