package provider

import (
	"strings"
	"testing"
)

func TestResponsesParserCorrelatesFunctionItemAndUsage(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		`event: response.output_item.added`,
		`data: {"type":"response.output_item.added","item":{"type":"function_call","id":"item_1","call_id":"call_1","name":"lookup"}}`,
		`event: response.function_call_arguments.delta`,
		`data: {"type":"response.function_call_arguments.delta","item_id":"item_1","delta":"{\"q\":\"go\"}"}`,
		`data: {"type":"response.function_call_arguments.done","item_id":"item_1","arguments":"{\"q\":\"go\"}"}`,
		`data: {"type":"response.completed","response":{"usage":{"input_tokens":12,"output_tokens":4}}}`,
	}, "\n"))
	response, err := parseResponsesSSE(stream, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.InputTokens != 12 || response.OutputTokens != 4 {
		t.Fatalf("usage = %d/%d", response.InputTokens, response.OutputTokens)
	}
	if len(response.ToolCalls) != 1 || response.ToolCalls[0].ID != "call_1" || response.ToolCalls[0].Arguments["q"] != "go" {
		t.Fatalf("unexpected tool calls: %+v", response.ToolCalls)
	}
}
