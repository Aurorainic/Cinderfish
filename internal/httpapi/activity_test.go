package httpapi

import (
	"testing"
	"time"
)

// TestActivityTrackerWindow checks the in-memory online window: a touched user
// counts, an aged-out entry is dropped, and the map does not keep growing.
func TestActivityTrackerWindow(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	tracker := newActivityTracker()
	tracker.now = func() time.Time { return now }

	if got := tracker.count(onlineUserThreshold); got != 0 {
		t.Fatalf("empty tracker count = %d, want 0", got)
	}

	tracker.touch("alice")
	tracker.touch("bob")
	tracker.touch("")
	if got := tracker.count(onlineUserThreshold); got != 2 {
		t.Fatalf("count = %d, want 2", got)
	}

	// Nine minutes later both are still inside the ten-minute window.
	now = now.Add(9 * time.Minute)
	if got := tracker.count(onlineUserThreshold); got != 2 {
		t.Fatalf("count after 9m = %d, want 2", got)
	}

	// Eleven minutes later both have aged out and the entries are removed.
	now = now.Add(2 * time.Minute)
	if got := tracker.count(onlineUserThreshold); got != 0 {
		t.Fatalf("count after 11m = %d, want 0", got)
	}
	if len(tracker.seen) != 0 {
		t.Errorf("stale entries left behind: %d", len(tracker.seen))
	}

	// A fresh touch is counted again.
	tracker.touch("alice")
	if got := tracker.count(onlineUserThreshold); got != 1 {
		t.Fatalf("count after re-touch = %d, want 1", got)
	}
}
