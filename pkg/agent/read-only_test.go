package agent

import (
	"context"
	"testing"

	"mncode/pkg/config"
	"mncode/pkg/provider"
)

func TestReadOnlySubagentBlocksMutatingTools(t *testing.T) {
	session := &Session{ReadOnly: true, Config: config.DefaultConfig()}
	result := session.executeToolCall(context.Background(), &provider.ToolCall{ID: "write", Name: "write_to_file"})
	if !result.IsError || result.Content == "" {
		t.Fatalf("expected read-only block: %+v", result)
	}
	if !readOnlyTool("view_file") || readOnlyTool("ast_edit") {
		t.Fatal("unexpected read-only tool policy")
	}
}
