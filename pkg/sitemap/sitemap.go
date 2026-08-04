package sitemap

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"SpeedMap/pkg/config"
)

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	URLs    []URL    `xml:"url"`
}

type URL struct {
	Loc string `xml:"loc"`
}

type SitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	Sitemaps []Sitemap `xml:"sitemap"`
}

type Sitemap struct {
	Loc string `xml:"loc"`
}

func FetchAndParse(sitemapURL string, cfg config.ScanConfig) ([]string, error) {
	sitemapURL = strings.TrimSpace(sitemapURL)
	if sitemapURL == "" {
		return nil, fmt.Errorf("sitemap URL cannot be empty")
	}

	// Auto prefix http/https if missing
	if !strings.HasPrefix(sitemapURL, "http://") && !strings.HasPrefix(sitemapURL, "https://") {
		sitemapURL = "https://" + sitemapURL
	}

	visited := make(map[string]bool)
	return fetchRecursive(sitemapURL, cfg, visited, 0)
}

func fetchRecursive(rawURL string, cfg config.ScanConfig, visited map[string]bool, depth int) ([]string, error) {
	if depth > 3 {
		return nil, nil // Prevent infinite sitemap index loops
	}

	if visited[rawURL] {
		return nil, nil
	}
	visited[rawURL] = true

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", rawURL, err)
	}

	// Apply Basic Auth
	if cfg.AuthUser != "" || cfg.AuthPass != "" {
		auth := cfg.AuthUser + ":" + cfg.AuthPass
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	// Apply Custom Headers
	for _, h := range cfg.Headers {
		if strings.TrimSpace(h.Key) != "" {
			req.Header.Set(h.Key, h.Value)
		}
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "SpeedMap-SitemapParser/1.0")
	}

	timeoutSec := cfg.NormalizedTimeout()
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d while fetching sitemap %s", resp.StatusCode, rawURL)
	}

	var reader io.Reader = resp.Body
	if strings.HasSuffix(strings.ToLower(rawURL), ".gz") || resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err == nil {
			defer gzReader.Close()
			reader = gzReader
		}
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body from %s: %w", rawURL, err)
	}

	// 1. Try parsing as SitemapIndex
	var index SitemapIndex
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charsetReader
	if err := decoder.Decode(&index); err == nil && len(index.Sitemaps) > 0 {
		var allURLs []string
		for _, sm := range index.Sitemaps {
			loc := strings.TrimSpace(sm.Loc)
			if loc != "" {
				subURLs, err := fetchRecursive(loc, cfg, visited, depth+1)
				if err == nil {
					allURLs = append(allURLs, subURLs...)
				}
			}
		}
		return deduplicate(allURLs), nil
	}

	// 2. Try parsing as URLSet
	var urlset URLSet
	decoder = xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charsetReader
	if err := decoder.Decode(&urlset); err == nil && len(urlset.URLs) > 0 {
		var urls []string
		for _, u := range urlset.URLs {
			loc := strings.TrimSpace(u.Loc)
			if isValidURL(loc) {
				urls = append(urls, loc)
			}
		}
		return deduplicate(urls), nil
	}

	// 3. Fallback: line-by-line extraction of URLs
	lines := strings.Split(string(body), "\n")
	var fallbackURLs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if isValidURL(line) {
			fallbackURLs = append(fallbackURLs, line)
		}
	}

	if len(fallbackURLs) > 0 {
		return deduplicate(fallbackURLs), nil
	}

	return nil, fmt.Errorf("no valid URLs found in sitemap at %s", rawURL)
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	return input, nil
}

func isValidURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func deduplicate(urls []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, u := range urls {
		if !seen[u] {
			seen[u] = true
			result = append(result, u)
		}
	}
	return result
}
