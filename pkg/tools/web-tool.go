package tools

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	maxWebResponseBytes = 512 * 1024
	maxWebContentChars  = 12_000
)

// URLPolicy controls outbound URL access. Local development targets are
// denied by default and must be explicitly opted in by the caller.
type URLPolicy struct {
	AllowLocalDevelopment bool
	LocalHosts            map[string]bool
}

func (p URLPolicy) allowsLocal(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if !p.AllowLocalDevelopment {
		return false
	}
	if len(p.LocalHosts) == 0 {
		return host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1"
	}
	return p.LocalHosts[host]
}

func (p URLPolicy) Validate(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme %q is not allowed", u.Scheme)
	}
	if u.User != nil {
		return nil, fmt.Errorf("URLs containing credentials are not allowed")
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("URL host is required")
	}
	if err := validateHost(u.Hostname(), p.allowsLocal(u.Hostname())); err != nil {
		return nil, err
	}
	return u, nil
}

func validateHost(host string, allowLocal bool) error {
	host = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(host), "."))
	if host == "" {
		return fmt.Errorf("URL host is required")
	}
	if ip := parseObfuscatedIPv4(host); ip != nil {
		if isPrivateOrMetadata(ip) && !allowLocal {
			return fmt.Errorf("private or metadata destination %q is not allowed", host)
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve URL host %q: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateOrMetadata(ip) && !allowLocal {
			return fmt.Errorf("private or metadata destination %q is not allowed", host)
		}
	}
	return nil
}

func parseObfuscatedIPv4(host string) net.IP {
	if ip := net.ParseIP(host); ip != nil {
		return ip
	}
	n, err := strconv.ParseUint(host, 0, 32)
	if err != nil {
		n, err = strconv.ParseUint(host, 10, 32)
	}
	if err != nil {
		return nil
	}
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func isPrivateOrMetadata(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
func policyTransport(policy URLPolicy) *http.Transport {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return &http.Transport{
		// Ambient proxies must not bypass the destination policy.
		Proxy:           nil,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if err := validateHost(host, policy.allowsLocal(host)); err != nil {
				return nil, err
			}
			// Dial the checked address, not the hostname, to avoid a DNS
			// rebinding between policy validation and connection creation.
			ips, err := net.LookupIP(host)
			if err != nil {
				if ip := parseObfuscatedIPv4(host); ip != nil {
					ips = []net.IP{ip}
				} else {
					return nil, err
				}
			}
			for _, ip := range ips {
				if isPrivateOrMetadata(ip) && !policy.allowsLocal(host) {
					continue
				}
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				err = dialErr
			}
			return nil, fmt.Errorf("no permitted address for %q: %w", host, err)
		},
	}
}

// WebTool fetches web page content and converts it to clean markdown.
// Policy is intentionally explicit: zero value permits only public HTTP(S).
type WebTool struct {
	Policy URLPolicy
}

func (w *WebTool) Name() string {
	return "read_url_content"
}

func (w *WebTool) Description() string {
	return "Fetch textual documentation and clean content from a public web URL via HTTP request."
}

func (w *WebTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"Url": map[string]interface{}{
				"type":        "string",
				"description": "URL to fetch content from (e.g. 'https://docs.github.com/en', 'https://pkg.go.dev/net/http').",
			},
		},
	}
}

func (w *WebTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	urlStr, _ := args["Url"].(string)
	if urlStr == "" {
		urlStr, _ = args["url"].(string)
	}
	if strings.TrimSpace(urlStr) == "" {
		return "", fmt.Errorf("Url is required")
	}
	u, err := w.Policy.Validate(urlStr)
	if err != nil {
		return "", err
	}
	urlStr = u.String()

	client := &http.Client{
		Timeout:   20 * time.Second,
		Transport: policyTransport(w.Policy),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if _, err := w.Policy.Validate(req.URL.String()); err != nil {
				return fmt.Errorf("redirect rejected: %w", err)
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP error %d %s", resp.StatusCode, resp.Status)
	}
	body, truncated, err := readBounded(resp.Body, maxWebResponseBytes)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	text := htmlToMarkdown(string(body))
	if len(text) > maxWebContentChars {
		text = text[:maxWebContentChars] + "\n\n...[Content truncated to 12,000 characters]"
	}
	if truncated {
		text += "\n\n...[Response truncated to 512 KiB]"
	}
	return fmt.Sprintf("URL: %s\nStatus: %d\n\n%s", urlStr, resp.StatusCode, text), nil
}

func readBounded(r io.Reader, max int64) ([]byte, bool, error) {
	if max < 1 {
		return nil, true, nil
	}
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > max {
		return data[:max], true, nil
	}
	return data, false, nil
}

func htmlToMarkdown(html string) string {
	// Strip script and style
	reScript := regexp.MustCompile(`(?is)<(script|style|svg|noscript|iframe)[^>]*>.*?</\1>`)
	cleaned := reScript.ReplaceAllString(html, "")

	// Preserve pre/code blocks
	reCode := regexp.MustCompile(`(?is)<pre[^>]*><code[^>]*>(.*?)</code></pre>`)
	cleaned = reCode.ReplaceAllString(cleaned, "\n```\n$1\n```\n")

	// Headings
	reH1 := regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	cleaned = reH1.ReplaceAllString(cleaned, "\n# $1\n")
	reH2 := regexp.MustCompile(`(?is)<h2[^>]*>(.*?)</h2>`)
	cleaned = reH2.ReplaceAllString(cleaned, "\n## $1\n")
	reH3 := regexp.MustCompile(`(?is)<h3[^>]*>(.*?)</h3>`)
	cleaned = reH3.ReplaceAllString(cleaned, "\n### $1\n")

	// Links
	reA := regexp.MustCompile(`(?is)<a[^>]+href="([^"]+)"[^>]*>(.*?)</a>`)
	cleaned = reA.ReplaceAllString(cleaned, "[$2]($1)")

	// Paragraphs & Line breaks
	reP := regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	cleaned = reP.ReplaceAllString(cleaned, "\n$1\n")
	reBr := regexp.MustCompile(`(?i)<br\s*/?>`)
	cleaned = reBr.ReplaceAllString(cleaned, "\n")
	reLi := regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	cleaned = reLi.ReplaceAllString(cleaned, "\n- $1")

	// Strip remaining HTML tags
	reTags := regexp.MustCompile(`<[^>]+>`)
	cleaned = reTags.ReplaceAllString(cleaned, " ")

	// Normalize spaces & blank lines
	reSpaces := regexp.MustCompile(`[ \t]+`)
	cleaned = reSpaces.ReplaceAllString(cleaned, " ")
	reBlankLines := regexp.MustCompile(`\n{3,}`)
	cleaned = reBlankLines.ReplaceAllString(cleaned, "\n\n")

	return strings.TrimSpace(cleaned)
}
