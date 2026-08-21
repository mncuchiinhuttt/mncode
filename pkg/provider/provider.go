package provider

import "context"

// Provider is the common interface for LLM backends (Anthropic, OpenAI, Gemini)
type Provider interface {
	Name() string
	Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error)
}
