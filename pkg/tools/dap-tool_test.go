package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestDAPToolSchemaAndValidation(t *testing.T) {
	tool := &DAPTool{}
	schema := tool.Schema()
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("debugger schema has no properties")
	}
	if _, ok := properties["action"]; !ok {
		t.Fatal("debugger schema has no action property")
	}
	if _, err := tool.Execute(context.Background(), map[string]interface{}{}); err == nil {
		t.Fatal("expected missing-action validation error")
	}
}

func TestDAPToolRequiresLaunchBeforeOtherActions(t *testing.T) {
	tool := &DAPTool{WorkspaceDir: t.TempDir()}
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "stack_trace", "session_id": "not-launched",
	})
	if err == nil || !strings.Contains(err.Error(), "not launched") {
		t.Fatalf("expected not-launched error, got %v", err)
	}
}

func TestDAPToolRejectsOutsideProgram(t *testing.T) {
	tool := &DAPTool{WorkspaceDir: t.TempDir()}
	_, err := tool.Execute(context.Background(), map[string]interface{}{
		"action": "launch", "program": "../outside",
	})
	if err == nil {
		t.Fatal("expected workspace boundary error")
	}
}

func TestDAPCallUsesContentLengthFraming(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() { _ = clientConn.Close(); _ = serverConn.Close() })
	session := &dapSession{
		conn:      clientConn,
		responses: make(chan dapMessage, 2),
		events:    make(chan dapMessage, 2),
		readErr:   make(chan error, 1),
	}
	go session.readLoop()
	go func() {
		reader := bufio.NewReader(serverConn)
		body, err := readLSPFrame(reader)
		if err != nil {
			session.readErr <- err
			return
		}
		var request map[string]interface{}
		if err := json.Unmarshal(body, &request); err != nil {
			session.readErr <- err
			return
		}
		id := int(request["seq"].(float64))
		response, _ := json.Marshal(map[string]interface{}{
			"seq": 2, "type": "response", "request_seq": id, "success": true,
			"command": request["command"], "body": map[string]string{"state": "ok"},
		})
		_, _ = fmt.Fprintf(serverConn, "Content-Length: %d\r\n\r\n", len(response))
		_, _ = serverConn.Write(response)
	}()
	var body json.RawMessage
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := session.call(ctx, "threads", map[string]string{}, &body); err != nil {
		t.Fatalf("DAP call failed: %v", err)
	}
	if !strings.Contains(string(body), "state") {
		t.Fatalf("unexpected DAP response: %s", body)
	}
}
