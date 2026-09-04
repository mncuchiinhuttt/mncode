package remote

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxRemoteBodyBytes = 1 << 20

// RemotePolicy validates the configured companion endpoint before any secret
// is sent. Local endpoints are opt-in for development only.
type RemotePolicy struct {
	AllowLocalDevelopment bool
	LocalHosts            map[string]bool
}

func (p RemotePolicy) allowsLocal(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if !p.AllowLocalDevelopment {
		return false
	}
	if len(p.LocalHosts) == 0 {
		return host == "localhost" || strings.HasPrefix(host, "127.") || host == "::1"
	}
	return p.LocalHosts[host]
}

func (p RemotePolicy) resolve(raw string) (*url.URL, []net.IP, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("invalid remote URL: %w", err)
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && p.allowsLocal(u.Hostname())) {
		return nil, nil, fmt.Errorf("remote URL must use HTTPS (HTTP is limited to explicit local development hosts)")
	}
	if u.User != nil || u.Hostname() == "" {
		return nil, nil, fmt.Errorf("remote URL must not contain credentials and requires a host")
	}

	host := strings.TrimSuffix(u.Hostname(), ".")
	ips, err := resolveRemoteHost(host)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve remote host %q: %w", u.Hostname(), err)
	}
	for _, ip := range ips {
		if isBlockedRemoteIP(ip) && !p.allowsLocal(host) {
			return nil, nil, fmt.Errorf("private or metadata remote destination %q is not allowed", u.Hostname())
		}
	}
	return u, ips, nil
}

func (p RemotePolicy) Validate(raw string) (*url.URL, error) {
	u, _, err := p.resolve(raw)
	return u, err
}

func resolveRemoteHost(host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	// Parse the integer form accepted by URL clients (for example,
	// 2130706433 == 127.0.0.1) before consulting DNS.
	if n, err := strconv.ParseUint(host, 0, 32); err == nil {
		return []net.IP{net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))}, nil
	}
	return net.LookupIP(host)
}

func isBlockedRemoteIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}


type remoteDestination struct {
	host string
	ips  []net.IP
}

type remoteDestinationKey struct{}

// remoteTransport resolves the policy destination immediately before each
// request, then dials only one of those resolved addresses. Proxy is disabled
// explicitly so HTTP_PROXY/HTTPS_PROXY cannot move a secret-bearing request
// to an ambient intermediary. Keep-alives are disabled so a later request
// cannot reuse a connection established for an earlier DNS result.
type remoteTransport struct {
	policy *RemotePolicy
	base   *http.Transport
}

func newRemoteTransport(policy *RemotePolicy) *remoteTransport {
	t := &remoteTransport{policy: policy}
	t.base = &http.Transport{
		Proxy:           nil,
		DisableKeepAlives: true,
		DialContext:     t.dialContext,
	}
	return t
}

func (t *remoteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("remote transport requires a request URL")
	}
	if t == nil || t.policy == nil {
		return nil, fmt.Errorf("remote transport has no policy")
	}
	u, ips, err := t.policy.resolve(req.URL.String())
	if err != nil {
		return nil, err
	}
	destination := remoteDestination{
		host: strings.ToLower(strings.TrimSuffix(u.Hostname(), ".")),
		ips:  ips,
	}
	ctx := context.WithValue(req.Context(), remoteDestinationKey{}, destination)
	return t.base.RoundTrip(req.Clone(ctx))
}

func (t *remoteTransport) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	destination, ok := ctx.Value(remoteDestinationKey{}).(remoteDestination)
	if !ok || destination.host == "" || len(destination.ips) == 0 {
		return nil, fmt.Errorf("remote destination was not resolved by policy")
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid remote dial address: %w", err)
	}
	if strings.ToLower(strings.TrimSuffix(host, ".")) != destination.host {
		return nil, fmt.Errorf("remote dial host %q does not match policy destination %q", host, destination.host)
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var lastErr error
	for _, ip := range destination.ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	return nil, fmt.Errorf("no permitted remote destination for %q: %w", destination.host, lastErr)
}

