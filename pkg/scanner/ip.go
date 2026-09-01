package scanner

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DomainResolution struct {
	Domain       string   `json:"domain"`
	IPs          []string `json:"ips"`
	PrimaryIP    string   `json:"ip"`
	IsCloudflare bool     `json:"isCloudflare"`
	Provider     string   `json:"provider"`
	ServerHeader string   `json:"serverHeader"`
}

var cloudflareCIDRs = []string{
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/12",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

func isCloudflareIP(ipStr string) bool {
	parsedIP := net.ParseIP(ipStr)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range cloudflareCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil && ipNet.Contains(parsedIP) {
			return true
		}
	}
	return false
}

// ResolveDomain resolves the domain's IP addresses and detects Cloudflare CDN proxying.
func ResolveDomain(targetURL string) (DomainResolution, error) {
	raw := strings.TrimSpace(targetURL)
	if raw == "" {
		return DomainResolution{}, fmt.Errorf("empty URL")
	}

	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return DomainResolution{}, err
	}

	host := u.Hostname()
	if host == "" {
		host = raw
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolver := net.DefaultResolver
	ips, err := resolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return DomainResolution{Domain: host}, err
	}

	ipStrings := make([]string, 0, len(ips))
	isCF := false
	for _, ip := range ips {
		s := ip.String()
		ipStrings = append(ipStrings, s)
		if isCloudflareIP(s) {
			isCF = true
		}
	}

	primary := ""
	if len(ipStrings) > 0 {
		primary = ipStrings[0]
	}

	serverHdr := ""
	provider := "Direct Server"
	if isCF {
		provider = "Cloudflare Proxy"
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	if resp, err := client.Head(u.String()); err == nil {
		defer resp.Body.Close()
		serverHdr = resp.Header.Get("Server")
		if resp.Header.Get("CF-Ray") != "" || strings.EqualFold(serverHdr, "cloudflare") {
			isCF = true
			provider = "Cloudflare Proxy"
		}
	}

	return DomainResolution{
		Domain:       host,
		IPs:          ipStrings,
		PrimaryIP:    primary,
		IsCloudflare: isCF,
		Provider:     provider,
		ServerHeader: serverHdr,
	}, nil
}
