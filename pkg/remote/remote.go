package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

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
	Session     *RemoteSession
	HTTPClient  *http.Client
	IsActive    bool
	LastEventID int64
	Mu          sync.RWMutex

	// Event Callbacks
	OnSteer  func(prompt string)
	OnAction func(answer string)
	OnCancel func()

	pushChan chan pushTask
	stopChan chan struct{}
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

// NewRemoteManager initializes a new RemoteManager
func NewRemoteManager(serverURL, apiKey string) *RemoteManager {
	if serverURL == "" {
		if env := os.Getenv("MNCODE_WEB_URL"); env != "" {
			serverURL = env
		} else {
			serverURL = "https://mncode.mncuchiinhuttt.dev"
		}
	}
	serverURL = strings.TrimRight(serverURL, "/")

	return &RemoteManager{
		ServerURL:  serverURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		pushChan:   make(chan pushTask, 200),
		stopChan:   make(chan struct{}),
	}
}

// InitSession requests a new remote pairing session from the server
func (rm *RemoteManager) InitSession(ctx context.Context, workspaceName string) (*RemoteSession, error) {
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

	url := fmt.Sprintf("%s/api/remote/session", rm.ServerURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rm.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("remote server error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Success     bool   `json:"success"`
		SessionID   string `json:"sessionId"`
		SecretToken string `json:"secretToken"`
		PairingURL  string `json:"pairingUrl"`
		Error       string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to parse remote session response: %w", err)
	}
	if !res.Success {
		return nil, fmt.Errorf("failed to create session: %s", res.Error)
	}

	cleanURL := res.PairingURL
	if strings.Contains(cleanURL, "?") {
		cleanURL = strings.Split(cleanURL, "?")[0]
	}
	if cleanURL == "" {
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

	url := fmt.Sprintf("%s/api/remote/push", rm.ServerURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rm.HTTPClient.Do(req)
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

	url := fmt.Sprintf("%s/api/remote/poll?sessionId=%s&secretToken=%s&afterId=%d",
		rm.ServerURL, session.SessionID, session.SecretToken, lastID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}

	resp, err := rm.HTTPClient.Do(req)
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

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || !res.Success {
		return
	}

	for _, ev := range res.Events {
		if ev.ID > rm.LastEventID {
			rm.LastEventID = ev.ID
		}

		switch ev.EventType {
		case "user_steer":
			if prompt, ok := ev.Payload["prompt"].(string); ok && prompt != "" {
				if rm.OnSteer != nil {
					rm.OnSteer(prompt)
				}
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

// Close stops the remote manager and cleans up
func (rm *RemoteManager) Close() {
	rm.Mu.Lock()
	if !rm.IsActive {
		rm.Mu.Unlock()
		return
	}
	rm.IsActive = false
	rm.Mu.Unlock()

	close(rm.stopChan)
	SetGlobalRemote(nil)
}
