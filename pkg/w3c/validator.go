package w3c

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type W3CMessage struct {
	Type         string `json:"type"` // "error", "info" (with subType "warning")
	SubType      string `json:"subType,omitempty"`
	LastLine     int    `json:"lastLine"`
	LastColumn   int    `json:"lastColumn"`
	FirstColumn  int    `json:"firstColumn,omitempty"`
	Message      string `json:"message"`
	Extract      string `json:"extract,omitempty"`
	HiliteStart  int    `json:"hiliteStart,omitempty"`
	HiliteLength int    `json:"hiliteLength,omitempty"`
}

type W3CReport struct {
	URL          string       `json:"url"`
	ErrorCount   int          `json:"errorCount"`
	WarningCount int          `json:"warningCount"`
	IsValid      bool         `json:"isValid"`
	Status       string       `json:"status"` // "valid", "warning", "invalid", "error"
	Messages     []W3CMessage `json:"messages"`
	Error        string       `json:"error,omitempty"`
}

type rawW3CResponse struct {
	Messages []W3CMessage `json:"messages"`
}

// Global rate-limiter ensuring max 1 request per second to W3C servers
var (
	rateLimiterMu sync.Mutex
	lastRequestAt time.Time
)

func enforceRateLimit() {
	rateLimiterMu.Lock()
	defer rateLimiterMu.Unlock()

	elapsed := time.Since(lastRequestAt)
	if elapsed < 1*time.Second {
		time.Sleep(1*time.Second - elapsed)
	}
	lastRequestAt = time.Now()
}

// ValidateURL sends page HTML or URL to official W3C Nu HTML Checker API.
// For local development sites (.localdev, localhost, etc.), it fetches HTML locally first and POSTs HTML source to W3C.
func ValidateURL(ctx context.Context, pageURL string) (*W3CReport, error) {
	enforceRateLimit()

	// Check if pageURL is a local domain (.localdev, localhost, 127.0.0.1, .test, .local)
	isLocal := isLocalDomain(pageURL)

	if isLocal {
		// Fetch HTML content locally (ignoring TLS cert errors) and POST to W3C Nu API
		htmlContent, err := fetchHTMLLocally(ctx, pageURL)
		if err != nil {
			return &W3CReport{
				URL:      pageURL,
				Status:   "error",
				Error:    fmt.Sprintf("Не вдалося завантажити HTML локального сайту: %v", err),
				Messages: []W3CMessage{},
			}, nil
		}
		return ValidateHTMLSource(ctx, pageURL, htmlContent)
	}

	// For public web URLs: Try GET via W3C doc parameter first
	apiURL := fmt.Sprintf("https://validator.w3.org/nu/?doc=%s&out=json", url.QueryEscape(pageURL))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create W3C request: %w", err)
	}

	req.Header.Set("User-Agent", "SpeedMap-Validator/1.0 (+https://github.com/SpeedMap)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback: If GET fails or times out, fetch HTML locally and POST
		htmlContent, fetchErr := fetchHTMLLocally(ctx, pageURL)
		if fetchErr == nil && len(htmlContent) > 0 {
			return ValidateHTMLSource(ctx, pageURL, htmlContent)
		}

		errMsg := "Помилка з'єднання з W3C API"
		if err != nil {
			errMsg = err.Error()
		}
		return &W3CReport{
			URL:      pageURL,
			Status:   "error",
			Error:    errMsg,
			Messages: []W3CMessage{},
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read W3C response body: %w", err)
	}

	return ParseW3CJSONResponse(pageURL, body)
}

// ValidateHTMLSource POSTs raw HTML source code directly to W3C Nu Checker API
func ValidateHTMLSource(ctx context.Context, pageURL string, htmlSource string) (*W3CReport, error) {
	apiURL := "https://validator.w3.org/nu/?out=json"

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBufferString(htmlSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create W3C POST request: %w", err)
	}

	req.Header.Set("Content-Type", "text/html; charset=utf-8")
	req.Header.Set("User-Agent", "SpeedMap-Validator/1.0 (+https://github.com/SpeedMap)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &W3CReport{
			URL:      pageURL,
			Status:   "error",
			Error:    fmt.Sprintf("Помилка відправки HTML до W3C API: %v", err),
			Messages: []W3CMessage{},
		}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read W3C response body: %w", err)
	}

	return ParseW3CJSONResponse(pageURL, body)
}

func isLocalDomain(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "localhost" || host == "127.0.0.1" ||
		strings.HasSuffix(host, ".localdev") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".test")
}

func fetchHTMLLocally(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) SpeedMap/1.0")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ParseW3CJSONResponse(pageURL string, jsonBody []byte) (*W3CReport, error) {
	var raw rawW3CResponse
	if err := json.Unmarshal(jsonBody, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse W3C JSON response: %w", err)
	}

	report := &W3CReport{
		URL:      pageURL,
		Messages: raw.Messages,
	}

	for _, msg := range raw.Messages {
		if msg.Type == "error" {
			report.ErrorCount++
		} else if msg.Type == "info" && strings.EqualFold(msg.SubType, "warning") {
			report.WarningCount++
		}
	}

	report.IsValid = (report.ErrorCount == 0)

	if report.ErrorCount > 0 {
		report.Status = "invalid"
	} else if report.WarningCount > 0 {
		report.Status = "warning"
	} else {
		report.Status = "valid"
	}

	return report, nil
}
