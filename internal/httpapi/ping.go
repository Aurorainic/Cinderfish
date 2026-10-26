package httpapi

import (
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
func decodeOptionalJSON(r *http.Request, dst any) *APIError {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return invalidParamError(map[string]any{"reason": "could not read request body"})
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return invalidParamError(map[string]any{"reason": "body is not valid JSON"})
	}
	return nil
}
