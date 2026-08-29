package codex

import (
	"encoding/json"
	"errors"
)

// ProtocolVersion pinned and supported by this integration.
const ProtocolVersion = "2024-11-05"

var (
	ErrRuntimeNotFound      = errors.New("official codex executable not found; install via 'npm i -g @openai/codex' or visit https://github.com/openai/codex")
	ErrIncompatibleVersion  = errors.New("incompatible codex app-server version")
	ErrAuthFailed           = errors.New("codex authentication failed")
	ErrSandboxViolation     = errors.New("requested capability violates codex sandbox approval policy")
	ErrKeyringUnavailable   = errors.New("secure OS keyring unavailable for codex credentials")
)

// Request is a JSON-RPC 2.0 request envelope.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response envelope.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// Notification is a JSON-RPC 2.0 notification envelope.
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RPCError represents a JSON-RPC error.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// InitializeParams parameters for app-server initialization.
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
	Capabilities    ClientCapabilities     `json:"capabilities"`
	WorkspaceRoots  []string               `json:"workspaceRoots,omitempty"`
	Settings        map[string]interface{} `json:"settings,omitempty"`
}

// ClientInfo describes this client.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ClientCapabilities declares supported features.
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
}

// LoginStartParams initiates ChatGPT OAuth login.
type LoginStartParams struct {
	Type string `json:"type"` // "chatgpt" or "chatgptDeviceCode"
}

// LoginStartResult returned when starting OAuth.
type LoginStartResult struct {
	Type            string `json:"type"`
	AuthURL         string `json:"authUrl,omitempty"`
	VerificationURI string `json:"verificationUri,omitempty"`
	UserCode        string `json:"userCode,omitempty"`
	ExpiresIn       int    `json:"expiresIn,omitempty"`
}

// AccountReadResult returned from account/read.
type AccountReadResult struct {
	Account *AccountInfo `json:"account,omitempty"`
}

// AccountInfo represents authenticated user profile (no secrets).
type AccountInfo struct {
	Email       string `json:"email,omitempty"`
	AccountID   string `json:"accountId,omitempty"`
	PlanType    string `json:"planType,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}
