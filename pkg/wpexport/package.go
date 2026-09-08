package wpexport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SpeedMap/pkg/config"
)

func WriteWebPFiles(wordpressPath string, images []ManifestImage, quality float32, authUser, authPass string) ([]WrittenImage, int, error) {
	_ = wordpressPath
	written, err := ConvertHeavyImages(images, quality, authUser, authPass)
	if err != nil {
		return nil, 0, err
	}
	return written, len(written), nil
}

// WriteDeployPackage writes a self-contained folder:
// apply.php, rollback.php, compare.html, manifest.json, images/NNN/{original,optimized}.
// PHP apply copies optimized.webp → wp uploads/{webpRel} and retargets attachments.
func WriteDeployPackage(packageDir, domain string, cfg config.ScanConfig, written []WrittenImage) (*ExportResult, error) {
	if len(written) == 0 {
		return nil, fmt.Errorf("no images for deploy package")
	}
	pkg := strings.TrimSpace(packageDir)
	if pkg == "" {
		return nil, fmt.Errorf("package dir is required")
	}
	if err := os.MkdirAll(pkg, 0755); err != nil {
		return nil, err
	}

	for _, im := range written {
		id := im.ID
		if id == "" {
			return nil, fmt.Errorf("missing package id for %s", im.Basename)
		}
		imgDir := filepath.Join(pkg, "images", id)
		if err := os.MkdirAll(imgDir, 0755); err != nil {
			return nil, err
		}
		optFile := "optimized.webp"
		if strings.ToLower(im.Format) == "svg" && !bytes.HasPrefix(im.WebPData, []byte("RIFF")) {
			optFile = "optimized.svg"
		}
		if err := os.WriteFile(filepath.Join(imgDir, optFile), im.WebPData, 0644); err != nil {
			return nil, err
		}
	}

	manifestImgs := ManifestImagesFromWritten(written)
	php, err := BuildApplyPHPFromManifest(domain, cfg, pkg, manifestImgs)
	if err != nil {
		return nil, err
	}
	applyPath := filepath.Join(pkg, "apply.php")
	if err := os.WriteFile(applyPath, []byte(php), 0644); err != nil {
		return nil, err
	}
	rollbackPath := filepath.Join(pkg, "rollback.php")
	if err := os.WriteFile(rollbackPath, []byte(BuildRollbackPHP(pkg)), 0644); err != nil {
		return nil, err
	}

	zipBytes, err := BuildReviewZIP(domain, written)
	if err != nil {
		return nil, err
	}
	// Also materialize compare.html + manifest.json beside apply.php (same as ZIP root).
	zr, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != "compare.html" && f.Name != "manifest.json" && f.Name != "render-report.html" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(pkg, f.Name), data, 0644); err != nil {
			return nil, err
		}
	}

	reviewZipPath := filepath.Join(pkg, "review.zip")
	if err := os.WriteFile(reviewZipPath, zipBytes, 0644); err != nil {
		return nil, err
	}

	var totalOrig, totalOpt int64
	for _, im := range written {
		totalOrig += im.OriginalBytes
		totalOpt += im.OptimizedBytes
	}
	savPct := 0.0
	if totalOrig > 0 {
		savPct = float64(totalOrig-totalOpt) / float64(totalOrig) * 100
	}

	_ = RecordExport(ExportRecord{
		ID:             filepath.Base(pkg),
		Domain:         domain,
		Timestamp:      time.Now(),
		FormattedTime:  time.Now().Format("02.01.2006 15:04:05"),
		PackageDir:     pkg,
		ManifestPath:   filepath.Join(pkg, "manifest.json"),
		ReviewZIP:      reviewZipPath,
		ApplyPHP:       applyPath,
		RollbackPHP:    rollbackPath,
		CompareHTML:    filepath.Join(pkg, "compare.html"),
		RenderReport:   filepath.Join(pkg, "render-report.html"),
		ImageCount:     len(written),
		OriginalBytes:  totalOrig,
		OptimizedBytes: totalOpt,
		SavingsPercent: savPct,
		ExistsOnDisk:   true,
	})

	return &ExportResult{
		ApplyPHP:      applyPath,
		RollbackPHP:   rollbackPath,
		ReviewZIP:     reviewZipPath,
		PackageDir:    pkg,
		WebPCount:     len(written),
		WordPressPath: pkg,
	}, nil
}

// BuildReviewZIP packs orig + webp + compare.html + manifest.json for task handoff.

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// buildPackageZIP zips an on-disk deploy package directory (apply.php + images/ + …).
func buildPackageZIP(packageDir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.Walk(packageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(packageDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeZipFile(zw, rel, data)
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildApplyPHP creates a self-contained WP-CLI eval-file script with embedded manifest.
