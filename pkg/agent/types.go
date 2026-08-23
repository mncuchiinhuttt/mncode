package agent

import (
	"strings"
	"sync"

	"mncode/pkg/accounts"
	"mncode/pkg/config"
	"mncode/pkg/mcp"
	"mncode/pkg/provider"
	"mncode/pkg/remote"
	"mncode/pkg/skills"
	"mncode/pkg/tools"
)

// UIEventListener receives events from the agent loop for display
type UIEventListener interface {
	OnQueryStart()
	OnToken(token string)
	OnThinking(thinking string)
	OnToolCallStart(tc *provider.ToolCall)
	OnToolCallResult(name string, result string, isError bool)
	OnSubagentStart(agentName, role, prompt string)
	OnSubagentComplete(agentName string, summary string)
	OnGoalDone(goal string, elapsedSecs float64, turns int, toolCount int)
	OnError(err error)
	ConfirmToolExecution(tc *provider.ToolCall) bool
	Flush()
}

// Session represents a single conversational agent session
type Session struct {
	ID           string
	WorkspaceDir string
	Config       *config.Config
	Provider     provider.Provider
	Tools        *tools.Registry
	Catalog      *skills.Catalog
	Accounts     *accounts.Store
	Router       *accounts.Router
	Tracker      interface {
		Record(model, accountID string, inputTokens, outputTokens int)
	}
	History      []provider.Message
	Subagents    *SubagentRegistry
	CodebaseMap  *CodebaseSummary
	MCP          *mcp.Manager
	UI           UIEventListener
	Remote       *remote.RemoteManager

	QueueMu      sync.Mutex
	SteerQueue   []string
	MessageQueue []string
	IsProcessing bool
}

// IsExecuting reports whether the agent loop is actively processing a turn
func (s *Session) IsExecuting() bool {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	return s.IsProcessing
}

// SetExecuting updates the active processing flag
func (s *Session) SetExecuting(val bool) {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	s.IsProcessing = val
}

// EnqueueSteer adds high-priority steering guidance into the active agent loop
func (s *Session) EnqueueSteer(steerMsg string) {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	s.SteerQueue = append(s.SteerQueue, strings.TrimSpace(steerMsg))
}

// DrainSteer drains all pending steer instructions
func (s *Session) DrainSteer() []string {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	if len(s.SteerQueue) == 0 {
		return nil
	}
	res := s.SteerQueue
	s.SteerQueue = nil
	return res
}

// EnqueueMessage places a message into the upcoming turn queue
func (s *Session) EnqueueMessage(msg string) {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	s.MessageQueue = append(s.MessageQueue, strings.TrimSpace(msg))
}

// DrainMessageQueue retrieves and clears all queued messages
func (s *Session) DrainMessageQueue() []string {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	if len(s.MessageQueue) == 0 {
		return nil
	}
	res := s.MessageQueue
	s.MessageQueue = nil
	return res
}

// HasQueuedMessages returns true if there are queued messages waiting
func (s *Session) HasQueuedMessages() bool {
	s.QueueMu.Lock()
	defer s.QueueMu.Unlock()
	return len(s.MessageQueue) > 0 || len(s.SteerQueue) > 0
}

// EnqueueDefault routes un-prefixed input based on the interrupt_mode setting (queue vs steer)
func (s *Session) EnqueueDefault(rawMsg string) (action string, payload string) {
	trimmed := strings.TrimSpace(rawMsg)
	if trimmed == "" {
		return "", ""
	}

	if strings.HasPrefix(trimmed, "/steer ") {
		guidance := strings.TrimSpace(strings.TrimPrefix(trimmed, "/steer "))
		s.EnqueueSteer(guidance)
		return "steer", guidance
	} else if strings.HasPrefix(trimmed, "/queue ") {
		prompt := strings.TrimSpace(strings.TrimPrefix(trimmed, "/queue "))
		s.EnqueueMessage(prompt)
		return "queue", prompt
	}

	mode := "queue"
	if s.Config != nil {
		mode = strings.ToLower(s.Config.GetSetting("interrupt_mode", "queue"))
	}

	if mode == "steer" {
		s.EnqueueSteer(trimmed)
		return "steer", trimmed
	}

	s.EnqueueMessage(trimmed)
	return "queue", trimmed
}
