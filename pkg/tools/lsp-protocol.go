package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (s *lspServer) send(ctx context.Context, method string, id *int64, params interface{}) error {
	body, err := json.Marshal(lspEnvelope{JSONRPC: "2.0", ID: id, Method: method, Params: mustJSON(params)})
	if err != nil {
		return err
	}
	if len(body) > maxLSPFrame {
		return fmt.Errorf("LSP request exceeds %d-byte frame limit", maxLSPFrame)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if _, err := fmt.Fprintf(s.stdin, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = s.stdin.Write(body)
	return err
}

func (s *lspServer) respond(ctx context.Context, id int64, result interface{}) error {
	rawResult := mustJSON(result)
	if rawResult == nil {
		rawResult = json.RawMessage("null")
	}
	body, err := json.Marshal(lspEnvelope{JSONRPC: "2.0", ID: &id, Result: rawResult})
	if len(body) > maxLSPFrame {
		return fmt.Errorf("LSP response exceeds %d-byte frame limit", maxLSPFrame)
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if _, err := fmt.Fprintf(s.stdin, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = s.stdin.Write(body)
	return err
}

func mustJSON(value interface{}) json.RawMessage {
	if value == nil {
		return nil
	}
	body, _ := json.Marshal(value)
	return body
}

func (s *lspServer) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	s.nextID++
	id := s.nextID
	if err := s.send(ctx, method, &id, params); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-s.readErr:
			return err
		case message := <-s.incoming:
			if message.ID != nil && message.Method != "" {
				var response interface{}
				if message.Method == "workspace/configuration" {
					response = []interface{}{}
				}
				_ = s.respond(context.Background(), *message.ID, response)
				continue
			}
			if message.ID == nil || *message.ID != id {
				continue
			}
			if message.Error != nil {
				return fmt.Errorf("LSP error %d: %s", message.Error.Code, message.Error.Message)
			}
			if result == nil || len(message.Result) == 0 || string(message.Result) == "null" {
				return nil
			}
			return json.Unmarshal(message.Result, result)
		}
	}
}

func (s *lspServer) notify(ctx context.Context, method string, params interface{}) error {
	return s.send(ctx, method, nil, params)
}

func (s *lspServer) initialize(ctx context.Context, workspace string) error {
	root, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	var result map[string]interface{}
	if err := s.call(ctx, "initialize", map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   pathToURI(root),
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"definition": map[string]interface{}{}, "references": map[string]interface{}{},
				"hover": map[string]interface{}{}, "rename": map[string]interface{}{},
			},
		},
	}, &result); err != nil {
		return err
	}
	return s.notify(ctx, "initialized", map[string]interface{}{})
}

func (s *lspServer) didOpen(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return s.notify(ctx, "textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": pathToURI(path), "languageId": languageID(path), "version": 1, "text": string(data),
		},
	})
}
