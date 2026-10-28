package httpapi

import (
	"context"
	"net/http"

	"milnfish/internal/streaming"
)

// AttachStreaming installs and starts the Streaming API endpoint.
//
// The bus lives on the Server so handlers publish unconditionally: an
// event with no subscriber is a cheap no-op, and the bus exists from the
// moment the Server is constructed.
func (s *Server) AttachStreaming(logf func(string, ...any)) *streaming.Bus {
	var followers streaming.ViewerLoader
	if s.pool != nil {
		followers = s
	}

	var markRead func(ctx context.Context, userID string) error
	if s.notifications != nil {
		markRead = s.notifications.MarkAllRead
	}

	var auth streaming.Authenticator
	if s.tokens != nil {
		auth = streamingAuth{tokens: s.tokens}
	}

	s.stream = streaming.New(streaming.Options{
		Bus:         s.streamBus,
		Auth:        auth,
		Followers:   followers,
		Logger:      logf,
		MarkAllRead: markRead,
	})
	return s.streamBus
}

// StreamBus exposes the event bus that carries Streaming API events.
func (s *Server) StreamBus() *streaming.Bus { return s.streamBus }

// handleStreaming serves GET /streaming, the WebSocket endpoint.
func (s *Server) handleStreaming(w http.ResponseWriter, r *http.Request) {
	if s.stream == nil || s.tokens == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	// The streaming endpoint answers with its own handshake, never with
	// the JSON error envelope.
	s.stream.ServeHTTP(w, r)
}
