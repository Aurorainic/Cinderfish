package media

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(dir, "/media")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	url, err := s.Put(context.Background(), "ab/cd/hash", strings.NewReader("hello"), 5, "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if url != "/media/ab/cd/hash" {
		t.Errorf("url = %q", url)
	}

	rc, err := s.Get(context.Background(), "ab/cd/hash")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want hello", data)
	}

	if err := s.Delete(context.Background(), "ab/cd/hash"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(context.Background(), "ab/cd/hash"); err != ErrNotFound {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestLocalStoragePathTraversal(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStorage(dir, "/media")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	// A key that tries to escape the root must stay inside it.
	outside := filepath.Join(filepath.Dir(dir), "escaped.txt")
	if _, err := s.Put(context.Background(), "../escaped.txt", strings.NewReader("x"), 1, "text/plain"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("the file escaped the storage root")
	}
}

func TestLocalStorageBackendName(t *testing.T) {
	s, err := NewLocalStorage(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}
	if s.Backend() != "local" {
		t.Errorf("backend = %q, want local", s.Backend())
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"photo.png":           "photo.png",
		"../../etc/passwd":    "passwd",
		`C:\Windows\evil.exe`: "evil.exe",
		"":                    "file",
		"..":                  "file",
		"normal name.jpg":     "normal name.jpg",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}
