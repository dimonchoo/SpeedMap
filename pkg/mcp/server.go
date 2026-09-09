package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"sync"

	"SpeedMap/pkg/analytics"
	"SpeedMap/pkg/config"
	"SpeedMap/pkg/scanner"
	"SpeedMap/pkg/sitemap"
	"SpeedMap/pkg/wpexport"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents an outgoing JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError defines standard JSON-RPC errors
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP Tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// TextContent is an MCP content item
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult is the response payload for tools/call
type CallToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// Server encapsulates the SpeedMap MCP Server
type Server struct {
	in  io.Reader
	out io.Writer
	mu  sync.Mutex
}

// NewServer creates a new Server instance
func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:  in,
		out: out,
	}
}

// RunStdioServer starts the MCP server on standard input/output
func RunStdioServer() error {
	s := NewServer(os.Stdin, os.Stdout)
	return s.Serve()
}

// Serve reads JSON-RPC messages line by line from input and writes responses
func (s *Server) Serve() error {
	sc := bufio.NewScanner(s.in)
	buf := make([]byte, 1024*1024)
	sc.Buffer(buf, 10*1024*1024)

	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, fmt.Sprintf("Parse error: %v", err))
			continue
		}

		resp := s.handleRequest(&req)
		if resp != nil {
			s.sendResponse(resp)
		}
	}

	return sc.Err()
}

