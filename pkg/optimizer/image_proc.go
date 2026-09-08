package optimizer

import (
	"image"
	"image/color"
	"math"
	"golang.org/x/image/draw"
)

// hasTransparency checks whether an image has any pixels with Alpha < 255
func hasTransparency(img image.Image) bool {
	bounds := img.Bounds()
	// Sample pixels to quickly detect transparency
	stepY := (bounds.Dy() / 40) + 1
	stepX := (bounds.Dx() / 40) + 1
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			_, _, _, a := img.At(x, y).RGBA()
			if a < 0xfffe {
				return true
			}
		}
	}
	return false
}

// isSmoothGradientOrUI detects whether an image is a flat/smooth gradient UI banner,
// icon, or graphic with low high-frequency noise. These images suffer from color banding
// under lossy compression and should be preserved in lossless WebP.
func isSmoothGradientOrUI(img *image.RGBA) bool {
	if img == nil {
		return false
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 200 || h < 50 {
		return false
	}

	// Must be a banner/cover proportion (e.g. aspect ratio >= 2.2 for horizontal page section/cover banners like CTA/GTS/csw-cover, and max width <= 2560).
	// Standard photos and content graphics (16:9, 16:10, 4:3, 1:1) and massive 4K imagery are excluded so they stay in high-efficiency Lossy WebP.
	aspect := float64(w) / float64(h)
	if aspect < 2.2 || w > 2560 {
		return false
	}

	stepY := (h / 60) + 1
	stepX := (w / 60) + 1

	var totalDelta float64
	var count int
	var highDeltaCount int
	var sumR, sumG, sumB float64
	var pixelCount float64

	absDiff := func(a, b uint8) float64 {
		if a > b {
			return float64(a - b)
		}
		return float64(b - a)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y-1; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X-1; x += stepX {
			idx := (y-bounds.Min.Y)*img.Stride + (x-bounds.Min.X)*4
			idxRight := idx + 4
			idxDown := idx + img.Stride

			r1, g1, b1 := img.Pix[idx], img.Pix[idx+1], img.Pix[idx+2]
			r2, g2, b2 := img.Pix[idxRight], img.Pix[idxRight+1], img.Pix[idxRight+2]
			r3, g3, b3 := img.Pix[idxDown], img.Pix[idxDown+1], img.Pix[idxDown+2]

			deltaX := absDiff(r1, r2) + absDiff(g1, g2) + absDiff(b1, b2)
			deltaY := absDiff(r1, r3) + absDiff(g1, g3) + absDiff(b1, b3)

			if deltaX > 20.0 {
				highDeltaCount++
			}
			if deltaY > 20.0 {
				highDeltaCount++
			}

			totalDelta += deltaX + deltaY
			count += 2

			sumR += float64(r1)
			sumG += float64(g1)
			sumB += float64(b1)
			pixelCount++
		}
	}

	if count == 0 || pixelCount == 0 {
		return false
	}
	meanDelta := totalDelta / float64(count)
	meanR := sumR / pixelCount
	meanG := sumG / pixelCount
	meanB := sumB / pixelCount

	// High edge density indicates photographic content, people, furniture, or complex icons.
	highEdgeRatio := float64(highDeltaCount) / float64(count)
	if highEdgeRatio > 0.045 {
		return false
	}

	// Blue/Cyan/Teal/Dark or Purple/Magenta smooth gradients have low-to-medium meanDelta (< 8.0).
	// These are the exact conditions where YUV 4:2:0 subsampling destroys smooth transitions with banding stripes/rings.
	// Real-world gradients: gts-bg1.png (~1.34), cmo-bg-new1.jpg (~1.74), pav-bg3.jpg (~1.98), dow-bg7-7.jpg (~3.90), soa-bg5.jpg (~7.26).
	isBlueCyanDominant := (meanB > meanR+5.0) || (meanG > meanR+15.0)
	isPurpleOrMagenta := (meanR > 90.0 && meanB > 90.0 && meanG < 140.0)

	return meanDelta < 8.0 && (isBlueCyanDominant || isPurpleOrMagenta)
}

// applyAntiBandingDither adds a subtle 4x4 Bayer dither (±1.5 RGB levels) before lossy encoding
// to break up Mach-band quantization plateaus (from 150px to 2px) and eliminate concentric rings.
func applyAntiBandingDither(src *image.RGBA) *image.RGBA {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	dithered := image.NewRGBA(bounds)

	bayer4x4 := [4][4]float64{
		{-1.5, 0.5, -1.0, 1.0},
		{0.0, -2.0, 0.5, -1.5},
		{-0.5, 1.5, -1.5, 0.5},
		{1.0, -1.0, 0.0, -2.0},
	}

	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v + 0.5)
	}

	idx := 0
	for y := 0; y < h; y++ {
		bayerRow := bayer4x4[y%4]
		for x := 0; x < w; x++ {
			bias := bayerRow[x%4]
			dithered.Pix[idx] = clamp(float64(src.Pix[idx]) + bias)
			dithered.Pix[idx+1] = clamp(float64(src.Pix[idx+1]) + bias)
			dithered.Pix[idx+2] = clamp(float64(src.Pix[idx+2]) + bias)
			dithered.Pix[idx+3] = src.Pix[idx+3]
			idx += 4
		}
	}
	return dithered
}

