package httpapi

import (
	"net/http"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// validTypePrefix accepts Misskey's drive `type` filter shape: a media type
// such as "image", "image/png" or "image/*". Anything else is rejected instead
// of being passed into LIKE.
func validTypePrefix(t string) bool {
	if t == "" {
		return true
	}
	for _, r := range t {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '/', r == '-', r == '+', r == '*':
		default:
			return false
		}
	}
	return true
}

// handleAdminDriveFiles is POST /api/admin/drive/files: every account's files,
// with Misskey's origin / type / user / hostname filters.
func (s *Server) handleAdminDriveFiles(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.drive == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}

	var body struct {
		Limit    int     `json:"limit"`
		SinceID  string  `json:"sinceId"`
		UntilID  string  `json:"untilId"`
		UserID   *string `json:"userId"`
		Type     string  `json:"type"`
		Origin   string  `json:"origin"`
		Hostname *string `json:"hostname"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if !validTypePrefix(body.Type) {
		writeAPIError(w, invalidParamError("type must look like image, image/png or image/*"))
		return
	}
	switch body.Origin {
	case "", "combined", "local", "remote":
	default:
		writeAPIError(w, invalidParamError("origin must be local, remote or combined"))
		return
	}

	files, err := s.drive.ListAll(r.Context(), media.AdminDriveParams{
		Limit:      body.Limit,
		SinceID:    body.SinceID,
		UntilID:    body.UntilID,
		UserID:     body.UserID,
		TypePrefix: strings.TrimSuffix(body.Type, "*"),
		Origin:     body.Origin,
		Hostname:   body.Hostname,
	})
	if err != nil {
		writeInternalServerError(w)
		return
	}

	ids := make([]string, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.UserID)
	}
	users, err := s.accounts.UsersByIDs(r.Context(), ids)
	if err != nil {
		writeInternalServerError(w)
		return
	}

	out := make([]map[string]any, 0, len(files))
	for _, f := range files {
		packed := packDriveFile(f)
		if u, ok := users[f.UserID]; ok {
			packed["user"] = packUserLite(u, "", nil)
		} else {
			packed["user"] = nil
		}
		out = append(out, packed)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminDriveShowFile is POST /api/admin/drive/show-file: one file with
// its owner, regardless of who uploaded it.
//
// Upstream's version also returns the user's IP and the request that uploaded
// it; Cinderfish stores neither, so the response is the drive file plus the
// owner and nothing else. Admins see the same fields the files list returns.
func (s *Server) handleAdminDriveShowFile(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if s.drive == nil || s.accounts == nil {
		writeInternalServerError(w)
		return
	}
	var body struct {
		FileID string `json:"fileId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	if body.FileID == "" {
		writeAPIError(w, invalidParamError("fileId is required"))
		return
	}

	file, err := s.drive.Get(r.Context(), body.FileID)
	if err != nil || file == nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: errIDNoSuchFile, Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	packed := packDriveFile(file)
	if owner, err := s.accounts.GetByID(r.Context(), file.UserID); err == nil && owner != nil {
		packed["user"] = packUserLite(owner, "", nil)
	} else {
		packed["user"] = nil
	}
	writeJSON(w, http.StatusOK, packed)
}
