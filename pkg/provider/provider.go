package provider

import "context"

// AccountIdentifiable is implemented by providers selected from the account router.
type AccountIdentifiable interface{ AccountID() string }

// TokenRefresher allows the execution boundary to refresh credentials before retrying.
type TokenRefresher interface{ RefreshTokenNow() (string, error) }

// Provider is the common interface for LLM backends (Anthropic, OpenAI, Gemini)
type Provider interface {
	Name() string
	Stream(ctx context.Context, req *CompletionRequest, cb func(StreamEvent) error) (*CompletionResponse, error)
}
