package agent

import (
	"testing"

	"mncode/pkg/provider"
)

type permissionProbeUI struct {
	allow bool
	calls int
}

func (ui *permissionProbeUI) OnQueryStart()                          {}
func (ui *permissionProbeUI) OnToken(string)                         {}
func (ui *permissionProbeUI) OnThinking(string)                      {}
func (ui *permissionProbeUI) OnToolCallStart(*provider.ToolCall)     {}
func (ui *permissionProbeUI) OnToolCallResult(string, string, bool)  {}
func (ui *permissionProbeUI) OnSubagentStart(string, string, string) {}
func (ui *permissionProbeUI) OnSubagentComplete(string, string)      {}
func (ui *permissionProbeUI) OnGoalDone(string, float64, int, int)   {}
func (ui *permissionProbeUI) OnError(error)                          {}
func (ui *permissionProbeUI) ConfirmToolExecution(*provider.ToolCall) bool {
	ui.calls++
	return ui.allow
}
func (ui *permissionProbeUI) Flush() {}

func TestSubagentPermissionBridgeDelegatesToParent(t *testing.T) {
	parent := &permissionProbeUI{allow: true}
	bridge := newSubagentUI(parent)

	if !bridge.ConfirmToolExecution(&provider.ToolCall{Name: "run_command"}) {
		t.Fatal("expected parent permission decision to be honored")
	}
	if parent.calls != 1 {
		t.Fatalf("parent permission calls = %d, want 1", parent.calls)
	}
}

func TestSubagentPermissionBridgeDeniesWithoutParent(t *testing.T) {
	bridge := newSubagentUI(nil)
	if bridge.ConfirmToolExecution(&provider.ToolCall{Name: "write_to_file"}) {
		t.Fatal("subagent permission must default to deny without a parent UI")
	}
}

var _ UIEventListener = (*permissionProbeUI)(nil)
