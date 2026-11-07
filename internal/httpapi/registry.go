package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// errIDNoSuchRegistryKey is Misskey's NO_SUCH_KEY id
// (packages/backend/src/server/api/endpoints/i/registry/get.ts).
const errIDNoSuchRegistryKey = "ac3ed68a-62f0-422b-a7bc-d5e09e8f6a6a"

// registryRequest is the shared parameter shape of the i/registry endpoints:
// a key, a scope and the cross-instance domain (which Cinderfish keeps in the
// signature for client compatibility but stores as local-only).
type registryRequest struct {
	Key    string          `json:"key"`
	Scope  []string        `json:"scope"`
	Value  json.RawMessage `json:"value"`
	Domain *string         `json:"domain"`
}

// requireRegistry reports whether the service is wired.
func (s *Server) requireRegistry(w http.ResponseWriter) bool {
	if s.registry == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return false
	}
	return true
}

// registryAuth decodes the body and resolves the caller in one step.
func (s *Server) registryAuth(w http.ResponseWriter, r *http.Request, body *registryRequest) (string, bool) {
	if err := decodeOptionalJSON(r, body); err != nil {
		writeAPIError(w, err)
		return "", false
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return "", false
	}
	return info.User.ID, true
}

func writeNoSuchRegistryKey(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "No such key.", Code: "NO_SUCH_KEY",
		ID: errIDNoSuchRegistryKey, Kind: "client", Status: http.StatusBadRequest,
	})
}

// handleRegistrySet implements POST /api/i/registry/set.
func (s *Server) handleRegistrySet(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	if body.Key == "" {
		writeAPIError(w, invalidParamError("key is required"))
		return
	}
	if err := s.registry.Set(r.Context(), userID, body.Key, body.Scope, body.Value); err != nil {
		writeInternalServerError(w)
		return
	}
	s.publishRegistryUpdated(userID, body.Scope, body.Key, body.Value)
	writeJSON(w, http.StatusOK, struct{}{})
}

// handleRegistryGet implements POST /api/i/registry/get: the raw stored value.
func (s *Server) handleRegistryGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	value, err := s.registry.Get(r.Context(), userID, body.Key, body.Scope)
	if err != nil {
		if errors.Is(err, account.ErrNoSuchRegistryKey) {
			writeNoSuchRegistryKey(w)
			return
		}
		writeInternalServerError(w)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(value)
}

// handleRegistryGetDetail implements POST /api/i/registry/get-detail.
func (s *Server) handleRegistryGetDetail(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	entry, err := s.registry.GetDetail(r.Context(), userID, body.Key, body.Scope)
	if err != nil {
		if errors.Is(err, account.ErrNoSuchRegistryKey) {
			writeNoSuchRegistryKey(w)
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"updatedAt": entry.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"value":     entry.Value,
	})
}

// handleRegistryKeys implements POST /api/i/registry/keys.
func (s *Server) handleRegistryKeys(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	keys, err := s.registry.Keys(r.Context(), userID, body.Scope)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, keys)
}

// handleRegistryGetAll implements POST /api/i/registry/get-all.
func (s *Server) handleRegistryGetAll(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	all, err := s.registry.All(r.Context(), userID, body.Scope)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make(map[string]any, len(all))
	for key, value := range all {
		out[key] = value
	}
	writeJSON(w, http.StatusOK, out)
}

// handleRegistryRemove implements POST /api/i/registry/remove.
func (s *Server) handleRegistryRemove(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	if err := s.registry.Remove(r.Context(), userID, body.Key, body.Scope); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// handleRegistryScopes implements POST /api/i/registry/scopes: the distinct
// scopes the caller has written to, as string[][].
func (s *Server) handleRegistryScopes(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	scopes, err := s.registry.Scopes(r.Context(), userID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, scopes)
}

// handleRegistryClear implements POST /api/i/registry/clear.
func (s *Server) handleRegistryClear(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	if err := s.registry.Clear(r.Context(), userID, body.Scope); err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// handleRegistryKeysWithType implements POST /api/i/registry/keys-with-type:
// the key list plus the JSON type of each stored value.
//
// Upstream keeps the type next to the value; Cinderfish infers it from the
// stored JSON, which is the same answer unless a client stored a number as a
// string.
func (s *Server) handleRegistryKeysWithType(w http.ResponseWriter, r *http.Request) {
	if !s.requireRegistry(w) {
		return
	}
	var body registryRequest
	userID, ok := s.registryAuth(w, r, &body)
	if !ok {
		return
	}
	all, err := s.registry.All(r.Context(), userID, body.Scope)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make(map[string]string, len(all))
	for key, raw := range all {
		out[key] = registryValueType(raw)
	}
	writeJSON(w, http.StatusOK, out)
}

// registryValueType names the JSON type of a stored registry value.
func registryValueType(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	switch {
	case trimmed == "" || trimmed == "null":
		return "null"
	case trimmed[0] == '"':
		return "string"
	case trimmed[0] == '{':
		return "object"
	case trimmed[0] == '[':
		return "array"
	case trimmed == "true" || trimmed == "false":
		return "boolean"
	default:
		return "number"
	}
}
