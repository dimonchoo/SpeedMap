package optimizer

import (
	"bytes"
	"encoding/base64"
	"image"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	trojanDataURIRegex = regexp.MustCompile(`(?i)data:image\/(png|jpeg|jpg|webp);base64,([A-Za-z0-9+/=\r\n\s]+)`)
	svgWidthRegex      = regexp.MustCompile(`(?i)\bwidth=["']([0-9.]+)(?:px|pt)?["']`)
	svgHeightRegex     = regexp.MustCompile(`(?i)\bheight=["']([0-9.]+)(?:px|pt)?["']`)
	svgViewBoxRegex    = regexp.MustCompile(`(?i)\bviewBox=["'](?:[0-9.-]+\s+){2}([0-9.]+)\s+([0-9.]+)["']`)
)

func parseSVGDimensions(data []byte) (int, int) {
	limit := len(data)
	if limit > 2048 {
		limit = 2048
	}
	head := data[:limit]

	var w, h int
	wMatch := svgWidthRegex.FindSubmatch(head)
	hMatch := svgHeightRegex.FindSubmatch(head)
	if len(wMatch) >= 2 && len(hMatch) >= 2 {
		if wf, err := strconv.ParseFloat(string(wMatch[1]), 64); err == nil {
			w = int(math.Round(wf))
		}
		if hf, err := strconv.ParseFloat(string(hMatch[1]), 64); err == nil {
			h = int(math.Round(hf))
		}
	}
	if w > 0 && h > 0 {
		return w, h
	}

	vbMatch := svgViewBoxRegex.FindSubmatch(head)
	if len(vbMatch) >= 3 {
		if wf, err := strconv.ParseFloat(string(vbMatch[1]), 64); err == nil {
			w = int(math.Round(wf))
		}
		if hf, err := strconv.ParseFloat(string(vbMatch[2]), 64); err == nil {
			h = int(math.Round(hf))
		}
	}
	return w, h
}

// TrojanSVGInfo represents details of a heavy raster image embedded inside an SVG wrapper.
type TrojanSVGInfo struct {
	Format        string  `json:"format"`
	RasterBytes   []byte  `json:"-"`
	NaturalWidth  int     `json:"naturalWidth"`
	NaturalHeight int     `json:"naturalHeight"`
	TotalSVGSize  int64   `json:"totalSvgSize"`
	Base64Size    int64   `json:"base64Size"`
	Base64Ratio   float64 `json:"base64Ratio"`
}

// IsSVGContent checks whether the URL or data buffer represents SVG XML.
func IsSVGContent(rawURL string, data []byte) bool {
	cleanURL := strings.ToLower(strings.Split(rawURL, "?")[0])
	if strings.HasSuffix(cleanURL, ".svg") {
		return true
	}
	if len(data) > 0 {
		limit := len(data)
		if limit > 512 {
			limit = 512
		}
		head := strings.ToLower(string(data[:limit]))
		if strings.Contains(head, "<svg") || strings.Contains(head, "<?xml") {
			return true
		}
	}
	return false
}

// DetectTrojanSVG inspects SVG data for embedded Base64 raster images.
// Returns TrojanSVGInfo and true if the SVG is primarily an embedded raster image.
func DetectTrojanSVG(svgBytes []byte) (*TrojanSVGInfo, bool) {
	if len(svgBytes) == 0 {
		return nil, false
	}
	matches := trojanDataURIRegex.FindSubmatch(svgBytes)
	if len(matches) < 3 {
		return nil, false
	}
	formatStr := strings.ToLower(string(matches[1]))
	if formatStr == "jpg" {
		formatStr = "jpeg"
	}
	b64Bytes := matches[2]
	cleanB64 := bytes.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, b64Bytes)

	rawRaster, err := base64.StdEncoding.DecodeString(string(cleanB64))
	if err != nil || len(rawRaster) == 0 {
		return nil, false
	}

	totalSize := int64(len(svgBytes))
	b64Size := int64(len(cleanB64))
	ratio := float64(b64Size) / float64(totalSize)

	// Trojan threshold: embedded bitmap >= 15KB or base64 constitutes >= 35% of total SVG file
	if len(rawRaster) < 15*1024 && ratio < 0.35 {
		return nil, false
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(rawRaster))
	var w, h int
	if err == nil {
		w = cfg.Width
		h = cfg.Height
	}

	return &TrojanSVGInfo{
		Format:        formatStr,
		RasterBytes:   rawRaster,
		NaturalWidth:  w,
		NaturalHeight: h,
		TotalSVGSize:  totalSize,
		Base64Size:    b64Size,
		Base64Ratio:   ratio,
	}, true
}

type ConversionResult struct {
	URL                 string  `json:"url"`
	Filename            string  `json:"filename"`
	OriginalWidth       int     `json:"originalWidth,omitempty"`
	OriginalHeight      int     `json:"originalHeight,omitempty"`
	OptimizedWidth      int     `json:"optimizedWidth,omitempty"`
	OptimizedHeight     int     `json:"optimizedHeight,omitempty"`
	OriginalBytes       int64   `json:"originalBytes"`
	OriginalFormatted   string  `json:"originalFormatted"`
	OptimizedBytes      int64   `json:"optimizedBytes"`
	OptimizedFormatted  string  `json:"optimizedFormatted"`
	SavingsBytes        int64   `json:"savingsBytes"`
	SavingsFormatted    string  `json:"savingsFormatted"`
	SavingsPercent      float64 `json:"savingsPercent"`
	QualityUsed         float32 `json:"qualityUsed"`
	IsLossless          bool    `json:"isLossless"`
	IsSkipped           bool    `json:"isSkipped"`
	AdaptiveApplied     bool    `json:"adaptiveApplied"`
	IsPaletted          bool    `json:"isPaletted,omitempty"`
	PaletteColors       int     `json:"paletteColors,omitempty"`
	DebandApplied       bool    `json:"debandApplied,omitempty"`
	OriginalDataBase64  string  `json:"originalDataBase64"`
	OptimizedWebPBase64 string  `json:"optimizedWebPBase64"`
	Error               string  `json:"error"`
}

