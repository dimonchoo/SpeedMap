package main

import (
	"fmt"

	"SpeedMap/pkg/profiles"
)

// ListSiteProfiles fetches all persistent site profiles from ~/.speedmap/profiles.json
func (a *App) ListSiteProfiles() ([]profiles.SiteProfile, error) {
	fmt.Println("[GO LOG] ListSiteProfiles called")
	list, err := profiles.ListProfiles()
	if err != nil {
		fmt.Printf("[GO LOG] ListSiteProfiles error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] ListSiteProfiles returned %d profiles\n", len(list))
	return list, nil
}

// SaveSiteProfile creates or updates a persistent site profile in ~/.speedmap/profiles.json
func (a *App) SaveSiteProfile(p profiles.SiteProfile) (*profiles.SiteProfile, error) {
	fmt.Printf("[GO LOG] SaveSiteProfile called for '%s' (%s)\n", p.Name, p.SitemapURL)
	saved, err := profiles.SaveProfile(p)
	if err != nil {
		fmt.Printf("[GO LOG] SaveSiteProfile error: %v\n", err)
		return nil, err
	}
	fmt.Printf("[GO LOG] SaveSiteProfile success: ID=%s\n", saved.ID)
	return saved, nil
}

// DeleteSiteProfile removes a site profile by ID
func (a *App) DeleteSiteProfile(id string) error {
	fmt.Printf("[GO LOG] DeleteSiteProfile called for ID=%s\n", id)
	err := profiles.DeleteProfile(id)
	if err != nil {
		fmt.Printf("[GO LOG] DeleteSiteProfile error: %v\n", err)
		return err
	}
	fmt.Printf("[GO LOG] DeleteSiteProfile success for ID=%s\n", id)
	return nil
}
