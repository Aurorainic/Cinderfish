package httpapi

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSoundFiles(t *testing.T) {
	if got := soundFiles(""); !reflect.DeepEqual(got, []any{nil}) {
		t.Fatalf("empty root = %v, want [nil]", got)
	}
	if got := soundFiles(t.TempDir()); !reflect.DeepEqual(got, []any{nil}) {
		t.Fatalf("missing directory = %v, want [nil]", got)
	}

	root := t.TempDir()
	mk := func(rel string) {
		t.Helper()
		path := filepath.Join(root, "static-assets", "sounds", rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("mp3"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	mk("syuilo/up.mp3")
	mk("syuilo/pope2.mp3")
	mk("other/voice.mp3")
	mk("syuilo/readme.txt")
	if err := os.MkdirAll(filepath.Join(root, "static-assets", "sounds", "empty"), 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}

	want := []any{nil, "other/voice", "syuilo/pope2", "syuilo/up"}
	if got := soundFiles(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("soundFiles = %v, want %v", got, want)
	}
}
