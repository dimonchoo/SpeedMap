package wpexport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

func getExportsRegistryFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".speedmap")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "exports.json"), nil
}

// RecordExport writes/updates the global export record registry
func RecordExport(record ExportRecord) error {
	path, err := getExportsRegistryFile()
	if err != nil {
		return err
	}

	var records []ExportRecord
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &records)
	}

	// Update existing record with same ID or prepend new
	found := false
	for i, r := range records {
		if r.ID == record.ID || r.PackageDir == record.PackageDir {
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		records = append([]ExportRecord{record}, records...)
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetExportHistory returns all tracked export records, verifying if their directories still exist on disk
func GetExportHistory(domain string) ([]ExportRecord, error) {
	path, err := getExportsRegistryFile()
	if err != nil {
		return nil, err
	}

	var records []ExportRecord
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &records)
	}

	var filtered []ExportRecord
	cleanDomain := strings.ToLower(strings.TrimSpace(domain))
	if strings.HasPrefix(cleanDomain, "http://") || strings.HasPrefix(cleanDomain, "https://") {
		if parsed, err := url.Parse(cleanDomain); err == nil {
			cleanDomain = strings.ToLower(parsed.Host)
		}
	}
	cleanDomain = strings.TrimPrefix(cleanDomain, "www.")
	simpleDomain := cleanDomain
	if dotIdx := strings.Index(simpleDomain, "."); dotIdx != -1 {
		simpleDomain = simpleDomain[:dotIdx]
	}

	for i := range records {
		r := &records[i]
		if _, err := os.Stat(r.PackageDir); err == nil {
			r.ExistsOnDisk = true
		} else {
			r.ExistsOnDisk = false
			continue
		}

		rDomain := strings.ToLower(r.Domain)
		rDir := strings.ToLower(r.PackageDir)

		if cleanDomain == "" || strings.Contains(rDomain, cleanDomain) || strings.Contains(rDir, cleanDomain) || (simpleDomain != "" && (strings.Contains(rDomain, simpleDomain) || strings.Contains(rDir, simpleDomain))) {
			filtered = append(filtered, *r)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	return filtered, nil
}

var (
	exportDiffMu    sync.RWMutex
	exportDiffCache = make(map[string]*ExportDiffReport)
)

// CompareExportPackages compares two on-disk export manifest.json files and produces a detailed diff
func CompareExportPackages(baseManifestPath, currentManifestPath string) (*ExportDiffReport, error) {
	cacheKey := fmt.Sprintf("%s:%s", baseManifestPath, currentManifestPath)
	exportDiffMu.RLock()
	if cached, ok := exportDiffCache[cacheKey]; ok {
		exportDiffMu.RUnlock()
		return cached, nil
	}
	exportDiffMu.RUnlock()
	type rawManifestItem struct {
		SourceURL      string   `json:"sourceUrl"`
		Basename       string   `json:"basename"`
		OriginalBytes  int64    `json:"originalBytes"`
		OptimizedBytes int64    `json:"optimizedBytes"`
		Pages          []string `json:"pages"`
	}

	loadManifest := func(p string) (map[string]interface{}, map[string]rawManifestItem, error) {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read %s: %w", p, err)
		}
		var meta struct {
			Domain    string            `json:"domain"`
			Generated string            `json:"generated"`
			Count     int               `json:"count"`
			Images    []rawManifestItem `json:"images"`
		}
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, nil, fmt.Errorf("failed to parse %s: %w", p, err)
		}
		imgMap := make(map[string]rawManifestItem, len(meta.Images))
		for _, im := range meta.Images {
			imgMap[im.SourceURL] = im
		}
		info := map[string]interface{}{
			"domain":    meta.Domain,
			"generated": meta.Generated,
			"count":     meta.Count,
		}
		return info, imgMap, nil
	}

	baseMeta, baseImgs, err := loadManifest(baseManifestPath)
	if err != nil {
		return nil, err
	}
	currMeta, currImgs, err := loadManifest(currentManifestPath)
	if err != nil {
		return nil, err
	}

	allURLs := make(map[string]bool)
	for u := range baseImgs {
		allURLs[u] = true
	}
	for u := range currImgs {
		allURLs[u] = true
	}

	formatKB := func(b int64) string {
		if b >= 1024*1024 {
			return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
		}
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}

	var files []ExportFileDiff
	var degraded, improved, same, newCount, removedCount int
	var baseTotalWebP, currTotalWebP int64

	for u := range allURLs {
		bIm, hasBase := baseImgs[u]
		cIm, hasCurr := currImgs[u]

		basename := u
		if hasCurr && cIm.Basename != "" {
			basename = cIm.Basename
		} else if hasBase && bIm.Basename != "" {
			basename = bIm.Basename
		} else if idx := strings.LastIndex(u, "/"); idx != -1 && idx < len(u)-1 {
			basename = u[idx+1:]
		}

		var origBytes, bWebP, cWebP int64
		var pages []string

		if hasBase {
			origBytes = bIm.OriginalBytes
			bWebP = bIm.OptimizedBytes
			baseTotalWebP += bWebP
			pages = bIm.Pages
		}
		if hasCurr {
			if origBytes == 0 {
				origBytes = cIm.OriginalBytes
			}
			cWebP = cIm.OptimizedBytes
			currTotalWebP += cWebP
			if len(pages) == 0 {
				pages = cIm.Pages
			}
		}

		delta := cWebP - bWebP
		status := "same"
		if !hasBase && hasCurr {
			status = "new"
			newCount++
		} else if hasBase && !hasCurr {
			status = "removed"
			removedCount++
		} else if delta > 5*1024 {
			status = "degraded"
			degraded++
		} else if delta < -5*1024 {
			status = "improved"
			improved++
		} else {
			status = "same"
			same++
		}

		deltaFormatted := formatKB(delta)
		if delta > 0 {
			deltaFormatted = "+" + deltaFormatted
		}

		files = append(files, ExportFileDiff{
			SourceURL:         u,
			Basename:          basename,
			OriginalBytes:     origBytes,
			OriginalFormatted: formatKB(origBytes),
			BaseWebPBytes:     bWebP,
			BaseWebPFormatted: formatKB(bWebP),
			CurrWebPBytes:     cWebP,
			CurrWebPFormatted: formatKB(cWebP),
			DeltaBytes:        delta,
			DeltaFormatted:    deltaFormatted,
			Status:            status,
			Pages:             pages,
		})
	}

	statusRank := func(s string) int {
		switch s {
		case "degraded":
			return 0
		case "improved":
			return 1
		case "new":
			return 2
		case "removed":
			return 3
		default:
			return 4
		}
	}

	sort.Slice(files, func(i, j int) bool {
		ri := statusRank(files[i].Status)
		rj := statusRank(files[j].Status)
		if ri != rj {
			return ri < rj
		}
		if files[i].Status == "degraded" {
			return files[i].DeltaBytes > files[j].DeltaBytes
		}
		if files[i].Status == "improved" {
			return files[i].DeltaBytes < files[j].DeltaBytes
		}
		if files[i].Status == "new" {
			return files[i].CurrWebPBytes > files[j].CurrWebPBytes
		}
		if files[i].Status == "removed" {
			return files[i].BaseWebPBytes > files[j].BaseWebPBytes
		}
		return files[i].CurrWebPBytes > files[j].CurrWebPBytes
	})

	baseTime := fmt.Sprintf("%v", baseMeta["generated"])
	currTime := fmt.Sprintf("%v", currMeta["generated"])

	rep := &ExportDiffReport{
		BasePackageDir:    filepath.Dir(baseManifestPath),
		BaseTime:          baseTime,
		CurrentPackageDir: filepath.Dir(currentManifestPath),
		CurrentTime:       currTime,
		TotalFiles:        len(files),
		DegradedCount:     degraded,
		ImprovedCount:     improved,
		SameCount:         same,
		NewCount:          newCount,
		RemovedCount:      removedCount,
		BaseTotalWebP:     baseTotalWebP,
		CurrentTotalWebP:  currTotalWebP,
		DeltaTotalWebP:    currTotalWebP - baseTotalWebP,
		Files:             files,
	}

	exportDiffMu.Lock()
	exportDiffCache[cacheKey] = rep
	exportDiffMu.Unlock()

	return rep, nil
}
