package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorClass string

const (
	ErrorClassUnknown        ErrorClass = "unknown"
	ErrorClassAuthentication ErrorClass = "authentication"
	ErrorClassAuthorization  ErrorClass = "authorization"
	ErrorClassRateLimit      ErrorClass = "rate_limit"
	ErrorClassServer         ErrorClass = "server"
	ErrorClassNetwork        ErrorClass = "network"
	ErrorClassProtocol       ErrorClass = "protocol"
)

// ProviderError retains provider/status metadata without changing Provider's API.
type ProviderError struct {
	Provider   string
	StatusCode int
	Class      ErrorClass
	Retryable  bool
	RetryAfter time.Duration
	Message    string
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s API error (status %d): %s", e.Provider, e.StatusCode, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s API request failed: %s", e.Provider, e.Message)
	}
	return fmt.Sprintf("%s API request failed", e.Provider)
}
func (e *ProviderError) Unwrap() error { return e.Err }

func classifyStatus(status int) (ErrorClass, bool) {
	switch status {
	case http.StatusUnauthorized:
		return ErrorClassAuthentication, true
	case http.StatusForbidden:
		return ErrorClassAuthorization, true
	case http.StatusTooManyRequests:
		return ErrorClassRateLimit, true
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		return ErrorClassProtocol, false
	case http.StatusRequestTimeout, http.StatusTooEarly:
		return ErrorClassServer, true
	}
	if status >= 500 && status <= 599 {
		return ErrorClassServer, true
	}
	return ErrorClassUnknown, false
}

func newHTTPError(name string, status int, body string, headers http.Header) error {
	class, retryable := classifyStatus(status)
	var retryAfter time.Duration
	if raw := strings.TrimSpace(headers.Get("Retry-After")); raw != "" {
		if seconds, err := strconv.Atoi(raw); err == nil && seconds >= 0 {
			retryAfter = time.Duration(seconds) * time.Second
		} else if when, err := http.ParseTime(raw); err == nil {
			retryAfter = time.Until(when)
			if retryAfter < 0 {
				retryAfter = 0
			}
		}
	}
	return &ProviderError{Provider: name, StatusCode: status, Class: class, Retryable: retryable, RetryAfter: retryAfter, Message: strings.TrimSpace(body)}
}
func newNetworkError(name string, err error) error {
	return &ProviderError{Provider: name, Class: ErrorClassNetwork, Retryable: true, Message: err.Error(), Err: err}
}

func ClassifyError(err error) *ProviderError {
	if err == nil {
		return nil
	}
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &ProviderError{Provider: "provider", Class: ErrorClassNetwork, Retryable: true, Message: err.Error(), Err: err}
	}
	if !errors.Is(err, context.Canceled) && strings.Contains(strings.ToLower(err.Error()), "connection reset") {
		return &ProviderError{Provider: "provider", Class: ErrorClassNetwork, Retryable: true, Message: err.Error(), Err: err}
	}
	return &ProviderError{Provider: "provider", Class: ErrorClassUnknown, Message: err.Error(), Err: err}
}

func IsRetryable(err error) bool {
	pe := ClassifyError(err)
	return pe != nil && pe.Retryable
}
func emitEvent(cb func(StreamEvent) error, event StreamEvent) error {
	if cb == nil {
		return nil
	}
	return cb(event)
}

const maxProviderErrorBody = 1 << 20

func readProviderErrorBody(r io.Reader) string {
	body, _ := io.ReadAll(io.LimitReader(r, maxProviderErrorBody+1))
	if len(body) > maxProviderErrorBody {
		return string(body[:maxProviderErrorBody]) + " [truncated]"
	}
	return string(body)
}
