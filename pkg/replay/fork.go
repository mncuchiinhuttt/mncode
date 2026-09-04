package replay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"mncode/pkg/commandutil"
	"mncode/pkg/provider"
)

// Fork reconstructs messages through an event sequence without executing tools.
func (s *Store) Fork(ctx context.Context, req ForkRequest) (ForkResult, error) {
	trace, events, err := s.Load(ctx, req.TraceID)
	if err != nil {
		return ForkResult{}, err
	}
	if !trace.Complete {
		return ForkResult{}, errors.New("cannot fork an incomplete trace")
	}
	if req.ReplayTools {
		return ForkResult{}, errors.New("tool replay is not supported")
	}
	at := req.At
	if at < 0 {
		at = int64(len(events))
	}
	if at == 0 || at > int64(len(events)) {
		return ForkResult{}, errors.New("fork sequence is out of range")
	}
	history := make([]provider.Message, 0)
	for _, event := range events {
		if event.Seq > at || event.Kind != KindMessage {
			continue
		}
		var message provider.Message
		if err := json.Unmarshal(event.Data, &message); err != nil {
			return ForkResult{}, err
		}
		history = append(history, message)
	}
	if len(history) == 0 {
		return ForkResult{}, errors.New("trace prefix contains no conversation messages")
	}
	if err := validateHistory(history); err != nil {
		return ForkResult{}, err
	}
	id := strings.TrimSpace(req.NewSessionID)
	if id == "" {
		id = commandutil.NewID("fork")
	}
	if id == trace.SessionID {
		return ForkResult{}, errors.New("fork must use a fresh session id")
	}
	if err := safeTraceID(id); err != nil {
		return ForkResult{}, err
	}
	return ForkResult{SessionID: id, ParentTraceID: trace.ID, At: at, History: history, Source: trace}, nil
}

func safeTraceID(id string) error {
	if strings.TrimSpace(id) == "" || len(id) > 100 || strings.ContainsAny(id, `/\\`) || strings.Contains(id, "..") {
		return errors.New("invalid trace id")
	}
	return nil
}

func validateHistory(history []provider.Message) error {
	seenUser, seenNonSystem := false, false
	for index, message := range history {
		switch message.Role {
		case provider.RoleSystem:
			if seenNonSystem {
				return errors.New("fork history has system message after conversation start")
			}
		case provider.RoleUser:
			seenUser, seenNonSystem = true, true
		case provider.RoleAssistant:
			seenNonSystem = true
			if len(message.ToolCalls) == 0 {
				continue
			}
			if index+1 >= len(history) || history[index+1].Role != provider.RoleTool || len(history[index+1].ToolResults) == 0 {
				return errors.New("fork history has tool calls without results")
			}
			callIDs := make(map[string]struct{}, len(message.ToolCalls))
			for _, call := range message.ToolCalls {
				if call.ID == "" {
					return errors.New("fork history has tool call without id")
				}
				callIDs[call.ID] = struct{}{}
			}
			resultIDs := make(map[string]struct{}, len(history[index+1].ToolResults))
			for _, result := range history[index+1].ToolResults {
				if result.ToolCallID == "" {
					return errors.New("fork history has tool result without id")
				}
				if _, ok := callIDs[result.ToolCallID]; !ok {
					return errors.New("fork history tool result does not match a tool call")
				}
				resultIDs[result.ToolCallID] = struct{}{}
			}
			if len(resultIDs) != len(callIDs) {
				return errors.New("fork history is missing a tool result")
			}
		case provider.RoleTool:
			if !seenNonSystem || len(message.ToolResults) == 0 || index == 0 || history[index-1].Role != provider.RoleAssistant || len(history[index-1].ToolCalls) == 0 {
				return errors.New("fork history has an orphan tool message")
			}
			seenNonSystem = true
		default:
			return fmt.Errorf("fork history has unsupported message role %q", message.Role)
		}
	}
	if !seenUser {
		return errors.New("fork history has no user message")
	}
	return nil
}
