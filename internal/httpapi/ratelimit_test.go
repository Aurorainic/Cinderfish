package httpapi

import (
	"testing"
	"time"
)

func TestRateLimiterWindow(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	limiter := newRateLimiter()
	limiter.now = func() time.Time { return now }
	limit := rateLimit{bucket: "test", max: 2, duration: time.Hour}

	for i := 0; i < 2; i++ {
		if ok, _ := limiter.allow("alice", limit); !ok {
			t.Fatalf("call %d refused inside the window", i+1)
		}
	}
	ok, retry := limiter.allow("alice", limit)
	if ok {
		t.Fatal("third call accepted, want a refusal")
	}
	if retry <= 0 || retry > time.Hour {
		t.Errorf("retry = %v, want (0, 1h]", retry)
	}

	// Another account has its own window.
	if ok, _ := limiter.allow("bob", limit); !ok {
		t.Error("bob refused because alice was limited")
	}
	// A different endpoint has its own window.
	if ok, _ := limiter.allow("alice", rateLimit{bucket: "other", max: 2, duration: time.Hour}); !ok {
		t.Error("alice refused on a different bucket")
	}

	// The window resets after its duration.
	now = now.Add(time.Hour)
	if ok, _ := limiter.allow("alice", limit); !ok {
		t.Error("call refused after the window reset")
	}
}
