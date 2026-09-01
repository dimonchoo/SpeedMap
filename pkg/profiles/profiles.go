package profiles

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"SpeedMap/pkg/config"
)

type SiteProfile struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	SitemapURL    string            `json:"sitemapUrl"`
	Config        config.ScanConfig `json:"config"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	LastScannedAt *time.Time        `json:"lastScannedAt,omitempty"`
}

var profilesMu sync.RWMutex

func GetProfilesFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".speedmap")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles.json"), nil
}

func ListProfiles() ([]SiteProfile, error) {
	profilesMu.RLock()
	defer profilesMu.RUnlock()

	return listProfilesInternal()
}

func listProfilesInternal() ([]SiteProfile, error) {
	filePath, err := GetProfilesFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return []SiteProfile{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read profiles file: %w", err)
	}

	var list []SiteProfile
	if err := json.Unmarshal(data, &list); err != nil {
		return []SiteProfile{}, nil
	}

	trueVal := true
	for i := range list {
		if list[i].Config.AdaptiveQuality == nil {
			list[i].Config.AdaptiveQuality = &trueVal
		}
		if list[i].Config.ResizeToRetina == nil {
			list[i].Config.ResizeToRetina = &trueVal
		}
		if list[i].Config.AutoPruneHistory == nil {
			list[i].Config.AutoPruneHistory = &trueVal
		}
		if list[i].Config.HistoryRetentionRuns == 0 {
			list[i].Config.HistoryRetentionRuns = 20
		}
		if list[i].Config.HistoryRetentionDays == 0 {
			list[i].Config.HistoryRetentionDays = 30
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].UpdatedAt.After(list[j].UpdatedAt)
	})

	return list, nil
}

func SaveProfile(profile SiteProfile) (*SiteProfile, error) {
	profilesMu.Lock()
	defer profilesMu.Unlock()

	profiles, err := listProfilesInternal()
	if err != nil {
		profiles = []SiteProfile{}
	}

	now := time.Now()
	if strings.TrimSpace(profile.Name) == "" {
		profile.Name = DeriveProfileName(profile.SitemapURL)
	}

	if profile.ID == "" {
		profile.ID = fmt.Sprintf("site_%d", now.UnixNano())
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	found := false
	for i, p := range profiles {
		if p.ID == profile.ID {
			// Retain CreatedAt
			if !p.CreatedAt.IsZero() {
				profile.CreatedAt = p.CreatedAt
			}
			profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		profiles = append(profiles, profile)
	}

	filePath, err := GetProfilesFilePath()
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal profiles: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write profiles file: %w", err)
	}

	return &profile, nil
}

func DeleteProfile(id string) error {
	profilesMu.Lock()
	defer profilesMu.Unlock()

	profiles, err := listProfilesInternal()
	if err != nil {
		return err
	}

	var updated []SiteProfile
	for _, p := range profiles {
		if p.ID != id {
			updated = append(updated, p)
		}
	}

	filePath, err := GetProfilesFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

func UpdateLastScanned(id string) {
	profilesMu.Lock()
	defer profilesMu.Unlock()

	profiles, err := listProfilesInternal()
	if err != nil || len(profiles) == 0 {
		return
	}

	now := time.Now()
	for i, p := range profiles {
		if p.ID == id {
			profiles[i].LastScannedAt = &now
			profiles[i].UpdatedAt = now

			filePath, err := GetProfilesFilePath()
			if err == nil {
				data, _ := json.MarshalIndent(profiles, "", "  ")
				_ = os.WriteFile(filePath, data, 0644)
			}
			break
		}
	}
}

func DeriveProfileName(sitemapURL string) string {
	if sitemapURL == "" {
		return "Новий сайт"
	}
	clean := strings.TrimPrefix(sitemapURL, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "www.")
	idx := strings.Index(clean, "/")
	if idx > 0 {
		clean = clean[:idx]
	}
	if clean == "" {
		return "Новий сайт"
	}
	return clean
}