// applyGradientDeband reconstructs smooth color gradations in paletted or low-color images
// by gently smoothing plateau boundaries while strictly preserving high-contrast edges and fine details.
func applyGradientDeband(src *image.RGBA) *image.RGBA {
	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 3 || h < 3 {
		return src
	}

	out := image.NewRGBA(bounds)
	stride := src.Stride

	// 4x4 Bayer matrix for gentle micro-dithering across gradient plateaus
	bayer := [4][4]float64{
		{-1.0, 0.33, -0.66, 0.66},
		{0.0, -1.33, 0.33, -1.0},
		{-0.33, 1.0, -1.0, 0.33},
		{0.66, -0.66, 0.0, -1.33},
	}

	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v + 0.5)
	}

	absDiff := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}

	for y := 0; y < h; y++ {
		rowIdx := y * stride
		bRow := bayer[y%4]

		for x := 0; x < w; x++ {
			idx := rowIdx + x*4
			r0 := src.Pix[idx]
			g0 := src.Pix[idx+1]
			b0 := src.Pix[idx+2]
			a0 := src.Pix[idx+3]

			// Boundary or transparent pixels remain untouched
			if a0 < 250 || x < 1 || x >= w-1 || y < 1 || y >= h-1 {
				out.Pix[idx] = r0
				out.Pix[idx+1] = g0
				out.Pix[idx+2] = b0
				out.Pix[idx+3] = a0
				continue
			}

			// Check maximum channel delta with 4 immediate orthogonal neighbors
			nIndices := [4]int{
				idx - 4,
				idx + 4,
				idx - stride,
				idx + stride,
			}

			maxDelta := 0
			sumR := int(r0)
			sumG := int(g0)
			sumB := int(b0)
			count := 1

			for _, ni := range nIndices {
				nr := src.Pix[ni]
				ng := src.Pix[ni+1]
				nb := src.Pix[ni+2]

				dr := absDiff(r0, nr)
				dg := absDiff(g0, ng)
				db := absDiff(b0, nb)
				d := dr
				if dg > d {
					d = dg
				}
				if db > d {
					d = db
				}
				if d > maxDelta {
					maxDelta = d
				}

				sumR += int(nr)
				sumG += int(ng)
				sumB += int(nb)
				count++
			}

			// If maxDelta is small (<= 8), this is a flat/gradient region (e.g. sky, smooth background)
			if maxDelta <= 8 {
				// Also include 4 diagonal neighbors for 3x3 kernel smoothing
				dIndices := [4]int{
					idx - stride - 4,
					idx - stride + 4,
					idx + stride - 4,
					idx + stride + 4,
				}
				isSmoothPatch := true
				for _, di := range dIndices {
					dr := absDiff(r0, src.Pix[di])
					dg := absDiff(g0, src.Pix[di+1])
					db := absDiff(b0, src.Pix[di+2])
					if dr > 10 || dg > 10 || db > 10 {
						isSmoothPatch = false
						break
					}
					sumR += int(src.Pix[di])
					sumG += int(src.Pix[di+1])
					sumB += int(src.Pix[di+2])
					count++
				}

				if isSmoothPatch {
					avgR := float64(sumR) / float64(count)
					avgG := float64(sumG) / float64(count)
					avgB := float64(sumB) / float64(count)

					dither := bRow[x%4]
					out.Pix[idx] = clamp(avgR + dither)
					out.Pix[idx+1] = clamp(avgG + dither)
					out.Pix[idx+2] = clamp(avgB + dither)
					out.Pix[idx+3] = a0
					continue
				}
			}

			// Edge / texture detail: preserve exact original pixels
			out.Pix[idx] = r0
			out.Pix[idx+1] = g0
			out.Pix[idx+2] = b0
			out.Pix[idx+3] = a0
		}
	}

	return out
}

