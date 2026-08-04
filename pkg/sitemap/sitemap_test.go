package sitemap

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"SpeedMap/pkg/config"
)

func TestFetchAndParseUrlSet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url><loc>https://example.com/page1</loc></url>
   <url><loc>https://example.com/page2</loc></url>
</urlset>`))
	}))
	defer ts.Close()

	urls, err := FetchAndParse(ts.URL, config.ScanConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
	if urls[0] != "https://example.com/page1" || urls[1] != "https://example.com/page2" {
		t.Errorf("unexpected URLs: %v", urls)
	}
}

func TestFetchAndParseBasicAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
   <url><loc>https://example.com/protected</loc></url>
</urlset>`))
	}))
	defer ts.Close()

	cfg := config.ScanConfig{
		AuthUser: "admin",
		AuthPass: "secret",
	}

	urls, err := FetchAndParse(ts.URL, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(urls) != 1 || urls[0] != "https://example.com/protected" {
		t.Fatalf("unexpected result: %v", urls)
	}
}
