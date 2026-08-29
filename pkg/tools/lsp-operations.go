package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (s *lspServer) locations(ctx context.Context, method, path string, position lspPosition) ([]lspLocation, error) {
	var raw json.RawMessage
	if err := s.call(ctx, method, map[string]interface{}{
		"textDocument": map[string]string{"uri": pathToURI(path)}, "position": position,
	}, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	if raw[0] == '[' {
		var locations []lspLocation
		if err := json.Unmarshal(raw, &locations); err == nil {
			valid := len(locations) == 0
			for _, location := range locations {
				if location.URI != "" {
					valid = true
					break
				}
			}
			if valid {
				return locations, nil
			}
		}
		var links []lspLocationLink
		if err := json.Unmarshal(raw, &links); err != nil {
			return nil, err
		}
		locationsFromLinks := make([]lspLocation, 0, len(links))
		for _, link := range links {
			locationsFromLinks = append(locationsFromLinks, lspLocation{URI: link.TargetURI, Range: link.TargetSelectionRange})
		}
		return locationsFromLinks, nil
	}
	var location lspLocation
	if err := json.Unmarshal(raw, &location); err != nil {
		return nil, err
	}
	return []lspLocation{location}, nil
}

func (s *lspServer) references(ctx context.Context, path string, position lspPosition) ([]lspLocation, error) {
	var locations []lspLocation
	err := s.call(ctx, "textDocument/references", map[string]interface{}{
		"textDocument": map[string]string{"uri": pathToURI(path)}, "position": position,
		"context": map[string]bool{"includeDeclaration": true},
	}, &locations)
	return locations, err
}

func (s *lspServer) hover(ctx context.Context, path string, position lspPosition) (string, error) {
	var raw json.RawMessage
	if err := s.call(ctx, "textDocument/hover", map[string]interface{}{
		"textDocument": map[string]string{"uri": pathToURI(path)}, "position": position,
	}, &raw); err != nil {
		return "", err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return "No hover information.", nil
	}
	var hover struct {
		Contents json.RawMessage `json:"contents"`
	}
	if err := json.Unmarshal(raw, &hover); err != nil {
		return "", err
	}
	return fmt.Sprintf("Hover: %s", formatMarkup(hover.Contents)), nil
}

type lspDiagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity"`
	Message  string   `json:"message"`
	Source   string   `json:"source"`
}

type lspDiagnosticsParams struct {
	URI         string          `json:"uri"`
	Diagnostics []lspDiagnostic `json:"diagnostics"`
}

func (s *lspServer) diagnostics(ctx context.Context, path string) (string, error) {
	deadline := time.NewTimer(1200 * time.Millisecond)
	defer deadline.Stop()
	uri := pathToURI(path)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "No diagnostics published.", nil
		case err := <-s.readErr:
			return "", err
		case message := <-s.incoming:
			if message.Method != "textDocument/publishDiagnostics" {
				continue
			}
			var payload lspDiagnosticsParams
			if err := json.Unmarshal(message.Params, &payload); err != nil || payload.URI != uri {
				continue
			}
			return formatDiagnostics(payload.Diagnostics), nil
		}
	}
}

func formatMarkup(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	var marked struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &marked) == nil && marked.Value != "" {
		return marked.Value
	}
	return string(raw)
}

func formatDiagnostics(items []lspDiagnostic) string {
	if len(items) == 0 {
		return "No diagnostics published."
	}
	var builder strings.Builder
	for _, item := range items {
		severity := "info"
		if item.Severity == 1 {
			severity = "error"
		} else if item.Severity == 2 {
			severity = "warning"
		}
		fmt.Fprintf(&builder, "%s:%d:%d [%s] %s", item.Source, item.Range.Start.Line+1, item.Range.Start.Character+1, severity, item.Message)
		builder.WriteByte('\n')
	}
	return strings.TrimSpace(builder.String())
}