// RedactSecrets removes common credential forms before values cross logging
// or error boundaries. It intentionally handles URL query values too.
func RedactSecrets(s string) string {
	bearer := regexp.MustCompile(`(?i)((?:authorization)["']?\s*[=:]\s*(?:bearer|basic)\s+)[^\s,}"]+`)
	s = bearer.ReplaceAllString(s, `${1}[REDACTED]`)
	for _, key := range []string{"apiKey", "api_key", "secretToken", "secret", "token", "authorization", "password"} {
		re := regexp.MustCompile(`(?i)(["']?` + regexp.QuoteMeta(key) + `["']?\s*[=:]\s*["']?)[^&\s,}"']+`)
		s = re.ReplaceAllString(s, `${1}[REDACTED]`)
	}
	return s
}

type RemoteSession struct {
	SessionID   string `json:"sessionId"`
	SecretToken string `json:"secretToken"`
	PairingURL  string `json:"pairingUrl"`
}

type QuestionPayload struct {
	Question      string   `json:"question"`
	Options       []string `json:"options,omitempty"`
	IsMultiSelect bool     `json:"isMultiSelect,omitempty"`
	IsBrainrot    bool     `json:"isBrainrot,omitempty"`
}

type pushTask struct {
	EventType string
	Payload   map[string]interface{}
}

type RemoteManager struct {
	ServerURL   string
	APIKey      string
	Policy      RemotePolicy
	Session     *RemoteSession
	HTTPClient  *http.Client
	IsActive    bool
	LastEventID int64
	Mu          sync.RWMutex
	OnSteer     func(prompt string)
	OnAction    func(answer string)
	OnCancel    func()
	pushChan    chan pushTask
	stopChan    chan struct{}
	stopOnce    sync.Once
	transport   *remoteTransport
}

var (
	globalManager *RemoteManager
	globalMu      sync.RWMutex
)

// GetGlobalRemote returns the active global RemoteManager if any
func GetGlobalRemote() *RemoteManager {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalManager
}

// SetGlobalRemote sets the active global RemoteManager
func SetGlobalRemote(rm *RemoteManager) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalManager = rm
}
func NewRemoteManager(serverURL, apiKey string) *RemoteManager {
	if serverURL == "" {
		if env := os.Getenv("MNCODE_WEB_URL"); env != "" {
			serverURL = env
		} else {
			serverURL = "https://mncode.mncuchiinhuttt.dev"
		}
	}
	serverURL = strings.TrimRight(serverURL, "/")

	rm := &RemoteManager{
		ServerURL: serverURL,
		APIKey:    apiKey,
		Policy:    RemotePolicy{},
		pushChan:  make(chan pushTask, 200),
		stopChan:  make(chan struct{}),
	}
	rm.transport = newRemoteTransport(&rm.Policy)
	rm.HTTPClient = &http.Client{
		Timeout:       10 * time.Second,
		Transport:     rm.transport,
		CheckRedirect: rm.checkRedirect,
	}
	return rm
}

func (rm *RemoteManager) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	origin := via[0].URL
	if origin.Scheme != req.URL.Scheme ||
		strings.ToLower(strings.TrimSuffix(origin.Hostname(), ".")) != strings.ToLower(strings.TrimSuffix(req.URL.Hostname(), ".")) ||
		origin.Port() != req.URL.Port() {
		return fmt.Errorf("remote redirect changes origin")
	}
	return nil
}

func (rm *RemoteManager) do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("remote request is nil")
	}
	rm.Mu.RLock()
	client := rm.HTTPClient
	transport := rm.transport
	rm.Mu.RUnlock()
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if transport == nil {
		transport = newRemoteTransport(&rm.Policy)
	}
	secured := *client
	secured.Transport = transport
	secured.CheckRedirect = rm.checkRedirect
	return secured.Do(req)
}

func (rm *RemoteManager) endpoint(path string, query url.Values) (string, error) {
	base, err := rm.Policy.Validate(rm.ServerURL)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	base.RawQuery = query.Encode()
	return base.String(), nil
}

