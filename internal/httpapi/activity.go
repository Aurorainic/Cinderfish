package httpapi

import (
	"sync"
	"time"
)

// onlineUserThreshold matches Misskey's USER_ONLINE_THRESHOLD: an account is
// "online" while its last activity is younger than ten minutes.
const onlineUserThreshold = 10 * time.Minute

// activityTracker remembers the last authenticated request of each user.
//
// Misskey keeps a lastActiveDate column and updates it per request; Cinderfish
// keeps the same information in memory instead of paying a write per API call.
// The consequences are documented on get-online-users-count: the count starts
// from zero on restart and covers this process only.
type activityTracker struct {
	mu   sync.Mutex
	seen map[string]time.Time
	now  func() time.Time
}

func newActivityTracker() *activityTracker {
	return &activityTracker{seen: make(map[string]time.Time), now: time.Now}
}

// touch records an authenticated request for userID.
func (a *activityTracker) touch(userID string) {
	if userID == "" {
		return
	}
	a.mu.Lock()
	a.seen[userID] = a.now()
	a.mu.Unlock()
}

// count returns how many users were active within threshold and drops the
// entries that have aged out, so the map does not grow without bound.
func (a *activityTracker) count(threshold time.Duration) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	cutoff := a.now().Add(-threshold)
	n := 0
	for id, at := range a.seen {
		if at.Before(cutoff) {
			delete(a.seen, id)
			continue
		}
		n++
	}
	return n
}
