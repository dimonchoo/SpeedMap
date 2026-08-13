package sitemap

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
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

	parsed, err := url.Parse(sitemapURL)
	if err != nil {
		return nil, fmt.Errorf("некоректний URL sitemap: %w", err)
	}

	isExplicitXML := strings.HasSuffix(strings.ToLower(parsed.Path), ".xml") || strings.HasSuffix(strings.ToLower(parsed.Path), ".xml.gz")

	// If user provided a specific XML sitemap file, fetch it directly
	if isExplicitXML {
		visited := &sync.Map{}
		urls, err := fetchRecursive(sitemapURL, cfg, visited, 0)
		if err == nil && len(urls) > 0 {
			return urls, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("не знайдено дійсних URL у sitemap: %s", sitemapURL)
	}

	// If user provided a base domain or non-XML URL (e.g. "https://infuse.com/"),
	// perform intelligent XML sitemap discovery:
	baseURL := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	// 1. Try robots.txt for Sitemap directives
	robotsSitemaps := discoverFromRobotsTxt(baseURL, cfg)
	for _, smURL := range robotsSitemaps {
		visited := &sync.Map{}
		urls, err := fetchRecursive(smURL, cfg, visited, 0)
		if err == nil && len(urls) > 0 {
			return urls, nil
		}
	}

	// 2. Try standard sitemap endpoints
	standardCandidates := []string{
		baseURL + "/sitemap_index.xml",
		baseURL + "/sitemap.xml",
		baseURL + "/wp-sitemap.xml",
	}
	for _, candidate := range standardCandidates {
		visited := &sync.Map{}
		urls, err := fetchRecursive(candidate, cfg, visited, 0)
		if err == nil && len(urls) > 0 {
			return urls, nil
		}
	}

	// 3. Fallback: try raw URL itself if it happens to be a plain text list of URLs
	visited := &sync.Map{}
	urls, err := fetchRecursive(sitemapURL, cfg, visited, 0)
	if err == nil && len(urls) > 0 {
		return urls, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("не знайдено дійсних XML sitemap або URL для сайту %s (перевірено /sitemap_index.xml, /sitemap.xml, /wp-sitemap.xml та robots.txt)", sitemapURL)
}

func discoverFromRobotsTxt(baseURL string, cfg config.ScanConfig) []string {
	robotsURL := strings.TrimRight(baseURL, "/") + "/robots.txt"
	req, err := http.NewRequest("GET", robotsURL, nil)
	if err != nil {
		return nil
	}
	if cfg.AuthUser != "" || cfg.AuthPass != "" {
		auth := cfg.AuthUser + ":" + cfg.AuthPass
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}
	for _, h := range cfg.Headers {
		if strings.TrimSpace(h.Key) != "" {
			req.Header.Set(h.Key, h.Value)
		}
	}
	req.Header.Set("User-Agent", "SpeedMap-SitemapParser/1.0")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var sitemaps []string
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "sitemap:") {
			sm := strings.TrimSpace(line[len("sitemap:"):])
			if isValidURL(sm) {
				sitemaps = append(sitemaps, sm)
			}
		}
	}
	return deduplicate(sitemaps)
}

func fetchRecursive(rawURL string, cfg config.ScanConfig, visited *sync.Map, depth int) ([]string, error) {
	if depth > 3 {
		return nil, nil // Prevent infinite sitemap index loops
	}

	if _, already := visited.LoadOrStore(rawURL, true); already {
		return nil, nil
	}

	timeoutSec := cfg.NormalizedTimeout()
	if timeoutSec < 90 {
		timeoutSec = 90 // Large WordPress sites (e.g. Yoast SEO) can take 30-50s to dynamically build post sitemaps
	}

	var body []byte
	var lastErr error

	// Retry loop for slow or fluctuating sitemap servers
	for attempt := 1; attempt <= 2; attempt++ {
		req, err := http.NewRequest("GET", rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("не вдалося створити запит для %s: %w", rawURL, err)
		}

		if cfg.AuthUser != "" || cfg.AuthPass != "" {
			auth := cfg.AuthUser + ":" + cfg.AuthPass
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
		}

		for _, h := range cfg.Headers {
			if strings.TrimSpace(h.Key) != "" {
				req.Header.Set(h.Key, h.Value)
			}
		}

		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "SpeedMap-SitemapParser/1.0")
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		client := &http.Client{
			Transport: tr,
			Timeout:   time.Duration(timeoutSec) * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		if resp.StatusCode >= 400 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP помилка %d при отриманні sitemap %s", resp.StatusCode, rawURL)
			time.Sleep(1 * time.Second)
			continue
		}

		var reader io.Reader = resp.Body
		if strings.HasSuffix(strings.ToLower(rawURL), ".gz") || resp.Header.Get("Content-Encoding") == "gzip" {
			gzReader, err := gzip.NewReader(resp.Body)
			if err == nil {
				defer gzReader.Close()
				reader = gzReader
			}
		}

		readBody, err := io.ReadAll(reader)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		body = readBody
		lastErr = nil
		break
	}

	if lastErr != nil {
		return nil, fmt.Errorf("помилка завантаження sitemap %s: %v", rawURL, lastErr)
	}

	// 1. Try parsing as SitemapIndex
	var index SitemapIndex
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charsetReader
	if err := decoder.Decode(&index); err == nil && len(index.Sitemaps) > 0 {
		var (
			allURLs []string
			urlsMu  sync.Mutex
			wg      sync.WaitGroup
			sem     = make(chan struct{}, 4) // Fetch up to 4 sub-sitemaps concurrently
		)

		for _, sm := range index.Sitemaps {
			loc := strings.TrimSpace(sm.Loc)
			if loc == "" {
				continue
			}
			wg.Add(1)
			go func(subURL string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				subURLs, err := fetchRecursive(subURL, cfg, visited, depth+1)
				if err == nil && len(subURLs) > 0 {
					urlsMu.Lock()
					allURLs = append(allURLs, subURLs...)
					urlsMu.Unlock()
				} else if err != nil {
					fmt.Printf("[WARN] Failed to fetch sub-sitemap %s: %v\n", subURL, err)
				}
			}(loc)
		}

		wg.Wait()
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

	return nil, fmt.Errorf("адреса %s не є валідним XML sitemap", rawURL)
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
