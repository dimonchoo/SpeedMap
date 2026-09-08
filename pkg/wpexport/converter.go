package wpexport

import (
	"bytes"
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/optimizer"
)

func CollectHeavyImages(images []analytics.AggregatedImage) []ManifestImage {
	byKey := make(map[string]int)
	out := make([]ManifestImage, 0)

	for _, img := range images {
		if img.URL == "" {
			continue
		}
		format := strings.ToLower(img.Format)
		if format == "" {
			format = guessFormat(img.URL)
		}
		if format == "avif" || format == "webp" {
			continue
		}
		if !rasterFormats[format] && format != "svg" {
			continue
		}

		sourceURL := preferOriginalURL(img.URL)
		basename, pathHint := basenameAndHint(sourceURL)
		webpRel := webpRelFromHint(pathHint, basename)
		key := strings.ToLower(webpRel)
		if key == "" || key == ".webp" {
			key = strings.ToLower(strings.Split(sourceURL, "?")[0])
		}

		pages := img.Pages
		if pages == nil {
			pages = []string{}
		}

		if idx, ok := byKey[key]; ok {
			ex := &out[idx]
			ex.Pages = mergePages(ex.Pages, pages)
			if img.IsHeavy {
				ex.IsHeavy = true
			}
			if img.NaturalWidth > ex.NaturalWidth {
				ex.NaturalWidth = img.NaturalWidth
			}
			if img.NaturalHeight > ex.NaturalHeight {
				ex.NaturalHeight = img.NaturalHeight
			}
			if img.MaxRenderedWidth > ex.MaxRenderedWidth {
				ex.MaxRenderedWidth = img.MaxRenderedWidth
			}
			if img.MaxRenderedHeight > ex.MaxRenderedHeight {
				ex.MaxRenderedHeight = img.MaxRenderedHeight
			}
			if img.RecommendedRetinaWidth > ex.RecommendedRetinaWidth {
				ex.RecommendedRetinaWidth = img.RecommendedRetinaWidth
			}
			if img.RecommendedRetinaHeight > ex.RecommendedRetinaHeight {
				ex.RecommendedRetinaHeight = img.RecommendedRetinaHeight
			}
			// Prefer larger source when same webpRel (png vs jpg collision)
			if img.MaxTransferSize > ex.Bytes {
				ex.Bytes = img.MaxTransferSize
				ex.SourceURL = sourceURL
				ex.PathHint = pathHint
				ex.Basename = basename
				ex.Format = format
				ex.WebpRel = webpRel
			} else if hasSizeSuffix(ex.SourceURL) && !hasSizeSuffix(sourceURL) {
				ex.SourceURL = sourceURL
			}
			continue
		}

		byKey[key] = len(out)
		out = append(out, ManifestImage{
			SourceURL:               sourceURL,
			PathHint:                pathHint,
			WebpRel:                 webpRel,
			Basename:                basename,
			Format:                  format,
			IsHeavy:                 img.IsHeavy,
			Bytes:                   img.MaxTransferSize,
			Pages:                   append([]string(nil), pages...),
			NaturalWidth:            img.NaturalWidth,
			NaturalHeight:           img.NaturalHeight,
			MaxRenderedWidth:        img.MaxRenderedWidth,
			MaxRenderedHeight:       img.MaxRenderedHeight,
			RecommendedRetinaWidth:  img.RecommendedRetinaWidth,
			RecommendedRetinaHeight: img.RecommendedRetinaHeight,
		})
	}

	heavy := make([]ManifestImage, 0, len(out))
	for _, im := range out {
		// Include strictly what impacts PageSpeed on the frontend:
		// 1. Confirmed heavy on network during scan (IsHeavy == true)
		// 2. Or lazy-loaded in DOM where network transferSize was 0 (verified on download)
		if im.IsHeavy || im.Bytes == 0 {
			heavy = append(heavy, im)
		}
	}
	return heavy
}

// ConvertHeavyImages downloads and converts each image via pkg/optimizer.
// Does NOT write into WordPress uploads — PHP apply copies from the package.
func ConvertHeavyImages(images []ManifestImage, quality float32, authUser, authPass string) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithThreshold(images, quality, 0, true, true, authUser, authPass)
}

