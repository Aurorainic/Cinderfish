package httpapi

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"milnfish/internal/federation"
)

// ApplyFederationConfig seeds the runtime federation state from the
// configuration: the mode row in instance_meta and the allowlist hosts.
//
// The configuration is a seed, not the source of truth. Once an
// administrator changes either through the API, the database wins and a
// restart does not undo it.
func (s *Server) ApplyFederationConfig(ctx context.Context) error {
	if s.federationPolicy == nil || s.pool == nil {
		return nil
	}

	// Seed the mode only when there is no row yet, so a runtime switch
	// survives a restart.
	mode := s.cfg.FederationMode
	if mode == "" {
		mode = "none"
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb($1::text))
ON CONFLICT (key) DO NOTHING
	`, mode); err != nil {
		return fmt.Errorf("federation: seed mode: %w", err)
	}

	for _, host := range s.cfg.Federation.AllowlistHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" {
			continue
		}
		if err := s.federationPolicy.AddAllowlistHost(ctx, host); err != nil {
			return fmt.Errorf("federation: seed allowlist %q: %w", host, err)
		}
	}
	return nil
}

// handleAdminFederationMode is POST /api/admin/federation/mode. It
// switches the mode at runtime; the change applies to the next request.
func (s *Server) handleAdminFederationMode(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		Mode string `json:"mode"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if _, err := federation.ParseMode(body.Mode); err != nil {
		writeAPIError(w, invalidParamError("mode must be public, allowlist or none"))
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb($1::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, body.Mode); err != nil {
		writeInternalServerError(w)
		return
	}
	log.Printf("httpapi: federation mode switched to %s", body.Mode)
	writeJSON(w, http.StatusOK, map[string]any{"mode": body.Mode})
}

// handleAdminAllowlist lists the allowlist, adds a host, or removes one,
// depending on the method and body.
func (s *Server) handleAdminAllowlist(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"hosts": s.federationPolicy.Allowlist(r.Context())})
		return
	}

	var body struct {
		Host   string `json:"host"`
		Remove bool   `json:"remove"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if strings.TrimSpace(body.Host) == "" {
		writeAPIError(w, invalidParamError("host is required"))
		return
	}
	var err error
	if body.Remove {
		err = s.federationPolicy.RemoveAllowlistHost(r.Context(), body.Host)
	} else {
		err = s.federationPolicy.AddAllowlistHost(r.Context(), body.Host)
	}
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"hosts": s.federationPolicy.Allowlist(r.Context())})
}