// toStraightRGBA converts any decoded image into *image.RGBA with STRAIGHT (unpremultiplied) RGBA bytes,
// avoiding Go's color.RGBAModel premultiplication bug from crushing anti-aliased edge colors to dark/black in libwebp C-API.
func toStraightRGBA(m image.Image) *image.RGBA {
	bounds := m.Bounds()
	if nrgba, ok := m.(*image.NRGBA); ok {
		rgba := &image.RGBA{
			Pix:    make([]uint8, len(nrgba.Pix)),
			Stride: nrgba.Stride,
			Rect:   nrgba.Rect,
		}
		copy(rgba.Pix, nrgba.Pix)
		return rgba
	}
	if rgba, ok := m.(*image.RGBA); ok {
		return rgba
	}
	rgba := image.NewRGBA(bounds)
	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := m.At(x, y)
			if nc, ok := color.NRGBAModel.Convert(c).(color.NRGBA); ok {
				rgba.Pix[idx] = nc.R
				rgba.Pix[idx+1] = nc.G
				rgba.Pix[idx+2] = nc.B
				rgba.Pix[idx+3] = nc.A
			} else {
				r, g, b, a := c.RGBA()
				if a > 0 {
					rgba.Pix[idx] = uint8((r * 255) / a)
					rgba.Pix[idx+1] = uint8((g * 255) / a)
					rgba.Pix[idx+2] = uint8((b * 255) / a)
					rgba.Pix[idx+3] = uint8(a >> 8)
				} else {
					rgba.Pix[idx] = 0
					rgba.Pix[idx+1] = 0
					rgba.Pix[idx+2] = 0
					rgba.Pix[idx+3] = 0
				}
			}
			idx += 4
		}
	}
	return rgba
}

func resizeProportional(src *image.RGBA, maxW, maxH int) *image.RGBA {
	bounds := src.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if origW <= 0 || origH <= 0 || (maxW <= 0 && maxH <= 0) {
		return src
	}

	targetW := origW
	targetH := origH

	if maxW > 0 && targetW > maxW {
		targetW = maxW
		targetH = int(math.Round(float64(origH) * float64(maxW) / float64(origW)))
	}
	if maxH > 0 && targetH > maxH {
		targetH = maxH
		targetW = int(math.Round(float64(origW) * float64(maxH) / float64(origH)))
	}

	if targetW <= 0 {
		targetW = 1
	}
	if targetH <= 0 {
		targetH = 1
	}

	if targetW >= origW && targetH >= origH {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// ConvertImageURLToWebPAdaptiveAuth encodes to WebP with default 100KB heavy threshold budget.
