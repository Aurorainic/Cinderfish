package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimit mirrors one of Misskey's per-endpoint `limit` blocks: a fixed
// window per account. Personal instances rarely hit these, but the caps are
// part of the API contract and one of them was an explicit TODO.
type rateLimit struct {
	bucket   string
	max      int
	duration time.Duration
}

var (
	limitNotesCreate     = rateLimit{bucket: "notes/create", max: 300, duration: time.Hour}
	limitFavoritesCreate = rateLimit{bucket: "notes/favorites/create", max: 20, duration: time.Hour}
	limitDriveCreate     = rateLimit{bucket: "drive/files/create", max: 120, duration: time.Hour}
	limitFollowingCreate = rateLimit{bucket: "following/create", max: 100, duration: time.Hour}
)

type rateWindow struct {
	start time.Time
	count int
}

// rateLimiter is an in-memory fixed-window counter. Misskey keeps the same
// numbers in Redis so every process shares them; a single API process (the
// normal Cinderfish deployment) needs no shared store, and a restart forgiving
// a nearly exhausted window is acceptable for an abuse cap.
type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	now     func() time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{windows: make(map[string]rateWindow), now: time.Now}
}

// allow records one request and reports whether it fits the window. The second
// value is how long until the window resets, for Retry-After.
func (l *rateLimiter) allow(userID string, limit rateLimit) (bool, time.Duration) {
	if l == nil || userID == "" || limit.max <= 0 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	key := userID + "\x00" + limit.bucket
	w, seen := l.windows[key]
	if !seen || now.Sub(w.start) >= limit.duration {
		l.windows[key] = rateWindow{start: now, count: 1}
		l.sweep(now)
		return true, 0
	}
	if w.count >= limit.max {
		return false, w.start.Add(limit.duration).Sub(now)
	}
	w.count++
	l.windows[key] = w
	return true, 0
}

// sweep drops entries whose window ended more than a day ago. It only runs when
// the map is large, so the common path stays a single lookup.
func (l *rateLimiter) sweep(now time.Time) {
	if len(l.windows) < 4096 {
		return
	}
	for key, w := range l.windows {
		if now.Sub(w.start) > 24*time.Hour {
			delete(l.windows, key)
		}
	}
}

// checkRateLimit answers 429 with the Misskey error when the account exceeded
// the endpoint's window, and returns false so the handler stops.
func (s *Server) checkRateLimit(w http.ResponseWriter, userID string, limit rateLimit) bool {
	ok, retry := s.rateLimiter.allow(userID, limit)
	if ok {
		return true
	}
	seconds := int(retry.Round(time.Second) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeAPIError(w, &APIError{
		Message: "Rate limit exceeded. Please try again later.",
		Code:    "RATE_LIMIT_EXCEEDED",
		ID:      errIDRateLimited, Kind: "client", Status: http.StatusTooManyRequests,
	})
	return false
}
