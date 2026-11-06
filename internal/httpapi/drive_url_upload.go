package httpapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// handleDriveFilesUploadFromURL implements POST /api/drive/files/upload-from-url.
//
// The fetch is deliberately restricted: only http(s), no address that resolves
// to a private/loopback/link-local block, no redirect to such an address, a
// 30-second timeout and the normal per-file size limit. This keeps the endpoint
// from becoming an SSRF probe for the host's internal network.
func (s *Server) handleDriveFilesUploadFromURL(w http.ResponseWriter, r *http.Request) {
	if s.drive == nil {
		writeAPIError(w, &APIError{
			Message: "Not available.", Code: "INTERNAL_ERROR",
			ID: errIDInternal, Kind: "server", Status: http.StatusServiceUnavailable,
		})
		return
	}
	var body struct {
		URL      string        `json:"url"`
		FolderID nullableField `json:"folderId"`
		Name     nullableField `json:"name"`
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

	target, err := url.Parse(strings.TrimSpace(body.URL))
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		writeAPIError(w, invalidParamError("url must be an http(s) URL"))
		return
	}

	var folderID *string
	if folder := body.FolderID.asUpdatePointer(); folder != nil && *folder != "" {
		if apiErr := s.checkOwnFolder(r, info.User.ID, *folder); apiErr != nil {
			writeAPIError(w, apiErr)
			return
		}
		folderID = folder
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return checkFetchTarget(next.Context(), next.URL)
		},
	}
	if err := checkFetchTarget(ctx, target); err != nil {
		writeAPIError(w, invalidParamError("url is not allowed: "+err.Error()))
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		writeAPIError(w, invalidParamError("url is invalid"))
		return
	}
	req.Header.Set("User-Agent", "Cinderfish/0.1 (+drive upload)")
	resp, err := client.Do(req)
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not download the file.", Code: "FETCH_FAILED",
			ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeAPIError(w, &APIError{
			Message: fmt.Sprintf("Remote answered %d.", resp.StatusCode), Code: "FETCH_FAILED",
			ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	limit := s.drive.MaxSize()
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		writeAPIError(w, &APIError{
			Message: "Could not download the file.", Code: "FETCH_FAILED",
			ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}
	if int64(len(data)) > limit {
		writeAPIError(w, &APIError{
			Message: "File is too large.", Code: "FILE_TOO_LARGE",
			ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
		})
		return
	}

	name := ""
	if v := body.Name.asUpdatePointer(); v != nil {
		name = strings.TrimSpace(*v)
	}
	if name == "" {
		name = path.Base(target.Path)
	}
	if name == "" || name == "/" || name == "." {
		name = "file"
	}
	contentType := resp.Header.Get("Content-Type")

	stored, err := s.drive.Save(ctx, info.User.ID, media.SanitizeName(name), bytes.NewReader(data), contentType, folderID)
	if err != nil {
		if errors.Is(err, media.ErrTooLarge) {
			writeAPIError(w, &APIError{
				Message: "File is too large.", Code: "FILE_TOO_LARGE",
				ID: errIDInternal, Kind: "client", Status: http.StatusBadRequest,
			})
			return
		}
		writeInternalServerError(w)
		return
	}
	writeJSON(w, http.StatusOK, packDriveFile(stored))
}

// checkFetchTarget rejects hosts that resolve to loopback, private or
// link-local addresses.
func checkFetchTarget(ctx context.Context, target *url.URL) error {
	host := target.Hostname()
	if host == "" {
		return errors.New("empty host")
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return errors.New("could not resolve the host")
	}
	for _, addr := range ips {
		ip := addr.IP
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return errors.New("the host resolves to a private address")
		}
	}
	return nil
}
