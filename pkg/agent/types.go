package agent

import (
	"mncode/pkg/accounts"
	"mncode/pkg/config"
	"mncode/pkg/provider"
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
	UI           UIEventListener
}
