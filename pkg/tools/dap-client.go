package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type dapSession struct {
	cmd        *exec.Cmd
	conn       net.Conn
	responses  chan dapMessage
	events     chan dapMessage
	readErr    chan error
	writeMu    sync.Mutex
	callMu     sync.Mutex
	seq        int
	tempDir    string
	configured bool
	closeOnce  sync.Once
}

type dapMessage struct {
	Seq        int             `json:"seq"`
	Type       string          `json:"type"`
	RequestSeq int             `json:"request_seq"`
	Command    string          `json:"command"`
	Event      string          `json:"event"`
	Success    bool            `json:"success"`
	Message    string          `json:"message"`
	Body       json.RawMessage `json:"body,omitempty"`
}

func (s *dapSession) readLoop() {
	reader := bufio.NewReaderSize(s.conn, 64<<10)
	for {
		body, err := readLSPFrame(reader)
		if err != nil {
			select {
			case s.readErr <- err:
			default:
			}
			return
		}
		var message dapMessage
		if err := json.Unmarshal(body, &message); err != nil {
			continue
		}
		if message.Type == "response" {
			s.responses <- message
			continue
		}
		select {
		case s.events <- message:
		default:
			if message.Event == "stopped" || message.Event == "terminated" || message.Event == "exited" {
				select {
				case <-s.events:
				default:
				}
				select {
				case s.events <- message:
				default:
				}
			}
		}
	}
}

func (s *dapSession) send(ctx context.Context, command string, args interface{}) (int, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.seq++
	seq := s.seq
	body, err := json.Marshal(map[string]interface{}{
		"seq": seq, "type": "request", "command": command, "arguments": args,
	})
	if err != nil {
		return 0, err
	}
	if len(body) > maxLSPFrame {
		return 0, fmt.Errorf("debugger request exceeds frame limit")
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}
	if _, err := fmt.Fprintf(s.conn, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return 0, err
	}
	_, err = s.conn.Write(body)
	return seq, err
}

func (s *dapSession) call(ctx context.Context, command string, args interface{}, bodyOut interface{}) error {
	s.callMu.Lock()
	defer s.callMu.Unlock()
	seq, err := s.send(ctx, command, args)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-s.readErr:
			return err
		case message := <-s.responses:
			if message.RequestSeq != seq {
				continue
			}
			if !message.Success {
				return fmt.Errorf("debugger rejected %s: %s", command, message.Message)
			}
			if bodyOut != nil && len(message.Body) > 0 {
				return json.Unmarshal(message.Body, bodyOut)
			}
			return nil
		}
	}
}

func (s *dapSession) waitResponses(ctx context.Context, pending map[int]bool) error {
	for len(pending) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-s.readErr:
			return err
		case message := <-s.responses:
			if !pending[message.RequestSeq] {
				continue
			}
			delete(pending, message.RequestSeq)
			if !message.Success {
				return fmt.Errorf("debugger rejected %s: %s", message.Command, message.Message)
			}
		}
	}
	return nil
}

func (s *dapSession) waitEvent(ctx context.Context, wanted ...string) (dapMessage, error) {
	allowed := make(map[string]bool, len(wanted))
	for _, event := range wanted {
		allowed[event] = true
	}
	for {
		select {
		case <-ctx.Done():
			return dapMessage{}, ctx.Err()
		case err := <-s.readErr:
			return dapMessage{}, err
		case message := <-s.events:
			if allowed[message.Event] {
				return message, nil
			}
		}
	}
}

func (s *dapSession) discardQueuedEvents() {
	for {
		select {
		case <-s.events:
		default:
			return
		}
	}
}

func (s *dapSession) close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = s.call(ctx, "disconnect", map[string]bool{"terminateDebuggee": true}, nil)
		cancel()
		closeErr = s.conn.Close()
		killProcessGroup(s.cmd)
		if s.tempDir != "" {
			_ = os.RemoveAll(s.tempDir)
		}
		_ = s.cmd.Wait()
	})
	return closeErr
}
