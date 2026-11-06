package httpapi_test

import (
	"net/http"
	"testing"
)

// TestUsersStats pins the seven activity counters of users/stats against the
// notes the user actually wrote: plain notes, replies, renotes, and the notes
// of other people they answered or renoted.
func TestUsersStats(t *testing.T) {
	ts := newTestServer(t, "open")
	aliceID, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	bobID, bobToken := createUser(t, ts, uniqueName(t)+"b")

	bobNote := createNote(t, ts, bobToken, map[string]any{"text": "bob's note"})
	createNote(t, ts, aliceToken, map[string]any{"text": "a plain note"})
	createNote(t, ts, aliceToken, map[string]any{"text": "a reply", "replyId": bobNote})
	createNote(t, ts, aliceToken, map[string]any{"text": "", "renoteId": bobNote})

	// Anonymous call by userId, as upstream.
	resp, out := post(t, ts, "/api/users/stats", map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/stats = %d %v", resp.StatusCode, out)
	}
	want := map[string]float64{
		"notesCount":   1,
		"repliesCount": 1,
		"renotesCount": 1,
		// Alice replied to one note of Bob's and renoted one.
		"repliedCount": 0,
		"renotedCount": 0,
	}
	for field, value := range want {
		if got, _ := out[field].(float64); got != value {
			t.Errorf("%s = %v, want %v (body %v)", field, out[field], value, out)
		}
	}
	if _, ok := out["pollVotesCount"]; !ok {
		t.Errorf("pollVotesCount missing from %v", out)
	}
	if _, ok := out["pollVotedCount"]; !ok {
		t.Errorf("pollVotedCount missing from %v", out)
	}

	// Bob's side of the same notes: one of his notes was replied to and one
	// was renoted.
	bobResp, bobBody := post(t, ts, "/api/users/stats", map[string]any{"userId": bobID})
	if bobResp.StatusCode != http.StatusOK {
		t.Fatalf("bob stats = %d %v", bobResp.StatusCode, bobBody)
	}
	if got, _ := bobBody["repliedCount"].(float64); got != 1 {
		t.Errorf("bob repliedCount = %v, want 1", bobBody["repliedCount"])
	}
	if got, _ := bobBody["renotedCount"].(float64); got != 1 {
		t.Errorf("bob renotedCount = %v, want 1", bobBody["renotedCount"])
	}

	// An empty userId answers for the caller, which the account-info page
	// uses for its own profile.
	_, self := post(t, ts, "/api/users/stats", map[string]any{"i": aliceToken})
	if got, _ := self["notesCount"].(float64); got != 1 {
		t.Errorf("self stats notesCount = %v, want 1", self["notesCount"])
	}

	// An empty userId without a credential cannot resolve anyone.
	if resp, _ := post(t, ts, "/api/users/stats", map[string]any{}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("anonymous empty userId = %d, want 400", resp.StatusCode)
	}

	// Unknown ids answer NO_SUCH_USER.
	resp, missing := post(t, ts, "/api/users/stats", map[string]any{"userId": "000000000000000000000000"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing user = %d %v, want 404", resp.StatusCode, missing)
	}
	if e, _ := missing["error"].(map[string]any); e["code"] != "NO_SUCH_USER" {
		t.Errorf("missing user code = %v, want NO_SUCH_USER", e["code"])
	}
}

// TestHashtagsSearch pins the composer's tag suggestions: an empty query
// ranks the most used tags, a query matches as a substring, and limit/offset
// bound the answer.
func TestHashtagsSearch(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	createNote(t, ts, aliceToken, map[string]any{"text": "#go first"})
	createNote(t, ts, bobToken, map[string]any{"text": "#go again"})
	createNote(t, ts, bobToken, map[string]any{"text": "#golang weekly"})
	createNote(t, ts, aliceToken, map[string]any{"text": "#cats purr"})

	tags := func(body map[string]any) []any {
		t.Helper()
		resp, raw := postRaw(t, ts, "/api/hashtags/search", body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("hashtags/search %v = %d %v", body, resp.StatusCode, raw)
		}
		list, _ := raw.([]any)
		return list
	}

	all := tags(map[string]any{})
	if len(all) != 3 {
		t.Fatalf("tags = %v, want three", all)
	}
	// Most used first; the one-use tags tie-break alphabetically.
	if all[0] != "go" || all[1] != "cats" || all[2] != "golang" {
		t.Errorf("tags = %v, want [go cats golang]", all)
	}

	matched := tags(map[string]any{"query": "go"})
	if len(matched) != 2 || matched[0] != "go" || matched[1] != "golang" {
		t.Errorf("query go = %v, want [go golang]", matched)
	}

	if limited := tags(map[string]any{"limit": 1}); len(limited) != 1 || limited[0] != "go" {
		t.Errorf("limit 1 = %v, want [go]", limited)
	}
	if skipped := tags(map[string]any{"offset": 1}); len(skipped) != 2 || skipped[0] != "cats" {
		t.Errorf("offset 1 = %v, want [cats golang]", skipped)
	}
}

// TestNotesFeatured pins the public ranking: engagement decides the order,
// private notes and renotes stay out, and limit is honoured.
func TestNotesFeatured(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")
	_, carolToken := createUser(t, ts, uniqueName(t)+"c")

	quiet := createNote(t, ts, aliceToken, map[string]any{"text": "quiet note"})
	popular := createNote(t, ts, bobToken, map[string]any{"text": "popular note"})

	for _, token := range []string{aliceToken, carolToken} {
		if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
			"i": token, "noteId": popular, "reaction": "👍",
		}); resp.StatusCode != http.StatusOK {
			t.Fatalf("react: %d %v", resp.StatusCode, out)
		}
	}
	// A reply counts as engagement for the note being answered; it is
	// home-only so it does not enter the public featured list itself.
	createNote(t, ts, carolToken, map[string]any{"text": "answering", "replyId": popular, "visibility": "home"})

	// Notes outside public visibility must not surface even with reactions
	// on them.
	private := createNote(t, ts, carolToken, map[string]any{"text": "private", "visibility": "followers"})
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": carolToken, "noteId": private, "reaction": "⭐",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("react private: %d %v", resp.StatusCode, out)
	}

	resp, raw := postRaw(t, ts, "/api/notes/featured", map[string]any{"limit": 10})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/featured = %d %v", resp.StatusCode, raw)
	}
	list, _ := raw.([]any)
	if len(list) != 2 {
		t.Fatalf("featured = %v, want two public notes", list)
	}
	first, _ := list[0].(map[string]any)
	if first["id"] != popular {
		t.Errorf("first featured = %v, want the engaged note %s", first["id"], popular)
	}
	second, _ := list[1].(map[string]any)
	if second["id"] != quiet {
		t.Errorf("second featured = %v, want %s", second["id"], quiet)
	}
	for _, item := range list {
		if n, _ := item.(map[string]any); n["id"] == private {
			t.Errorf("private note %s appeared in featured", private)
		}
	}

	// The limit bounds the answer.
	if resp, raw := postRaw(t, ts, "/api/notes/featured", map[string]any{"limit": 1}); resp.StatusCode != http.StatusOK {
		t.Fatalf("featured limit 1 = %d %v", resp.StatusCode, raw)
	} else if limited, _ := raw.([]any); len(limited) != 1 {
		t.Errorf("featured limit 1 = %d notes, want 1", len(limited))
	}

	// sinceId excludes notes at or before the anchor.
	resp, raw = postRaw(t, ts, "/api/notes/featured", map[string]any{"sinceId": quiet})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("featured sinceId = %d %v", resp.StatusCode, raw)
	}
	if after, _ := raw.([]any); len(after) != 1 {
		t.Errorf("sinceId quiet = %d notes, want only the newer one", len(after))
	}
}
