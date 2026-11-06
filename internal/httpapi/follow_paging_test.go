package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// followIDs returns one side's user ids from a follow-list response.
func followIDs(t *testing.T, items []any, field string) []string {
	t.Helper()
	out := make([]string, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if id, _ := m[field].(string); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// TestFollowListDateCursors covers sinceDate/untilDate on following/list and
// users/following|followers. The bounds filter on the follow row's createdAt,
// independently of the id cursors.
func TestFollowListDateCursors(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolToken, carolID := f.registerUser(t, "carol")
	_, daveID := f.registerUser(t, "dave")

	// alice follows bob, carol and dave; bob's follow is backdated.
	for _, id := range []string{bobID, carolID, daveID} {
		if resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": id}); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow %s = %d %v", id, resp.StatusCode, out)
		}
	}
	backdate := func(follower, followee string) {
		t.Helper()
		if _, err := f.pool.Exec(context.Background(),
			`UPDATE followings SET created_at = now() - interval '2 hours' WHERE follower_id = $1 AND followee_id = $2`,
			follower, followee); err != nil {
			t.Fatalf("backdate: %v", err)
		}
	}
	backdate(aliceID, bobID)

	now := time.Now()
	hourAgo := now.Add(-time.Hour).UnixMilli()
	threeHoursAgo := now.Add(-3 * time.Hour).UnixMilli()

	// sinceDate drops the backdated row; untilDate keeps only it.
	list := postListWithToken(t, f.ts, "/api/following/list", aliceToken, map[string]any{"sinceDate": hourAgo})
	got := followIDs(t, list, "followeeId")
	if len(got) != 2 {
		t.Errorf("sinceDate following = %v, want carol and dave", got)
	}
	for _, id := range got {
		if id == bobID {
			t.Errorf("sinceDate kept the backdated follow: %v", got)
		}
	}

	list = postListWithToken(t, f.ts, "/api/following/list", aliceToken, map[string]any{"untilDate": hourAgo})
	if got = followIDs(t, list, "followeeId"); len(got) != 1 || got[0] != bobID {
		t.Errorf("untilDate following = %v, want just bob", got)
	}

	// A window around the old row keeps exactly it.
	list = postListWithToken(t, f.ts, "/api/following/list", aliceToken, map[string]any{
		"sinceDate": threeHoursAgo, "untilDate": hourAgo,
	})
	if got = followIDs(t, list, "followeeId"); len(got) != 1 || got[0] != bobID {
		t.Errorf("window following = %v, want just bob", got)
	}

	// users/following supports the same bounds.
	list = postListWithToken(t, f.ts, "/api/users/following", "", map[string]any{
		"userId": aliceID, "sinceDate": hourAgo,
	})
	if got = followIDs(t, list, "followeeId"); len(got) != 2 {
		t.Errorf("users/following sinceDate = %v, want two", got)
	}

	// users/followers: bob and carol follow alice, bob's row is backdated.
	for _, pair := range []struct {
		token string
		id    string
	}{{bobToken, bobID}, {carolToken, carolID}} {
		if resp, out := postWithToken(t, f.ts, "/api/following/create", pair.token, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
			t.Fatalf("follow alice = %d %v", resp.StatusCode, out)
		}
	}
	backdate(bobID, aliceID)

	list = postListWithToken(t, f.ts, "/api/users/followers", "", map[string]any{
		"userId": aliceID, "sinceDate": hourAgo,
	})
	if got = followIDs(t, list, "followerId"); len(got) != 1 || got[0] != carolID {
		t.Errorf("users/followers sinceDate = %v, want just carol", got)
	}
	list = postListWithToken(t, f.ts, "/api/users/followers", "", map[string]any{
		"userId": aliceID, "untilDate": hourAgo,
	})
	if got = followIDs(t, list, "followerId"); len(got) != 1 || got[0] != bobID {
		t.Errorf("users/followers untilDate = %v, want just bob", got)
	}
}

// TestFollowListDateBoundary pins the inclusive bounds: sinceDate uses >= and
// untilDate <=, so a row exactly on the millisecond is included.
func TestFollowListDateBoundary(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")
	if resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow = %d %v", resp.StatusCode, out)
	}
	// A fixed millisecond timestamp on the row.
	const at = int64(1_800_000_000_123)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE followings SET created_at = to_timestamp($1 / 1000.0) WHERE follower_id = $2 AND followee_id = $3`,
		at, aliceID, bobID); err != nil {
		t.Fatalf("set timestamp: %v", err)
	}

	includes := func(body map[string]any) bool {
		t.Helper()
		list := postListWithToken(t, f.ts, "/api/following/list", aliceToken, body)
		return len(followIDs(t, list, "followeeId")) == 1
	}
	if !includes(map[string]any{"sinceDate": at}) {
		t.Error("sinceDate at the exact millisecond excluded the row")
	}
	if !includes(map[string]any{"untilDate": at}) {
		t.Error("untilDate at the exact millisecond excluded the row")
	}
	if includes(map[string]any{"sinceDate": at + 1}) {
		t.Error("sinceDate one millisecond later kept the row")
	}
	if includes(map[string]any{"untilDate": at - 1}) {
		t.Error("untilDate one millisecond earlier kept the row")
	}
}
