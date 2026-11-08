package mail

import (
	"context"
	"sync"
)

// SwitchableSender wraps a Sender and lets the active sender be replaced at
// runtime (the admin console edits the SMTP settings without a restart).
type SwitchableSender struct {
	mu    sync.RWMutex
	inner Sender
}

// NewSwitchableSender wraps an initial sender.
func NewSwitchableSender(inner Sender) *SwitchableSender {
	return &SwitchableSender{inner: inner}
}

// Switch replaces the active sender.
func (s *SwitchableSender) Switch(inner Sender) {
	s.mu.Lock()
	s.inner = inner
	s.mu.Unlock()
}

// Send delivers through the active sender.
func (s *SwitchableSender) Send(ctx context.Context, m Message) error {
	s.mu.RLock()
	inner := s.inner
	s.mu.RUnlock()
	return inner.Send(ctx, m)
}
