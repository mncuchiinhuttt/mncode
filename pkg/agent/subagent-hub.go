package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrPeerUnavailable = errors.New("subagent peer is unavailable")
	ErrPeerInboxFull   = errors.New("subagent peer inbox is full")
)

// SendMessage delivers one message to a running peer's bounded inbox and records it.
func (r *SubagentRegistry) SendMessage(ctx context.Context, from, to, content string) error {
	if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" || strings.TrimSpace(content) == "" {
		return fmt.Errorf("from, to, and content are required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.RLock()
	inbox, available := r.inboxes[to]
	for _, record := range r.records {
		if record.ID == to {
			available = record.Status == "running"
			break
		}
	}
	r.mu.RUnlock()
	if inbox == nil || !available {
		return fmt.Errorf("%w: %s is not a running peer", ErrPeerUnavailable, to)
	}

	r.messageMu.Lock()
	defer r.messageMu.Unlock()
	message := SubagentMessage{
		ID: fmt.Sprintf("submsg-%d", atomic.AddUint64(&r.messageID, 1)), From: from, To: to,
		Content: content, CreatedAt: time.Now().UTC(),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	select {
	case inbox <- message:
		r.mu.Lock()
		r.messages = append(r.messages, message)
		if len(r.messages) > 2048 {
			r.messages = r.messages[len(r.messages)-2048:]
		}
		r.mu.Unlock()
		return nil
	default:
		return ErrPeerInboxFull
	}
}

func (r *SubagentRegistry) BroadcastMessage(ctx context.Context, from, content string) error {
	r.mu.RLock()
	peers := make([]string, 0, len(r.records))
	for _, record := range r.records {
		if record.ID != from && record.Status == "running" {
			peers = append(peers, record.ID)
		}
	}
	r.mu.RUnlock()
	var firstErr error
	for _, peer := range peers {
		if err := r.SendMessage(ctx, from, peer, content); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
func (r *SubagentRegistry) ReceiveMessage(ctx context.Context, peer string) (SubagentMessage, error) {
	if strings.TrimSpace(peer) == "" {
		return SubagentMessage{}, fmt.Errorf("peer is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	if r.inboxes == nil {
		r.inboxes = make(map[string]chan SubagentMessage)
	}
	var inbox chan SubagentMessage
	for _, record := range r.records {
		if record.ID == peer {
			inbox = r.inboxes[peer]
			if inbox == nil {
				inbox = make(chan SubagentMessage, 32)
				r.inboxes[peer] = inbox
			}
			break
		}
	}
	if inbox == nil {
		r.mu.Unlock()
		return SubagentMessage{}, fmt.Errorf("%w: %s is not registered", ErrPeerUnavailable, peer)
	}
	r.mu.Unlock()
	select {
	case message := <-inbox:
		return message, nil
	case <-ctx.Done():
		return SubagentMessage{}, ctx.Err()
	}
}

// Messages returns a snapshot of all peer messages in send order.
func (r *SubagentRegistry) Messages() []SubagentMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]SubagentMessage(nil), r.messages...)
}
