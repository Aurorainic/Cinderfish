package media_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// TestSwitchableStorage verifies the runtime switch used by the admin
// object-storage page: both backends receive writes, and the active one is
// reported through Backend().
func TestSwitchableStorage(t *testing.T) {
	dirA := filepath.Join(t.TempDir(), "a")
	dirB := filepath.Join(t.TempDir(), "b")
	a, err := media.NewLocalStorage(dirA, "/media")
	if err != nil {
		t.Fatalf("local a: %v", err)
	}
	b, err := media.NewLocalStorage(dirB, "/media")
	if err != nil {
		t.Fatalf("local b: %v", err)
	}

	sw := media.NewSwitchableStorage(a)
	ctx := context.Background()

	if _, err := sw.Put(ctx, "first.txt", strings.NewReader("one"), 3, "text/plain"); err != nil {
		t.Fatalf("put on a: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirA, "first.txt")); err != nil {
		t.Errorf("first write did not land in backend a: %v", err)
	}

	sw.Switch(b)
	if _, err := sw.Put(ctx, "second.txt", strings.NewReader("two"), 3, "text/plain"); err != nil {
		t.Fatalf("put on b: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirB, "second.txt")); err != nil {
		t.Errorf("second write did not land in backend b: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirA, "second.txt")); err == nil {
		t.Error("second write must not land in the old backend")
	}

	// Only the active backend serves reads: objects left in the old store
	// are unreachable until cinderfish-media-migrate moves them, which is
	// what the admin page's notice says.
	if _, err := sw.Get(ctx, "first.txt"); err == nil {
		t.Error("objects in the previous backend must not be served after the switch")
	}

	if sw.Backend() != "local" {
		t.Errorf("backend = %q, want local", sw.Backend())
	}
}
