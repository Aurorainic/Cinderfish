package httpapi_test

import (
	"net/http"
	"testing"
)

// The clips/* family plus notes/clips and users/clips, per
// packages/backend/src/server/api/endpoints/clips/.

func TestClipsCRUDAndNotes(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	resp, clip := postWithToken(t, f.ts, "/api/clips/create", aliceToken, map[string]any{
		"name":        "funny",
		"description": "good posts only",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, clip)
	}
	clipID, _ := clip["id"].(string)
	if clipID == "" {
		t.Fatalf("create returned no id: %v", clip)
	}
	if clip["name"] != "funny" || clip["isPublic"] != false {
		t.Errorf("created clip = %v, want name funny and isPublic false", clip)
	}
	if clip["description"] != "good posts only" {
		t.Errorf("created description = %v", clip["description"])
	}
	if user, _ := clip["user"].(map[string]any); user == nil {
		t.Errorf("packed clip carries no user: %v", clip)
	}

	// The owner sees their clip in clips/list; the packed owner-only
	// notesCount travels.
	if got := postListWithToken(t, f.ts, "/api/clips/list", aliceToken, nil); len(got) != 1 {
		t.Fatalf("list after create = %d entries, want 1", len(got))
	}

	// A private clip is NO_SUCH_CLIP for everyone else, while anonymous
	// show of a non-existent clip is the same error.
	resp, out := postWithToken(t, f.ts, "/api/clips/show", bobToken, map[string]any{"clipId": clipID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_CLIP" {
		t.Fatalf("foreign show: %d %v", resp.StatusCode, out)
	}

	// Make it public; bob can show it and sees favouriting fields.
	resp, out = postWithToken(t, f.ts, "/api/clips/update", aliceToken, map[string]any{"clipId": clipID, "isPublic": true})
	if resp.StatusCode != http.StatusOK || out["isPublic"] != true {
		t.Fatalf("update isPublic: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/show", bobToken, map[string]any{"clipId": clipID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("public show: %d %v", resp.StatusCode, out)
	}
	if out["isFavorited"] != false || out["favoritedCount"] != float64(0) {
		t.Errorf("viewer fields = %v/%v, want false/0", out["isFavorited"], out["favoritedCount"])
	}
	// The owner also gets notesCount.
	_, out = postWithToken(t, f.ts, "/api/clips/show", aliceToken, map[string]any{"clipId": clipID})
	if out["notesCount"] != float64(0) {
		t.Errorf("owner notesCount = %v, want 0", out["notesCount"])
	}

	// Add a note; duplicates are ALREADY_CLIPPED with the load-bearing id
	// Firefish's menu matches.
	resp, created := postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "to be clipped"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("note create: %d %v", resp.StatusCode, created)
	}
	noteID, _ := created["createdNote"].(map[string]any)["id"].(string)
	if code := postNoContentToken(t, f.ts, "/api/clips/add-note", aliceToken, map[string]any{"clipId": clipID, "noteId": noteID}); code != http.StatusNoContent {
		t.Fatalf("add-note: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/add-note", aliceToken, map[string]any{"clipId": clipID, "noteId": noteID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_CLIPPED" {
		t.Fatalf("duplicate add: %d %v", resp.StatusCode, out)
	}
	if e, ok := out["error"].(map[string]any); !ok || e["id"] != "734806c4-542c-463a-9311-15c512803965" {
		t.Errorf("ALREADY_CLIPPED id = %v, want the verbatim upstream id", out["error"])
	}

	// clips/notes returns the clipped note; search narrows it out when no
	// word matches.
	notes := postListWithToken(t, f.ts, "/api/clips/notes", aliceToken, map[string]any{"clipId": clipID})
	if len(notes) != 1 {
		t.Fatalf("clip notes = %d, want 1", len(notes))
	}
	notes = postListWithToken(t, f.ts, "/api/clips/notes", aliceToken, map[string]any{"clipId": clipID, "search": "matches-nothing"})
	if len(notes) != 0 {
		t.Errorf("clip notes with unmatched search = %d, want 0", len(notes))
	}
	notes = postListWithToken(t, f.ts, "/api/clips/notes", aliceToken, map[string]any{"clipId": clipID, "search": "clipped"})
	if len(notes) != 1 {
		t.Errorf("clip notes with matching search = %d, want 1", len(notes))
	}

	// notes/clips reports the public clip for that note and refuses
	// unknown notes.
	if got := postListWithToken(t, f.ts, "/api/notes/clips", bobToken, map[string]any{"noteId": noteID}); len(got) != 1 {
		t.Errorf("notes/clips = %d entries, want 1", len(got))
	}
	resp, out = post(t, f.ts, "/api/notes/clips", map[string]any{"noteId": "missing0000000000"})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_NOTE" {
		t.Fatalf("notes/clips unknown: %d %v", resp.StatusCode, out)
	}

	// users/clips shows alice's public clips, and clippedCount rose.
	if got := postListWithToken(t, f.ts, "/api/users/clips", bobToken, map[string]any{"userId": aliceID}); len(got) != 1 {
		t.Errorf("users/clips = %d entries, want 1", len(got))
	}
	_, show := postWithToken(t, f.ts, "/api/notes/show", bobToken, map[string]any{"noteId": noteID})
	if show["clippedCount"] != float64(1) {
		t.Errorf("clippedCount = %v, want 1", show["clippedCount"])
	}

	// remove-note twice: the second is NO_SUCH_NOTE.
	if code := postNoContentToken(t, f.ts, "/api/clips/remove-note", aliceToken, map[string]any{"clipId": clipID, "noteId": noteID}); code != http.StatusNoContent {
		t.Fatalf("remove-note: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/remove-note", aliceToken, map[string]any{"clipId": clipID, "noteId": noteID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_NOTE" {
		t.Fatalf("second remove-note: %d %v", resp.StatusCode, out)
	}

	// Favourites round-trip; my-favorites lists them.
	if code := postNoContentToken(t, f.ts, "/api/clips/favorite", bobToken, map[string]any{"clipId": clipID}); code != http.StatusNoContent {
		t.Fatalf("favorite: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/favorite", bobToken, map[string]any{"clipId": clipID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_FAVORITED" {
		t.Fatalf("duplicate favorite: %d %v", resp.StatusCode, out)
	}
	favs := postListWithToken(t, f.ts, "/api/clips/my-favorites", bobToken, nil)
	if len(favs) != 1 {
		t.Fatalf("my-favorites = %d entries, want 1", len(favs))
	}
	_, out = postWithToken(t, f.ts, "/api/clips/show", bobToken, map[string]any{"clipId": clipID})
	if out["isFavorited"] != true || out["favoritedCount"] != float64(1) {
		t.Errorf("after favourite = %v/%v, want true/1", out["isFavorited"], out["favoritedCount"])
	}
	if code := postNoContentToken(t, f.ts, "/api/clips/unfavorite", bobToken, map[string]any{"clipId": clipID}); code != http.StatusNoContent {
		t.Fatalf("unfavorite: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/unfavorite", bobToken, map[string]any{"clipId": clipID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NOT_FAVORITED" {
		t.Fatalf("unfavorite unfavourited: %d %v", resp.StatusCode, out)
	}

	// Update distinguishes an omitted description from an explicit null:
	// omitted leaves it, empty string clears it.
	resp, out = postWithToken(t, f.ts, "/api/clips/update", aliceToken, map[string]any{"clipId": clipID, "name": "renamed"})
	if resp.StatusCode != http.StatusOK || out["description"] != "good posts only" {
		t.Fatalf("update without description = %d %v, want description kept", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/update", aliceToken, map[string]any{"clipId": clipID, "description": ""})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update empty description: %d %v", resp.StatusCode, out)
	}
	if _, present := out["description"]; present && out["description"] != nil {
		t.Errorf("empty description should clear the column, got %v", out["description"])
	}

	// Deleting removes it everywhere.
	if code := postNoContentToken(t, f.ts, "/api/clips/delete", aliceToken, map[string]any{"clipId": clipID}); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/clips/show", aliceToken, map[string]any{"clipId": clipID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_CLIP" {
		t.Fatalf("show after delete: %d %v", resp.StatusCode, out)
	}
	_ = bobID
}

func TestClipLimits(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")

	// Ten clips per user, the upstream role default.
	for i := 0; i < 10; i++ {
		resp, clip := postWithToken(t, f.ts, "/api/clips/create", aliceToken, map[string]any{"name": "clip"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create %d: %d %v", i, resp.StatusCode, clip)
		}
	}
	resp, out := postWithToken(t, f.ts, "/api/clips/create", aliceToken, map[string]any{"name": "one more"})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "TOO_MANY_CLIPS" {
		t.Fatalf("11th create: %d %v", resp.StatusCode, out)
	}
}