// ConvertHeavyImagesWithThreshold converts images with custom threshold byte budget using concurrent workers.
func ConvertHeavyImagesWithThreshold(images []ManifestImage, quality float32, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithProgress(images, quality, thresholdBytes, adaptive, resizeToRetina, authUser, authPass, nil)
}

// ConvertHeavyImagesWithProgress converts images with custom threshold byte budget and reports live progress.
func ConvertHeavyImagesWithProgress(images []ManifestImage, quality float32, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string, onProgress func(done, total int, name string)) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithProgressExt(images, quality, 80.0, true, thresholdBytes, adaptive, resizeToRetina, authUser, authPass, onProgress)
}

// ConvertHeavyImagesWithProgressExt converts images with custom threshold byte budget, min quality floor, and skip protection.
func ConvertHeavyImagesWithProgressExt(images []ManifestImage, quality float32, minQuality float32, skipIfNoSavings bool, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string, onProgress func(done, total int, name string)) ([]WrittenImage, error) {
	return ConvertHeavyImagesWithProgressAndOverrides(images, quality, minQuality, skipIfNoSavings, thresholdBytes, adaptive, resizeToRetina, authUser, authPass, nil, onProgress)
}

// ConvertHeavyImagesWithProgressAndOverrides converts images with support for per-image Image Studio overrides.
func ConvertHeavyImagesWithProgressAndOverrides(images []ManifestImage, quality float32, minQuality float32, skipIfNoSavings bool, thresholdBytes int64, adaptive bool, resizeToRetina bool, authUser, authPass string, overrides map[string]ImageOverride, onProgress func(done, total int, name string)) ([]WrittenImage, error) {
	if len(images) == 0 {
		return nil, fmt.Errorf("no images to convert")
	}

	numWorkers := runtime.NumCPU() * 4
	if numWorkers < 8 {
		numWorkers = 8
	}
	if numWorkers > 32 {
		numWorkers = 32
	}
	if numWorkers > len(images) {
		numWorkers = len(images)
	}

	type convertTask struct {
		index int
		img   ManifestImage
	}

	type convertResult struct {
		index int
		item  *WrittenImage
		err   error
	}

	tasks := make(chan convertTask, len(images))
	resultsChan := make(chan convertResult, len(images))
	var processed int32
	totalImages := len(images)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasks {
				img := task.img
				func() {
					defer func() {
						done := int(atomic.AddInt32(&processed, 1))
						if onProgress != nil {
							onProgress(done, totalImages, img.Basename)
						}
					}()

					var override *ImageOverride
					if overrides != nil {
						if o, ok := overrides[img.SourceURL]; ok {
							override = &o
						} else if o, ok := overrides[img.WebpRel]; ok {
							override = &o
						}
					}

					// User explicitly chose to skip this image in Image Studio
					if override != nil && override.Skip {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: marked as skipped in studio", img.SourceURL)}
						return
					}

					maxW := 0
					maxH := 0
					if resizeToRetina && img.MaxRenderedWidth > 0 {
						maxW = img.MaxRenderedWidth * 2
						maxH = img.MaxRenderedHeight * 2
					}
					if override != nil {
						if override.Retina && (override.MaxW > 0 || override.MaxH > 0) {
							maxW = override.MaxW
							maxH = override.MaxH
						} else if !override.Retina {
							maxW = 0
							maxH = 0
						}
					}

					var res *optimizer.ConversionResult
					var err error

					if override != nil {
						// Convert with user-specified overrides
						origBytes, fetchErr := optimizer.FetchImageBytes(img.SourceURL, authUser, authPass)
						if fetchErr != nil {
							resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: %w", img.SourceURL, fetchErr)}
							return
						}
						q := override.Quality
						if q <= 0 {
							q = quality
						}
						tuneOpts := optimizer.ImageTuneOptions{
							Quality:  q,
							Lossless: override.Lossless,
							Exact:    override.Exact,
							MaxW:     maxW,
							MaxH:     maxH,
							Dither:   override.Dither,
						}
						res, err = optimizer.ConvertImageBytesTuned(img.SourceURL, origBytes, tuneOpts)
					} else {
						res, err = optimizer.ConvertImageURLToWebPAdaptiveBudgetAuthResizeMinQuality(img.SourceURL, quality, minQuality, skipIfNoSavings, thresholdBytes, adaptive, maxW, maxH, authUser, authPass)
					}

					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: %w", img.SourceURL, err)}
						return
					}
					if res.IsSkipped {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: no WebP savings at min quality floor (%.0f%%)", img.SourceURL, minQuality)}
						return
					}
					webpData, err := decodeDataURL(res.OptimizedWebPBase64)
					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("decode webp %s: %w", img.SourceURL, err)}
						return
					}
					origData, err := decodeDataURL(res.OriginalDataBase64)
					if err != nil {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("decode orig %s: %w", img.SourceURL, err)}
						return
					}

					// If thresholdBytes is set, ensure the actual downloaded original meets the heavy threshold (unless user explicitly tuned/approved it)
					if override == nil && thresholdBytes > 0 && int64(len(origData)) < thresholdBytes {
						resultsChan <- convertResult{index: task.index, err: fmt.Errorf("skip %s: downloaded size %d B < threshold %d B", img.SourceURL, len(origData), thresholdBytes)}
						return
					}

					if res.OriginalWidth > 0 {
						img.NaturalWidth = res.OriginalWidth
					}
					if res.OriginalHeight > 0 {
						img.NaturalHeight = res.OriginalHeight
					}
					if img.MaxRenderedWidth > 0 {
						retW := img.MaxRenderedWidth * 2
						retH := img.MaxRenderedHeight * 2
						if img.NaturalWidth > 0 && retW > img.NaturalWidth {
							retW = img.NaturalWidth
						}
						if img.NaturalHeight > 0 && retH > img.NaturalHeight {
							retH = img.NaturalHeight
						}
						img.RecommendedRetinaWidth = retW
						img.RecommendedRetinaHeight = retH
					}

					img.Quality = res.QualityUsed
					img.IsLossless = res.IsLossless
					img.IsOverridden = (override != nil)

					rel := img.WebpRel
					isVectorSVG := strings.HasPrefix(res.OptimizedWebPBase64, "data:image/svg+xml")
					if isVectorSVG {
						rel = targetRelFromHint(img.PathHint, img.Basename, "svg")
					} else if rel == "" {
						rel = webpRelFromHint(img.PathHint, img.Basename)
					}
					rel = filepath.ToSlash(rel)
					img.WebpRel = rel

					ext := path.Ext(img.Basename)
					if ext == "" {
						ext = "." + img.Format
					}

					optW := res.OptimizedWidth
					optH := res.OptimizedHeight
					if optW == 0 {
						optW = img.NaturalWidth
					}
					if optH == 0 {
						optH = img.NaturalHeight
					}

					written := &WrittenImage{
						ManifestImage:      img,
						OrigExt:            strings.TrimPrefix(strings.ToLower(ext), "."),
						OrigData:           origData,
						WebPData:           webpData,
						OptimizedWidth:     optW,
						OptimizedHeight:    optH,
						OriginalBytes:      res.OriginalBytes,
						OptimizedBytes:     res.OptimizedBytes,
						SavingsPercent:     res.SavingsPercent,
						OriginalFormatted:  res.OriginalFormatted,
						OptimizedFormatted: res.OptimizedFormatted,
					}
					resultsChan <- convertResult{index: task.index, item: written}
				}()
			}
		}()
	}

	for i, img := range images {
		tasks <- convertTask{index: i, img: img}
	}
	close(tasks)

	wg.Wait()
	close(resultsChan)

	rawResults := make([]*WrittenImage, len(images))
	for res := range resultsChan {
		if res.err != nil {
			fmt.Printf("[wpexport] %v\n", res.err)
			continue
		}
		rawResults[res.index] = res.item
	}

	ok := make([]WrittenImage, 0, len(images))
	for _, item := range rawResults {
		if item != nil {
			id := fmt.Sprintf("%03d", len(ok)+1)
			item.ID = id
			if strings.ToLower(item.Format) == "svg" && !bytes.HasPrefix(item.WebPData, []byte("RIFF")) {
				item.PackageWebP = fmt.Sprintf("images/%s/optimized.svg", id)
			} else {
				item.PackageWebP = fmt.Sprintf("images/%s/optimized.webp", id)
			}
			ok = append(ok, *item)
		}
	}

	if len(ok) == 0 {
		return nil, fmt.Errorf("no WebP conversions succeeded (check auth / image URLs)")
	}
	return ok, nil
}

// WriteWebPFiles is deprecated: use ConvertHeavyImages + WriteDeployPackage.
// Kept as a thin wrapper that only converts (does not touch uploads).
