package browserctl

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxBrowserOutputBytes = 512 * 1024

// NetworkPolicy is applied both before navigation and at the browser egress
// boundary. Local development is opt-in and scoped to explicit host names.
type NetworkPolicy struct {
	AllowLocalDevelopment bool
	LocalHosts            map[string]bool
}

func (p NetworkPolicy) allowsLocal(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if !p.AllowLocalDevelopment {
		return false
	}
	if len(p.LocalHosts) == 0 {
		return host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1"
	}
	return p.LocalHosts[host]
}

func (p NetworkPolicy) Validate(raw string) (*url.URL, error) {
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
	if err := p.validateHost(u.Hostname()); err != nil {
		return nil, err
	}
	return u, nil
}

func (p NetworkPolicy) validateHost(host string) error {
	allowLocal := p.allowsLocal(host)
	ips, err := resolveHost(host)
	if err != nil {
		return fmt.Errorf("resolve browser destination %q: %w", host, err)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) && !allowLocal {
			return fmt.Errorf("private or metadata destination %q is not allowed", host)
		}
	}
	return nil
}

func resolveHost(host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	if n, err := strconv.ParseUint(host, 0, 32); err == nil {
		return []net.IP{net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))}, nil
	}
	return net.LookupIP(strings.TrimSuffix(host, "."))
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}

type EgressProxy struct {
	policy    NetworkPolicy
	ln        net.Listener
	server    *http.Server
	closeOnce sync.Once
}

func StartEgressProxy(policy NetworkPolicy) (*EgressProxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("start browser egress proxy: %w", err)
	}
	p := &EgressProxy{policy: policy, ln: ln}
	p.server = &http.Server{Handler: http.HandlerFunc(p.handle)}
	go func() { _ = p.server.Serve(ln) }()
	return p, nil
}

func (p *EgressProxy) Address() string {
	if p == nil || p.ln == nil {
		return ""
	}
	return p.ln.Addr().String()
}
func (p *EgressProxy) Close() error {
	if p == nil {
		return nil
	}
	var err error
	p.closeOnce.Do(func() { err = p.server.Close() })
	return err
}

func (p *EgressProxy) dial(ctx context.Context, network, host, port string) (net.Conn, error) {
	if err := p.policy.validateHost(host); err != nil {
		return nil, err
	}
	ips, err := resolveHost(host)
	if err != nil {
		return nil, err
	}
	d := &net.Dialer{Timeout: 10 * time.Second}
	for _, ip := range ips {
		if isBlockedIP(ip) && !p.policy.allowsLocal(host) {
			continue
		}
		conn, dialErr := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	return nil, fmt.Errorf("no permitted browser destination for %q: %w", host, err)
}

func (p *EgressProxy) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
		return
	}
	u := r.URL
	if u.Scheme == "" {
		u, _ = url.Parse("http://" + r.Host + r.RequestURI)
	}
	if u == nil {
		http.Error(w, "invalid proxy URL", http.StatusBadRequest)
		return
	}
	if _, err := p.policy.Validate(u.String()); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	r.URL = u
	r.RequestURI = ""
	r.Header.Del("Proxy-Authorization")
	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		return p.dial(ctx, network, host, port)
	}}
	resp, err := transport.RoundTrip(r)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (p *EgressProxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		host, port = r.Host, "443"
	}
	conn, err := p.dial(r.Context(), "tcp", host, port)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = conn.Close()
		http.Error(w, "proxy hijack unavailable", http.StatusInternalServerError)
		return
	}
	client, rw, err := hj.Hijack()
	if err != nil {
		_ = conn.Close()
		return
	}
	_, _ = rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
	_ = rw.Flush()
	go func() { _, _ = io.Copy(conn, client); _ = conn.Close() }()
	_, _ = io.Copy(client, conn)
	_ = client.Close()
}

func boundedBrowserText(s string) string {
	if len(s) <= maxBrowserOutputBytes {
		return s
	}
	return s[:maxBrowserOutputBytes] + "\n...[output truncated]"
}
