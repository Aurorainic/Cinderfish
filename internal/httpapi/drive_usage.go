package httpapi

import "net/http"

// handleDrive implements POST /api/drive: the caller's quota and usage.
//
// Cinderfish has no instance-wide drive quota, so `capacity` is 0 and clients
// should read it as "unlimited". `filesCount` is an extra field (Misskey's
// DriveInfo carries only capacity/usage) used by the settings page.
func (s *Server) handleDrive(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body map[string]any
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, body)
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	usage, files, err := s.drive.Usage(r.Context(), info.User.ID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"capacity":   0,
		"usage":      usage,
		"filesCount": files,
	})
}
