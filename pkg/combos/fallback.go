package combos

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

// ResolveRoleModels determines the effective primary and fallback models.
func ResolveRoleModels(m ComboMember) (primary string, fallback string) {
	meta := FindRoleMeta(m.Role)

	// Resolve Primary Model
	p := strings.TrimSpace(m.Model)
	if p == "" || strings.EqualFold(p, "auto") {
		primary = meta.AutoPrimaryModel
	} else {
		primary = p
	}

	// Resolve Fallback Model
	fb := strings.TrimSpace(m.FallbackModel)
	if strings.EqualFold(fb, "auto") {
		fallback = meta.AutoFallbackModel
	} else if strings.EqualFold(fb, "none") {
		fallback = ""
	} else if fb != "" {
		fallback = fb
	} else {
		fallback = ""
	}

	return primary, fallback
}

// IsRetryableFailoverError reports whether an error should trigger the fallback model.
func IsRetryableFailoverError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(err.Error())
	patterns := []string{
		"429", "rate limit", "quota", "too many requests", "resource_exhausted",
		"500", "502", "503", "504", "overloaded", "server error", "internal error",
		"bad gateway", "service unavailable", "gateway timeout",
		"connection reset", "broken pipe", "timeout", "deadline exceeded",
		"context deadline exceeded",
	}
	for _, p := range patterns {
		if strings.Contains(msg, p) {
			return true
		}
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// FallbackLogger reports model failover events to UI and telemetry.
type FallbackLogger interface {
	OnModelFallback(role, fromModel, toModel string, cause error)
}

// ExecuteWithModelFallback executes an action with primary model and automatic fallback on transient failure.
func ExecuteWithModelFallback[T any](
	ctx context.Context,
	role string,
	primaryModel string,
	fallbackModel string,
	logger FallbackLogger,
	fn func(ctx context.Context, model string) (T, error),
) (T, string, error) {
	result, err := fn(ctx, primaryModel)
	if err == nil {
		return result, primaryModel, nil
	}

	// If no fallback is configured or error is non-retryable, return original error
	if strings.TrimSpace(fallbackModel) == "" || !IsRetryableFailoverError(err) {
		return result, primaryModel, err
	}

	// Trigger model fallback
	if logger != nil {
		logger.OnModelFallback(role, primaryModel, fallbackModel, err)
	}

	fallbackResult, fallbackErr := fn(ctx, fallbackModel)
	if fallbackErr != nil {
		return fallbackResult, fallbackModel, fmt.Errorf("primary model %s failed (%v); fallback model %s also failed: %w", primaryModel, err, fallbackModel, fallbackErr)
	}

	return fallbackResult, fallbackModel, nil
}