func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "speedmap",
					"version": "1.0.0",
				},
			},
		}

	case "notifications/initialized":
		return nil

	case "ping":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.getToolsList(),
			},
		}

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: CallToolResult{
					Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Invalid arguments: %v", err)}},
					IsError: true,
				},
			}
		}

		result, isErr := s.callTool(callParams.Name, callParams.Arguments)
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			jsonBytes = []byte(fmt.Sprintf("%v", result))
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []TextContent{
					{
						Type: "text",
						Text: string(jsonBytes),
					},
				},
				IsError: isErr,
			},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func (s *Server) getToolsList() []Tool {
	return []Tool{
		{
			Name:        "speedmap_verify_manifest",
			Description: "Verifies an exported SpeedMap WebP package/manifest against a target site (e.g. UAT or Staging). Performs parallel HTTP checks to ensure all WebP images return 200 OK with matching dimensions, and inspects page HTML to verify that raster URLs have been replaced with WebP.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"manifest_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the export package directory or manifest.json file (e.g. /Users/.../speedmap-webp-20260904-135055).",
					},
					"target_url": map[string]interface{}{
						"type":        "string",
						"description": "Target site URL to verify against (e.g. https://uat.infuse.com). If omitted, uses the manifest's original domain.",
					},
					"check_pages": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, fetches page HTML to verify that the old raster image URLs were replaced with WebP URLs (default true).",
					},
					"max_images": map[string]interface{}{
						"type":        "integer",
						"description": "Optional limit on the number of images to check (0 = check all images).",
					},
					"concurrency": map[string]interface{}{
						"type":        "integer",
						"description": "Number of concurrent HTTP workers (1 to 20, default 8).",
					},
				},
				"required": []string{"manifest_path"},
			},
		},
		{
			Name:        "speedmap_scan_url",
			Description: "Audits a single webpage URL using headless Chrome DevTools Protocol to measure real-time Core Web Vitals (TTFB, FCP, LCP, CLS, TBT), Cloudflare CDN cache status & PoP, render-blocking resources, and heavy images.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Target webpage URL to audit (e.g. https://example.com)",
					},
					"is_mobile": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, emulates Mobile 4G. If false, runs in Desktop mode.",
					},
					"auto_scroll": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, automatically scrolls page down to trigger lazy loading.",
					},
					"timeout_sec": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds (default 30).",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "speedmap_scan_sitemap",
			Description: "Parses an XML Sitemap, scans pages concurrently, and computes aggregate site-wide health score, vitals averages, and image savings.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sitemap_url": map[string]interface{}{
						"type":        "string",
						"description": "Sitemap URL to scan (e.g. https://example.com/sitemap.xml)",
					},
					"max_urls": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of URLs to scan from sitemap (default 5).",
					},
					"concurrency": map[string]interface{}{
						"type":        "integer",
						"description": "Number of concurrent worker tabs (1 to 5, default 2).",
					},
					"is_mobile": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable Mobile 4G throttling emulation.",
					},
				},
				"required": []string{"sitemap_url"},
			},
		},
		{
			Name:        "speedmap_resolve_domain",
			Description: "Resolves DNS records and detects whether domain traffic is proxied through Cloudflare CDN.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "URL or domain name to resolve (e.g. https://example.com)",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

func (s *Server) callTool(name string, args map[string]interface{}) (interface{}, bool) {
	switch name {
	case "speedmap_verify_manifest":
		manifestPath, _ := args["manifest_path"].(string)
		if manifestPath == "" {
			return map[string]string{"error": "manifest_path parameter is required"}, true
		}
		targetURL, _ := args["target_url"].(string)

		checkPages := true
		if cp, ok := args["check_pages"].(bool); ok {
			checkPages = cp
		}

		maxImages := 0
		if mi, ok := args["max_images"].(float64); ok && mi > 0 {
			maxImages = int(mi)
		}

		concurrency := 8
		if c, ok := args["concurrency"].(float64); ok && c > 0 {
			concurrency = int(c)
		}

		req := wpexport.ManifestVerifyRequest{
			ManifestPathOrDir: manifestPath,
			TargetURL:         targetURL,
			CheckPages:        checkPages,
			MaxImages:         maxImages,
			Concurrency:       concurrency,
			TimeoutSec:        15,
		}

		resp, err := wpexport.VerifyManifest(context.Background(), req, nil)
		if err != nil {
			return map[string]string{"error": fmt.Sprintf("Verification error: %v", err)}, true
		}
		return resp, !resp.Summary.AllPassed

	case "speedmap_scan_url":
		rawURL, _ := args["url"].(string)
		if rawURL == "" {
			return map[string]string{"error": "url parameter is required"}, true
		}

		isMobile, _ := args["is_mobile"].(bool)
		autoScroll, _ := args["auto_scroll"].(bool)
		timeoutSec := 30
		if t, ok := args["timeout_sec"].(float64); ok && t > 0 {
			timeoutSec = int(t)
		}

		cfg := config.ScanConfig{
			IsMobile:   isMobile,
			AutoScroll: autoScroll,
			TimeoutSec: timeoutSec,
		}

		sc := scanner.NewScanner(cfg)
		defer sc.Cancel()

		res := sc.ScanSingleURL(1, rawURL)
		if res.Error != "" {
			return res, true
		}
		return res, false

	case "speedmap_scan_sitemap":
		sitemapURL, _ := args["sitemap_url"].(string)
		if sitemapURL == "" {
			return map[string]string{"error": "sitemap_url parameter is required"}, true
		}

		maxURLs := 5
		if m, ok := args["max_urls"].(float64); ok && m > 0 {
			maxURLs = int(m)
		}
		concurrency := 2
		if c, ok := args["concurrency"].(float64); ok && c > 0 {
			concurrency = int(c)
		}
		isMobile, _ := args["is_mobile"].(bool)

		cfg := config.ScanConfig{
			Concurrency: concurrency,
			IsMobile:    isMobile,
			TimeoutSec:  30,
		}

		urls, err := sitemap.FetchAndParse(sitemapURL, cfg)
		if err != nil {
			return map[string]string{"error": fmt.Sprintf("Sitemap fetch error: %v", err)}, true
		}

		if len(urls) > maxURLs {
			urls = urls[:maxURLs]
		}

		sc := scanner.NewScanner(cfg)
		defer sc.Cancel()

		results, err := sc.ScanURLs(urls, nil)
		if err != nil {
			return map[string]string{"error": fmt.Sprintf("Scan error: %v", err)}, true
		}

		siteAnalytics := analytics.ComputeSiteAnalytics(results)

		return map[string]interface{}{
			"scanned_count": len(results),
			"analytics":     siteAnalytics,
			"pages":         results,
		}, false

	case "speedmap_resolve_domain":
		rawURL, _ := args["url"].(string)
		if rawURL == "" {
			return map[string]string{"error": "url parameter is required"}, true
		}
		res, err := scanner.ResolveDomain(rawURL)
		if err != nil {
			return map[string]string{"error": fmt.Sprintf("Resolve error: %v", err)}, true
		}
		return res, false

	default:
		return map[string]string{"error": fmt.Sprintf("Unknown tool: %s", name)}, true
	}
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Host
}

func (s *Server) sendResponse(resp *JSONRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _ := json.Marshal(resp)
	fmt.Fprintf(s.out, "%s\n", data)
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	s.sendResponse(resp)
}