// InitSession requests a new remote pairing session from the server
func (rm *RemoteManager) InitSession(ctx context.Context, workspaceName string) (*RemoteSession, error) {
	if _, err := rm.Policy.Validate(rm.ServerURL); err != nil {
		return nil, err
	}
	clientOS := fmt.Sprintf("%s (%s)", runtime.GOOS, runtime.GOARCH)

	bodyData := map[string]interface{}{
		"clientOs":      clientOS,
		"workspaceName": workspaceName,
		"apiKey":        rm.APIKey,
	}
	jsonBody, err := json.Marshal(bodyData)
	if err != nil {
		return nil, err
	}
	url, err := rm.endpoint("/api/remote/session", nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rm.do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxRemoteBodyBytes))
		return nil, fmt.Errorf("remote server error (%d): %s", resp.StatusCode, RedactSecrets(string(respBytes)))
	}
	var res struct {
		Success     bool   `json:"success"`
		SessionID   string `json:"sessionId"`
		SecretToken string `json:"secretToken"`
		PairingURL  string `json:"pairingUrl"`
		Error       string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRemoteBodyBytes)).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse remote session response: %w", err)
	}
	if !res.Success {
		return nil, fmt.Errorf("failed to create session: %s", RedactSecrets(res.Error))
	}
	cleanURL := res.PairingURL
	if idx := strings.Index(cleanURL, "?"); idx >= 0 {
		cleanURL = cleanURL[:idx]
	}
	if cleanURL == "" {
		cleanURL = fmt.Sprintf("%s/remote/%s", rm.ServerURL, res.SessionID)
	}
	if _, err := rm.Policy.Validate(cleanURL); err != nil {
		cleanURL = fmt.Sprintf("%s/remote/%s", rm.ServerURL, res.SessionID)
	}

	session := &RemoteSession{
		SessionID:   res.SessionID,
		SecretToken: res.SecretToken,
		PairingURL:  cleanURL,
	}

	rm.Mu.Lock()
	rm.Session = session
	rm.IsActive = true
	rm.Mu.Unlock()

	// Start background workers
	go rm.pushWorker()
	go rm.pollWorker()

	SetGlobalRemote(rm)
	return session, nil
}

// PushTerminalOutput queues terminal text chunks for streaming to the web
func (rm *RemoteManager) PushTerminalOutput(text string) {
	if !rm.IsActive || rm.Session == nil || strings.TrimSpace(text) == "" {
		return
	}
	select {
	case rm.pushChan <- pushTask{
		EventType: "terminal_output",
		Payload:   map[string]interface{}{"text": text},
	}:
	default:
		// Drop if buffer full to avoid blocking terminal
	}
}

// PushAgentStatus updates the current active agent & phase
func (rm *RemoteManager) PushAgentStatus(role, phase, tokens string) {
	if !rm.IsActive || rm.Session == nil {
		return
	}
	select {
	case rm.pushChan <- pushTask{
		EventType: "agent_status",
		Payload: map[string]interface{}{
			"role":   role,
			"phase":  phase,
			"tokens": tokens,
		},
	}:
	default:
	}
}

// PushQuestion pushes an interactive Human-in-the-Loop decision prompt
func (rm *RemoteManager) PushQuestion(q QuestionPayload) {
	if !rm.IsActive || rm.Session == nil {
		return
	}
	select {
	case rm.pushChan <- pushTask{
		EventType: "ask_question",
		Payload: map[string]interface{}{
			"question":      q.Question,
			"options":       q.Options,
			"isMultiSelect": q.IsMultiSelect,
			"isBrainrot":    q.IsBrainrot,
		},
	}:
	default:
	}
}

// PushQuestionResolved notifies the web companion that the question was answered
func (rm *RemoteManager) PushQuestionResolved() {
	if !rm.IsActive || rm.Session == nil {
		return
	}
	select {
	case rm.pushChan <- pushTask{
		EventType: "question_resolved",
		Payload:   map[string]interface{}{"resolved": true},
	}:
	default:
	}
}

func (rm *RemoteManager) pushWorker() {
	for {
		select {
		case <-rm.stopChan:
			return
		case task := <-rm.pushChan:
			rm.sendPush(task)
		}
	}
}

