package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// errIDNoSuchFolder is Misskey's NO_SUCH_FOLDER id
// (packages/backend/src/server/api/endpoints/drive/folders/show.ts).
const errIDNoSuchFolder = "d74ab9eb-bb09-4bba-bf24-fb58f761e1e9"

// packDriveFolder renders a drive_folders row in Misskey's DriveFolder shape.
func packDriveFolder(f *media.Folder) map[string]any {
	return map[string]any{
		"id":           f.ID,
		"createdAt":    f.CreatedAt.UTC().Format(time.RFC3339Nano),
		"name":         f.Name,
		"parentId":     f.ParentID,
		"foldersCount": f.FoldersCount,
		"filesCount":   f.FilesCount,
		// The nested parent is only filled by endpoints that load it; the
		// listing and show answers leave it null, which the schema allows.
		"parent": nil,
	}
}

// requireFolders reports whether the folder service is wired.
func (s *Server) requireFolders(w http.ResponseWriter) bool {
	if s.folders == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return false
	}
	return true
}

func writeNoSuchFolder(w http.ResponseWriter) {
	writeAPIError(w, &APIError{
		Message: "No such folder.", Code: "NO_SUCH_FOLDER",
		ID: errIDNoSuchFolder, Kind: "client", Status: http.StatusNotFound,
	})
}

// handleDriveFolders implements POST /api/drive/folders: the caller's folders
// inside folderId (the root when it is absent or null).
func (s *Server) handleDriveFolders(w http.ResponseWriter, r *http.Request) {
	if !s.requireFolders(w) {
		return
	}

	var body struct {
		Limit     int     `json:"limit"`
		SinceID   string  `json:"sinceId"`
		UntilID   string  `json:"untilId"`
		FolderID  *string `json:"folderId"`
		SinceDate *int64  `json:"sinceDate"`
		UntilDate *int64  `json:"untilDate"`
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
	// A date-only cursor is not part of the store's list signature yet; the
	// id cursors cover the client's pagination.
	_ = body.SinceDate
	_ = body.UntilDate

	if body.FolderID != nil {
		if err := s.checkOwnFolder(r, info.User.ID, *body.FolderID); err != nil {
			writeAPIError(w, err)
			return
		}
	}

	list, err := s.folders.ListFolders(r.Context(), info.User.ID, body.FolderID, body.Limit, body.SinceID, body.UntilID)
	if err != nil {
		writeInternalServerError(w)
		return
	}
	out := make([]any, 0, len(list))
	for _, f := range list {
		out = append(out, packDriveFolder(f))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleDriveFoldersCreate implements POST /api/drive/folders/create.
func (s *Server) handleDriveFoldersCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireFolders(w) {
		return
	}

	var body struct {
		Name     string  `json:"name"`
		ParentID *string `json:"parentId"`
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
	if len(body.Name) > maxFolderNameLength {
		writeAPIError(w, invalidParamError("name is too long"))
		return
	}

	folder, err := s.folders.CreateFolder(r.Context(), info.User.ID, body.Name, body.ParentID)
	if err != nil {
		if errors.Is(err, media.ErrFolderNotFound) {
			writeNoSuchFolder(w)
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packDriveFolder(folder))
}

// handleDriveFoldersShow implements POST /api/drive/folders/show.
func (s *Server) handleDriveFoldersShow(w http.ResponseWriter, r *http.Request) {
	if !s.requireFolders(w) {
		return
	}

	var body struct {
		FolderID string `json:"folderId"`
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
	folder, err := s.folders.GetFolder(r.Context(), body.FolderID)
	if err != nil || folder.UserID != info.User.ID {
		writeNoSuchFolder(w)
		return
	}
	writeJSON(w, http.StatusOK, packDriveFolder(folder))
}

// handleDriveFoldersUpdate implements POST /api/drive/folders/update. name and
// parentId are both optional; an explicit null parentId moves the folder to
// the root, while an absent key leaves the parent alone.
func (s *Server) handleDriveFoldersUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireFolders(w) {
		return
	}

	var body struct {
		FolderID string  `json:"folderId"`
		Name     *string `json:"name"`
		// nullableField keeps "absent" and "explicit null" apart: null moves
		// the folder to the root, absent leaves its parent alone.
		ParentID nullableField `json:"parentId"`
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
	if body.Name != nil && len(*body.Name) > maxFolderNameLength {
		writeAPIError(w, invalidParamError("name is too long"))
		return
	}

	move := body.ParentID.provided()
	var parentID *string
	if move && body.ParentID.value != nil && *body.ParentID.value != "" {
		parentID = body.ParentID.value
	}

	folder, err := s.folders.UpdateFolder(r.Context(), info.User.ID, body.FolderID, body.Name, parentID, move)
	if err != nil {
		switch {
		case errors.Is(err, media.ErrFolderNotFound):
			writeNoSuchFolder(w)
		case errors.Is(err, media.ErrFolderCycle):
			writeAPIError(w, invalidParamError("a folder cannot be moved into itself"))
		default:
			writeInternalServerError(w)
		}
		return
	}
	writeJSON(w, http.StatusOK, packDriveFolder(folder))
}

// handleDriveFoldersDelete implements POST /api/drive/folders/delete. The
// folder's files and subfolders move up to its parent, as upstream.
func (s *Server) handleDriveFoldersDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireFolders(w) {
		return
	}

	var body struct {
		FolderID string `json:"folderId"`
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

	if err := s.folders.DeleteFolder(r.Context(), info.User.ID, body.FolderID); err != nil {
		if errors.Is(err, media.ErrFolderNotFound) {
			writeNoSuchFolder(w)
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
}

// maxFolderNameLength is Misskey's folder name cap.
const maxFolderNameLength = 200
