package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// handlePing answers POST|GET /api/ping. Misskey's ping takes no
// parameters and replies with the server's current timestamp.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"pong": time.Now().UnixMilli()})
}

// decodeOptionalJSON decodes a JSON request body into dst. An empty body
// is not an error; malformed JSON yields a Misskey INVALID_PARAM error.
//
// The raw body is kept so the caller can also look up the credential in
// the "i" field, which Misskey accepts alongside the Authorization
// header.
func decodeOptionalJSON(r *http.Request, dst any) *APIError {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil {
		return invalidParamError(map[string]any{"reason": "could not read request body"})
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return invalidParamError(map[string]any{"reason": "body is not valid JSON"})
	}
	return nil
}

// bodyAsMap returns the request body as a generic map. It must be called
// after decodeOptionalJSON, which restores the body for re-reading.
func bodyAsMap(r *http.Request) map[string]any {
	var m map[string]any
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
	if err != nil || len(raw) == 0 {
		return nil
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	_ = json.Unmarshal(raw, &m)
	return m
}
