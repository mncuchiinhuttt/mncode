package agent

import "mncode/pkg/provider"

// subagentUI keeps delegated output quiet while preserving the parent's
// permission policy for every tool the subagent requests.
type subagentUI struct {
	parent UIEventListener
}

func newSubagentUI(parent UIEventListener) *subagentUI {
	return &subagentUI{parent: parent}
}

func (ui *subagentUI) OnQueryStart()                          {}
func (ui *subagentUI) OnToken(string)                         {}
func (ui *subagentUI) OnThinking(string)                      {}
func (ui *subagentUI) OnToolCallStart(*provider.ToolCall)     {}
func (ui *subagentUI) OnToolCallResult(string, string, bool)  {}
func (ui *subagentUI) OnSubagentStart(string, string, string) {}
func (ui *subagentUI) OnSubagentComplete(string, string)      {}
func (ui *subagentUI) OnGoalDone(string, float64, int, int)   {}
func (ui *subagentUI) OnError(error)                          {}
func (ui *subagentUI) Flush()                                 {}

// ConfirmToolExecution forwards the decision to the parent UI. A missing
// parent is fail-closed so a detached subagent can never self-approve tools.
func (ui *subagentUI) ConfirmToolExecution(call *provider.ToolCall) bool {
	if ui == nil || ui.parent == nil || call == nil {
		return false
	}
	return ui.parent.ConfirmToolExecution(call)
}

var _ UIEventListener = (*subagentUI)(nil)