func (rm *RemoteManager) sendPush(task pushTask) {
	rm.Mu.RLock()
	session := rm.Session
	rm.Mu.RUnlock()
	if session == nil {
		return
	}

	payload := map[string]interface{}{
		"sessionId":   session.SessionID,
		"secretToken": session.SecretToken,
		"eventType":   task.EventType,
		"payload":     task.Payload,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	url, err := rm.endpoint("/api/remote/push", nil)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := rm.do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

func (rm *RemoteManager) pollWorker() {
	ticker := time.NewTicker(400 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopChan:
			return
		case <-ticker.C:
			rm.pollIncoming()
		}
	}
}

func (rm *RemoteManager) pollIncoming() {
	rm.Mu.RLock()
	session := rm.Session
	lastID := rm.LastEventID
	rm.Mu.RUnlock()

	if session == nil {
		return
	}

	query := url.Values{}
	query.Set("sessionId", session.SessionID)
	query.Set("secretToken", session.SecretToken)
	query.Set("afterId", fmt.Sprintf("%d", lastID))
	url, err := rm.endpoint("/api/remote/poll", query)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	resp, err := rm.do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var res struct {
		Success bool `json:"success"`
		Events  []struct {
			ID        int64                  `json:"id"`
			EventType string                 `json:"event_type"`
			Payload   map[string]interface{} `json:"payload"`
		} `json:"events"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxRemoteBodyBytes)).Decode(&res); err != nil || !res.Success {
		return
	}

	for _, ev := range res.Events {
		if ev.ID > rm.LastEventID {
			rm.LastEventID = ev.ID
		}

		switch ev.EventType {
		case "user_steer":
			prompt, _ := ev.Payload["prompt"].(string)
			imgBase64, _ := ev.Payload["image"].(string)

			if imgBase64 != "" {
				savedPath, err := saveRemoteImage(imgBase64)
				if err == nil && savedPath != "" {
					if prompt == "" {
						prompt = fmt.Sprintf("Please inspect and analyze this attached image: [Image: %s]", savedPath)
					} else {
						prompt = fmt.Sprintf("%s\n\n[Image: %s]", prompt, savedPath)
					}
					fmt.Printf("\r\n\033[1;38;5;212m[PHOTO] [Mobile Image Received]\033[0m Saved to %s\r\n", savedPath)
				}
			}

			if prompt != "" && rm.OnSteer != nil {
				rm.OnSteer(prompt)
			}
		case "user_action":
			if answer, ok := ev.Payload["answer"].(string); ok && answer != "" {
				if rm.OnAction != nil {
					rm.OnAction(answer)
				}
			}
		case "cancel_run":
			if rm.OnCancel != nil {
				rm.OnCancel()
			}
		}
	}
}

func saveRemoteImage(imgBase64 string) (string, error) {
	idx := strings.Index(imgBase64, ",")
	raw := imgBase64
	ext := ".jpg"
	if idx != -1 {
		header := strings.ToLower(imgBase64[:idx])
		if strings.Contains(header, "png") {
			ext = ".png"
		} else if strings.Contains(header, "webp") {
			ext = ".webp"
		} else if strings.Contains(header, "gif") {
			ext = ".gif"
		}
		raw = imgBase64[idx+1:]
	}

	if len(raw) > (maxRemoteBodyBytes*4/3 + 4) {
		return "", fmt.Errorf("remote image exceeds %d byte limit", maxRemoteBodyBytes)
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	if len(data) > maxRemoteBodyBytes {
		return "", fmt.Errorf("remote image exceeds %d byte limit", maxRemoteBodyBytes)
	}

	dir := filepath.Join(".", ".mncode", "remote_images")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("remote_%d%s", time.Now().UnixNano()/1e6, ext)
	targetPath := filepath.Join(dir, filename)
	if err := os.WriteFile(targetPath, data, 0o600); err != nil {
		return "", err
	}
	return targetPath, nil
}

// Close stops the remote manager and cleans up
func (rm *RemoteManager) Close() {
	rm.Mu.Lock()
	if !rm.IsActive {
		rm.Mu.Unlock()
		return
	}
	rm.IsActive = false
	rm.Mu.Unlock()
	rm.stopOnce.Do(func() { close(rm.stopChan) })
	SetGlobalRemote(nil)
}
