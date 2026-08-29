package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PeerMessage is the transport-neutral message shape exposed to tool callers.
type PeerMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// PeerMessenger is implemented by the agent's in-process subagent hub.
type PeerMessenger interface {
	SendMessage(context.Context, string, string, string) error
	BroadcastMessage(context.Context, string, string) error
	ReceiveMessage(context.Context, string) (PeerMessage, error)
}

// SubagentPeerTool lets one running subagent coordinate with its peers.
type SubagentPeerTool struct {
	SelfID string
	Hub    PeerMessenger
}

// Name returns the model-facing peer messaging tool name.
func (t *SubagentPeerTool) Name() string { return "subagent_message" }

// Description explains the bounded in-process messaging contract.
func (t *SubagentPeerTool) Description() string {
	return "Send, broadcast, or receive messages from sibling subagents in the same parent run. Messages are in-process and never leave mncode."
}

// Schema returns peer messaging actions and their arguments.
func (t *SubagentPeerTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action":     map[string]interface{}{"type": "string", "enum": []string{"send", "broadcast", "receive"}},
			"to":         map[string]interface{}{"type": "string", "description": "Target subagent ID for send."},
			"message":    map[string]interface{}{"type": "string", "description": "Message body, bounded to 16 KiB."},
			"timeout_ms": map[string]interface{}{"type": "integer", "minimum": 1, "maximum": 30000},
		},
		"required": []string{"action"},
	}
}

// Execute performs a peer send, broadcast, or bounded receive operation.
func (t *SubagentPeerTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if t.Hub == nil || strings.TrimSpace(t.SelfID) == "" {
		return "", fmt.Errorf("peer hub is unavailable")
	}
	action, _ := args["action"].(string)
	message, _ := args["message"].(string)
	if len(message) > 16<<10 {
		return "", fmt.Errorf("message exceeds 16 KiB")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch action {
	case "send":
		to, _ := args["to"].(string)
		if err := t.Hub.SendMessage(ctx, t.SelfID, to, message); err != nil {
			return "", err
		}
		return fmt.Sprintf("message sent to %s", to), nil
	case "broadcast":
		if err := t.Hub.BroadcastMessage(ctx, t.SelfID, message); err != nil {
			return "", err
		}
		return "message broadcast to active peers", nil
	case "receive":
		wait := numberArgument(args, "timeout_ms", 5000)
		if wait < 1 {
			wait = 1
		}
		if wait > 30000 {
			wait = 30000
		}
		waitCtx, cancel := context.WithTimeout(ctx, time.Duration(wait)*time.Millisecond)
		defer cancel()
		message, err := t.Hub.ReceiveMessage(waitCtx, t.SelfID)
		if err != nil {
			return "", err
		}
		body, err := json.Marshal(message)
		if err != nil {
			return "", err
		}
		return string(body), nil
	default:
		return "", fmt.Errorf("unsupported peer action: %s", action)
	}
}
