package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// handleDriveFilesCreate implements POST /api/drive/files/create: it
// accepts a multipart upload and stores the file.
func (s *Server) handleDriveFilesCreate(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	info, err := s.authenticate(r.Context(), r, nil)
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	if !s.checkRateLimit(w, info.User.ID, limitDriveCreate) {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeAPIError(w, invalidParamError(map[string]any{"reason": "expected a multipart upload"}))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeAPIError(w, invalidParamError(map[string]any{"reason": "no file field"}))
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	name := media.SanitizeName(header.Filename)

	// folderId is optional: an absent or empty field uploads to the root.
	var folderID *string
	if v := r.FormValue("folderId"); v != "" {
		if err := s.checkOwnFolder(r, info.User.ID, v); err != nil {
			writeAPIError(w, err)
			return
		}
		folderID = &v
	}

	stored, err := s.drive.Save(r.Context(), info.User.ID, name, file, contentType, folderID)
	if err != nil {
		if errors.Is(err, media.ErrTooLarge) {
			writeAPIError(w, &APIError{
				Message: "File is too large.", Code: "FILE_TOO_LARGE",
				ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeAPIError(w, &APIError{
			Message: "Could not store the file.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	packed := packDriveFile(stored)
	s.publishDriveFileCreated(info.User.ID, packed)
	writeJSON(w, http.StatusOK, packed)
}

// checkOwnFolder verifies a folder id belongs to the caller before an upload
// or a listing uses it.
func (s *Server) checkOwnFolder(r *http.Request, userID, folderID string) *APIError {
	folder, err := s.folders.GetFolder(r.Context(), folderID)
	if err != nil {
		if errors.Is(err, media.ErrNotFound) {
			return &APIError{
				Message: "No such folder.", Code: "NO_SUCH_FOLDER",
				ID: errIDNoSuchFolder, Kind: "client", Status: http.StatusBadRequest,
			}
		}
		return &APIError{
			Message: "Could not load the folder.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		}
	}
	if folder.UserID != userID {
		return &APIError{
			Message: "The folder is not yours.", Code: "ACCESS_DENIED",
			ID: errIDForbidden, Kind: "permission", Status: http.StatusForbidden,
		}
	}
	return nil
}

// handleDriveFilesList implements POST /api/drive/files.
func (s *Server) handleDriveFilesList(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		Limit    int     `json:"limit"`
		SinceID  string  `json:"sinceId"`
		UntilID  string  `json:"untilId"`
		FolderID *string `json:"folderId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	if body.FolderID != nil {
		if err := s.checkOwnFolder(r, info.User.ID, *body.FolderID); err != nil {
			writeAPIError(w, err)
			return
		}
	}
	files, err := s.drive.List(r.Context(), info.User.ID, body.Limit, body.SinceID, body.UntilID, body.FolderID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not list the files.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusInternalServerError,
		})
		return
	}

	out := make([]any, 0, len(files))
	for _, f := range files {
		out = append(out, packDriveFile(f))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleDriveFilesShow implements POST /api/drive/files/show.
func (s *Server) handleDriveFilesShow(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		FileID string `json:"fileId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	f, err := s.drive.Get(r.Context(), body.FileID)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	writeJSON(w, http.StatusOK, packDriveFile(f))
}

// handleDriveFilesDelete implements POST /api/drive/files/delete.
func (s *Server) handleDriveFilesDelete(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		FileID string `json:"fileId"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}

	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}

	if err := s.drive.Delete(r.Context(), body.FileID, info.User.ID); err != nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{})
}

// handleDriveFilesUpdate implements POST /api/drive/files/update: rename a
// file, edit its caption, toggle the sensitive flag, or move it between
// folders. A null folderId moves the file back to the root.
func (s *Server) handleDriveFilesUpdate(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}

	var body struct {
		FileID      string        `json:"fileId"`
		FolderID    nullableField `json:"folderId"`
		Name        nullableField `json:"name"`
		IsSensitive *bool         `json:"isSensitive"`
		Comment     nullableField `json:"comment"`
	}
	if err := decodeOptionalJSON(r, &body); err != nil {
		writeAPIError(w, err)
		return
	}
	info, err := s.authenticate(r.Context(), r, bodyAsMap(r))
	if err != nil || info == nil {
		writeAuthenticationError(w)
		return
	}
	if body.FileID == "" {
		writeAPIError(w, invalidParamError("fileId is required"))
		return
	}

	patch := media.FileUpdate{
		IsSensitive: body.IsSensitive,
		Name:        body.Name.asUpdatePointer(),
		Comment:     body.Comment.asUpdatePointer(),
	}
	if body.FolderID.provided() {
		folder := body.FolderID.asUpdatePointer()
		if folder == nil || *folder == "" {
			patch.ClearFolder = true
		} else {
			patch.FolderID = folder
		}
	}

	f, err := s.drive.Update(r.Context(), info.User.ID, body.FileID, patch)
	switch {
	case errors.Is(err, media.ErrNotFound):
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	case errors.Is(err, media.ErrNoSuchFolder):
		writeAPIError(w, &APIError{
			Message: "No such folder.", Code: "NO_SUCH_FOLDER",
			ID: errIDInvalidParam, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	case errors.Is(err, media.ErrInvalidName), errors.Is(err, media.ErrInvalidComment):
		writeAPIError(w, invalidParamError("the file name or comment is invalid"))
		return
	case err != nil:
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packDriveFile(f))
}

// handleMediaServe serves locally stored files.
func (s *Server) handleMediaServe(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusNotFound,
		})
		return
	}

	key := r.PathValue("path")
	if key == "" {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}

	rc, err := s.storage.Get(r.Context(), key)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "No such file.", Code: "NO_SUCH_FILE",
			ID: "b6992544-63e7-67f0-fa7f-32444b1b5306", Kind: "client", Status: http.StatusNotFound,
		})
		return
	}
	defer rc.Close()

	// Thumbnails keep their format in the key (".thumb.png"), but original
	// uploads are keyed by content hash with no extension, so their type has
	// to come from the drive row. A file that is not in drive_files (or a
	// server without a database) falls back to sniffing as octet-stream.
	contentType := mediaContentType(key)
	if contentType == "application/octet-stream" && s.pool != nil {
		var stored string
		if err := s.pool.QueryRow(r.Context(),
			`SELECT content_type FROM drive_files WHERE storage_key = $1 LIMIT 1`, key).Scan(&stored); err == nil && stored != "" {
			contentType = stored
		}
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if mediaContentTypeSandboxed(contentType) {
		// SVG, HTML and similar formats execute script in the instance's
		// origin when navigated to directly, and a stored upload is other
		// users' input. Serving them as a download keeps the bytes
		// available without handing them the origin; clients render their
		// own preview through the API instead.
		w.Header().Set("Content-Disposition", "attachment")
		w.Header().Set("Content-Security-Policy", "sandbox")
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
}

// mediaContentTypeSandboxed reports whether a content type can carry
// active content (script, XSLT, plugin navigation) and therefore must not
// be served inline from the instance origin.
func mediaContentTypeSandboxed(contentType string) bool {
	main, _, _ := strings.Cut(contentType, ";")
	main = strings.ToLower(strings.TrimSpace(main))
	switch main {
	case "image/svg+xml", "text/html", "application/xhtml+xml", "text/xml",
		"application/xml", "text/xsl", "application/xslt+xml",
		"application/pdf", "application/x-shockwave-flash":
		return true
	}
	return false
}

// packDriveFile renders a file in Misskey's DriveFile schema.
func packDriveFile(f *media.File) map[string]any {
	return map[string]any{
		"id":           f.ID,
		"createdAt":    f.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		"name":         f.Name,
		"type":         f.ContentType,
		"size":         f.Size,
		"isSensitive":  f.IsSensitive,
		"url":          f.URL,
		"thumbnailUrl": nilIfEmpty(f.ThumbnailURL),
		"webpublicUrl": nil,
		"comment":      nilIfEmpty(f.Comment),
		"folderId":     f.FolderID,
		"blurhash":     nil,
	}
}

// mediaContentType maps a storage key's extension to a content type.
func mediaContentType(key string) string {
	if ext := path.Ext(key); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}
