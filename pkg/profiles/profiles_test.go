package profiles

import (
	"os"
	"testing"

	"SpeedMap/pkg/config"
)

func TestProfileCRUD(t *testing.T) {
	// Create a temp profiles file for testing
	tmpFile, err := os.CreateTemp("", "profiles_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	p1 := SiteProfile{
		Name:       "Infuse Media",
		SitemapURL: "https://infuse.com/sitemap.xml",
		Config: config.ScanConfig{
			Concurrency: 3,
			IsMobile:    true,
		},
	}

	saved, err := SaveProfile(p1)
	if err != nil {
		t.Fatalf("Failed to save profile: %v", err)
	}

	if saved.ID == "" {
		t.Errorf("Expected generated ID, got empty string")
	}

	if saved.Name != "Infuse Media" {
		t.Errorf("Expected name 'Infuse Media', got '%s'", saved.Name)
	}

	list, err := ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}

	if len(list) == 0 {
		t.Fatalf("Expected at least 1 profile in list, got 0")
	}

	// Clean up created profile
	err = DeleteProfile(saved.ID)
	if err != nil {
		t.Fatalf("Failed to delete profile: %v", err)
	}
}

func TestDeriveProfileName(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://infuse.com/sitemap.xml", "infuse.com"},
		{"http://www.sreality.cz/sitemap.xml", "sreality.cz"},
		{"https://example.org:8080/sitemap.xml", "example.org:8080"},
		{"", "Новий сайт"},
	}

	for _, tt := range tests {
		got := DeriveProfileName(tt.url)
		if got != tt.expected {
			t.Errorf("DeriveProfileName(%q) = %q; want %q", tt.url, got, tt.expected)
		}
	}
}
