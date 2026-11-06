package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// handleGetSounds implements POST /api/get-sounds.
//
// Misskey reads its `assets/sounds` directory and answers
// `[null, "<folder>/<name>", …]`, where the null is the "none" option. The same
// list is served here from the built frontend's `static-assets/sounds`, so the
// settings picker only offers files the browser can actually play.
func (s *Server) handleGetSounds(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, soundFiles(s.webRoot()))
}

// webRoot is the directory the static frontend is served from.
func (s *Server) webRoot() string {
	if s.web != nil && s.web.root != "" {
		return s.web.root
	}
	if s.cfg != nil {
		return s.cfg.WebDir
	}
	return ""
}

// soundFiles lists "<folder>/<name>" for every mp3 under
// <webRoot>/static-assets/sounds, sorted, with a leading nil.
func soundFiles(webRoot string) []any {
	out := []any{nil}
	if webRoot == "" {
		return out
	}
	root := filepath.Join(webRoot, "static-assets", "sounds")
	dirs, err := os.ReadDir(root)
	if err != nil {
		return out
	}
	names := make([]string, 0, 8)
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, dir.Name()))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := filepath.Ext(entry.Name())
			if !strings.EqualFold(ext, ".mp3") {
				continue
			}
			names = append(names, dir.Name()+"/"+strings.TrimSuffix(entry.Name(), ext))
		}
	}
	sort.Strings(names)
	for _, name := range names {
		out = append(out, name)
	}
	return out
}
