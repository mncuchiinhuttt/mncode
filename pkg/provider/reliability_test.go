package provider

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestClassifyProviderHTTPStatus(t *testing.T) {
	err := newHTTPError("openai", 429, "slow down", nil)
	classified := ClassifyError(err)
	if classified.Class != ErrorClassRateLimit || !classified.Retryable || classified.StatusCode != 429 {
		t.Fatalf("unexpected classification: %+v", classified)
	}
	if !IsRetryable(err) {
		t.Fatal("429 should be retryable")
	}
	if IsRetryable(newHTTPError("openai", 400, "bad request", nil)) {
		t.Fatal("400 should not be retryable")
	}
}

func TestOpenAIParserOrdersToolCallsAndPropagatesCallbackErrors(t *testing.T) {
	stream := strings.NewReader(strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"two","function":{"name":"second","arguments":"{}"}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"one","function":{"name":"first","arguments":"{}"}}]}}]}`,
		"data: [DONE]",
		"",
	}, "\n"))
	response, err := (&OpenAIProvider{}).parseSSE(stream, func(StreamEvent) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(response.ToolCalls) != 2 || response.ToolCalls[0].ID != "one" || response.ToolCalls[1].ID != "two" {
		t.Fatalf("tool calls were not ordered by stream index: %+v", response.ToolCalls)
	}

	sentinel := errors.New("callback stopped")
	_, err = (&OpenAIProvider{}).parseSSE(strings.NewReader(`data: {"choices":[{"delta":{"content":"hello"}}]}`+"\n"), func(StreamEvent) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error = %v, want %v", err, sentinel)
	}
}

func TestOpenAIParserAcceptsLargeSSELine(t *testing.T) {
	content := strings.Repeat("x", 100_000)
	stream := fmt.Sprintf("data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n", content)
	response, err := (&OpenAIProvider{}).parseSSE(strings.NewReader(stream), nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.Content != content {
		t.Fatalf("content length = %d, want %d", len(response.Content), len(content))
	}
}

func TestClassifyTransientStreamReadFailureAsRetryable(t *testing.T) {
	classified := ClassifyError(io.ErrUnexpectedEOF)
	if classified.Class != ErrorClassNetwork || !classified.Retryable {
		t.Fatalf("unexpected transient classification: %+v", classified)
	}
}
