package httpapi_test

import (
	"net/http"
	"testing"
)

// The antennas/* family, per
// packages/backend/src/server/api/endpoints/antennas/.

func TestAntennasCRUDAndCaps(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")

	// Both keyword arrays empty is EMPTY_KEYWORD, upstream's vacuous every().
	resp, out := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "empty", "src": "all",
		"keywords": [][]string{}, "excludeKeywords": [][]string{},
		"users": []string{}, "caseSensitive": false, "withReplies": false, "withFile": false,
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "EMPTY_KEYWORD" {
		t.Fatalf("empty keyword create: %d %v", resp.StatusCode, out)
	}

	resp, antenna := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "watch", "src": "all",
		"keywords":        [][]string{{"hello"}},
		"excludeKeywords": [][]string{},
		"users":           []string{},
		"caseSensitive":   false, "withReplies": false, "withFile": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, antenna)
	}
	antennaID, _ := antenna["id"].(string)
	if antennaID == "" {
		t.Fatalf("create returned no id: %v", antenna)
	}
	if antenna["src"] != "all" || antenna["name"] != "watch" {
		t.Errorf("created antenna = %v", antenna)
	}

	// A five-antenna cap, the upstream role default.
	for i := 0; i < 4; i++ {
		resp, a := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
			"name": "filler", "src": "all",
			"keywords": [][]string{{"kw"}}, "excludeKeywords": [][]string{},
			"users": []string{}, "caseSensitive": false, "withReplies": false, "withFile": false,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("filler %d: %d %v", i, resp.StatusCode, a)
		}
	}
	resp, out = postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "one over", "src": "all",
		"keywords": [][]string{{"kw"}}, "excludeKeywords": [][]string{},
		"users": []string{}, "caseSensitive": false, "withReplies": false, "withFile": false,
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "TOO_MANY_ANTENNAS" {
		t.Fatalf("6th create: %d %v", resp.StatusCode, out)
	}

	// list and show.
	if got := postListWithToken(t, f.ts, "/api/antennas/list", aliceToken, nil); len(got) != 5 {
		t.Errorf("antennas/list = %d, want 5", len(got))
	}
	// Another account's list is empty and their antennas invisible.
	bobsToken := tokenOf(f, t, "bob2")
	if got := postListWithToken(t, f.ts, "/api/antennas/list", bobsToken, nil); len(got) != 0 {
		t.Errorf("bob antennas/list = %d, want 0", len(got))
	}
	resp, out = postWithToken(t, f.ts, "/api/antennas/show", bobsToken, map[string]any{"antennaId": antennaID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_ANTENNA" {
		t.Fatalf("foreign show: %d %v", resp.StatusCode, out)
	}

	// update merges: only withFile changes; keywords stay.
	resp, out = postWithToken(t, f.ts, "/api/antennas/update", aliceToken, map[string]any{
		"antennaId": antennaID, "withFile": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %d %v", resp.StatusCode, out)
	}
	if out["withFile"] != true || out["name"] != "watch" {
		t.Errorf("after update = %v/%v, want withFile true, name kept", out["withFile"], out["name"])
	}
	kws, _ := out["keywords"].([]any)
	if len(kws) != 1 {
		t.Errorf("keywords after update = %v, want kept", out["keywords"])
	}

	// An unknown src is rejected.
	resp, out = postWithToken(t, f.ts, "/api/antennas/update", aliceToken, map[string]any{
		"antennaId": antennaID, "src": "instances",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("instances src: %d %v", resp.StatusCode, out)
	}

	// Delete; then show answers NO_SUCH_ANTENNA.
	if code := postNoContentToken(t, f.ts, "/api/antennas/delete", aliceToken, map[string]any{"antennaId": antennaID}); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/antennas/show", aliceToken, map[string]any{"antennaId": antennaID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_ANTENNA" {
		t.Fatalf("show after delete: %d %v", resp.StatusCode, out)
	}
	_ = bobID
}

func TestAntennaIngestAndTimeline(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	// A keyword antenna of alice's.
	resp, antenna := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "fruit", "src": "all",
		"keywords":        [][]string{{"apple"}, {"pear", "sweet"}},
		"excludeKeywords": [][]string{{"spoiler"}},
		"users":           []string{},
		"caseSensitive":   false, "withReplies": false, "withFile": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, antenna)
	}
	antennaID, _ := antenna["id"].(string)

	// Notes by either author: the antenna collects both.
	resp, created := postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{"text": "an apple a day"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice note: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "very sweet pear"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob note: %d %v", resp.StatusCode, created)
	}
	// Non-matching and excluded notes do not collect.
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "apple spoiler alert"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("spoiler note: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "nothing relevant"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("plain note: %d %v", resp.StatusCode, created)
	}
	// A reply is dropped while withReplies is false.
	resp, created = postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "apple reply", "replyId": created["createdNote"].(map[string]any)["id"],
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reply note: %d %v", resp.StatusCode, created)
	}

	notes := postListWithToken(t, f.ts, "/api/antennas/notes", aliceToken, map[string]any{"antennaId": antennaID})
	if len(notes) != 2 {
		t.Fatalf("antennas/notes = %d, want the two keyword notes", len(notes))
	}

	// src users: only the listed author's notes collect.
	resp, usersAntenna := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "bobby", "src": "users", "users": []string{"@bob"},
		"keywords":        [][]string{{"any"}},
		"excludeKeywords": [][]string{},
		"caseSensitive":   false, "withReplies": true, "withFile": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users antenna: %d %v", resp.StatusCode, usersAntenna)
	}
	usersAntennaID, _ := usersAntenna["id"].(string)
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "any bob note"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob note: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{"text": "any alice note"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice note: %d %v", resp.StatusCode, created)
	}
	notes = postListWithToken(t, f.ts, "/api/antennas/notes", aliceToken, map[string]any{"antennaId": usersAntennaID})
	if len(notes) != 1 {
		t.Fatalf("src users timeline = %d, want only bob's note", len(notes))
	}

	// src list: members of a user list.
	resp, list := postWithToken(t, f.ts, "/api/users/lists/create", aliceToken, map[string]any{"name": "pals"})
	listID, _ := list["id"].(string)
	if code := postNoContentToken(t, f.ts, "/api/users/lists/push", aliceToken, map[string]any{"listId": listID, "userId": bobID}); code != http.StatusNoContent {
		t.Fatalf("push bob: %d", code)
	}
	resp, listAntenna := postWithToken(t, f.ts, "/api/antennas/create", aliceToken, map[string]any{
		"name": "pals feed", "src": "list", "userListId": listID,
		"keywords":        [][]string{{"listnote"}},
		"excludeKeywords": [][]string{},
		"users":           []string{},
		"caseSensitive":   false, "withReplies": true, "withFile": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list antenna: %d %v", resp.StatusCode, listAntenna)
	}
	listAntennaID, _ := listAntenna["id"].(string)
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "listnote from bob"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob listnote: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", tokenOf(f, t, "carol3"), map[string]any{"text": "listnote from carol"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("carol listnote: %d %v", resp.StatusCode, created)
	}
	notes = postListWithToken(t, f.ts, "/api/antennas/notes", aliceToken, map[string]any{"antennaId": listAntennaID})
	if len(notes) != 1 {
		t.Fatalf("src list timeline = %d, want only the member's note", len(notes))
	}

	// remove-note drops one collected note.
	collected := postListWithToken(t, f.ts, "/api/antennas/notes", aliceToken, map[string]any{"antennaId": antennaID})
	first, _ := collected[0].(map[string]any)
	if code := postNoContentToken(t, f.ts, "/api/antennas/remove-note", aliceToken, map[string]any{
		"antennaId": antennaID, "noteId": first["id"],
	}); code != http.StatusNoContent {
		t.Fatalf("remove-note: %d", code)
	}
	notes = postListWithToken(t, f.ts, "/api/antennas/notes", aliceToken, map[string]any{"antennaId": antennaID})
	if len(notes) != 1 {
		t.Errorf("after remove-note = %d, want 1", len(notes))
	}
}
