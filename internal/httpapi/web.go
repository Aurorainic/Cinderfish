package httpapi

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// staticSite serves the built Vue frontend from disk. Requests that do
// not name a real file fall back to index.html so client-side routes
// work on a hard reload.
type staticSite struct {
	root string
}

func newStaticSite(root string) (*staticSite, error) {
	if root == "" {
		return nil, os.ErrNotExist
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		return nil, err
	}
	return &staticSite{root: root}, nil
}

// handles reports whether a request belongs to the SPA rather than the
// API or ActivityPub surface. Protocol namespaces always go to the mux;
// /@user and /notes/{id} are shared, so the browser only wins when it
// asks for HTML.
func (s *staticSite) handles(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	p := r.URL.Path
	for _, prefix := range []string{
		"/api", "/media", "/.well-known", "/oauth", "/users", "/nodeinfo",
		"/inbox", "/streaming",
	} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return false
		}
	}
	if strings.HasPrefix(p, "/@") || strings.HasPrefix(p, "/notes/") {
		return strings.Contains(r.Header.Get("Accept"), "text/html")
	}
	return true
}

func (s *staticSite) serve(w http.ResponseWriter, r *http.Request) {
	p := filepath.FromSlash(pathClean(r.URL.Path))
	if p != string(filepath.Separator) {
		full := filepath.Join(s.root, p)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.FileServer(http.Dir(s.root)).ServeHTTP(w, r)
			return
		}
	}
	// A missing asset is a real 404: answering index.html would hand the
	// browser HTML where it asked for JavaScript, and a stale index.html that
	// references a pruned chunk would then fail with a MIME error instead of a
	// plain not-found. Client routes still fall through to the shell below.
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		http.NotFound(w, r)
		return
	}
	// Read index.html per request: it is a few hundred bytes, and a
	// rebuild then goes live without restarting the server.
	index, err := os.ReadFile(filepath.Join(s.root, "index.html"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
}

// pathClean normalises a URL path. Unlike path.Clean on an untrusted
// path it first rejects backslashes, which are separators on Windows.
func pathClean(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	parts := make([]string, 0, 8)
	for _, seg := range strings.Split(p, "/") {
		switch seg {
		case "", ".":
			continue
		case "..":
			if len(parts) > 0 {
				parts = parts[:len(parts)-1]
			}
		default:
			parts = append(parts, seg)
		}
	}
	return "/" + strings.Join(parts, "/")
}
