package provider

import (
	"strings"
	"testing"
)

func TestAntigravityParserReadsUsageMetadata(t *testing.T) {
	stream := strings.NewReader("data: {\"response\":{\"usageMetadata\":{\"promptTokenCount\":4200,\"candidatesTokenCount\":900,\"thoughtsTokenCount\":1800},\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"done\"}]}}]}}\n\ndata: [DONE]\n")
	response, err := (&AntigravityProvider{}).parseSSE(stream, func(StreamEvent) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if response.InputTokens != 4200 || response.OutputTokens != 900 || response.ThinkingTokens != 1800 {
		t.Fatalf("unexpected usage: input=%d output=%d thinking=%d", response.InputTokens, response.OutputTokens, response.ThinkingTokens)
	}
}
