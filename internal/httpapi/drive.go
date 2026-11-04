package httpapi

import (
	"errors"
	"io"
	"net/http"

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

	writeJSON(w, http.StatusOK, packDriveFile(stored))
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

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, rc)
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
		"thumbnailUrl": nil,
		"webpublicUrl": nil,
		"comment":      nil,
		"folderId":     f.FolderID,
		"blurhash":     nil,
	}
}
