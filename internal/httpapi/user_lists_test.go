package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The users/lists/* family and the notes/user-list-timeline endpoint, per
// packages/backend/src/server/api/endpoints/users/lists/.

func TestUserListCRUDAndTimeline(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	_, carolID := f.registerUser(t, "carol")

	resp, list := postWithToken(t, f.ts, "/api/users/lists/create", aliceToken, map[string]any{"name": "friends"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, list)
	}
	listID, _ := list["id"].(string)
	if listID == "" {
		t.Fatalf("create returned no id: %v", list)
	}
	if list["name"] != "friends" || list["isPublic"] != false {
		t.Errorf("created list = %v, want name friends and isPublic false", list)
	}
	if ids, ok := list["userIds"].([]any); !ok || len(ids) != 0 {
		t.Errorf("new list userIds = %v, want empty", list["userIds"])
	}
	if created, _ := list["createdAt"].(string); created == "" {
		t.Errorf("createdAt missing on %v", list)
	}

	// The owner's list call shows it; another account's shows nothing.
	if got := postListWithToken(t, f.ts, "/api/users/lists/list", aliceToken, nil); len(got) != 1 {
		t.Fatalf("list after create = %d entries, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/users/lists/list", aliceToken, map[string]any{"userId": carolID}); len(got) != 0 {
		t.Errorf("another user's public lists = %d entries, want 0", len(got))
	}

	// Push a member; pushing the same one twice is ALREADY_ADDED.
	if code := postNoContentToken(t, f.ts, "/api/users/lists/push", aliceToken, map[string]any{"listId": listID, "userId": bobID}); code != http.StatusNoContent {
		t.Fatalf("push: %d", code)
	}
	resp, out := postWithToken(t, f.ts, "/api/users/lists/push", aliceToken, map[string]any{"listId": listID, "userId": bobID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_ADDED" {
		t.Fatalf("duplicate push: %d %v", resp.StatusCode, out)
	}

	// The owner can show their private list.
	resp, out = postWithToken(t, f.ts, "/api/users/lists/show", aliceToken, map[string]any{"listId": listID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner show: %d %v", resp.StatusCode, out)
	}
	if ids := listOfIds(out["userIds"]); len(ids) != 1 || ids[0] != bobID {
		t.Errorf("owner show userIds = %v, want [bob]", out["userIds"])
	}
	// get-memberships carries the member packed as UserLite.
	memberships := postListWithToken(t, f.ts, "/api/users/lists/get-memberships", aliceToken, map[string]any{"listId": listID})
	if len(memberships) != 1 {
		t.Fatalf("get-memberships = %v, want one row", memberships)
	}
	first, _ := memberships[0].(map[string]any)
	if first["userId"] != bobID {
		t.Errorf("membership userId = %v, want bob", first["userId"])
	}
	if user, _ := first["user"].(map[string]any); user["id"] != bobID {
		t.Errorf("membership user = %v, want the packed bob", first["user"])
	}

	// Make it public; an anonymous viewer can now show it, and favourite
	// bookkeeping counts bob's favourite.
	resp, _ = postWithToken(t, f.ts, "/api/users/lists/update", aliceToken, map[string]any{"listId": listID, "isPublic": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %d", resp.StatusCode)
	}
	resp, out = post(t, f.ts, "/api/users/lists/show", map[string]any{"listId": listID, "forPublic": true})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("anonymous show: %d %v", resp.StatusCode, out)
	}
	if out["likedCount"] != float64(0) || out["isLiked"] != false {
		t.Errorf("anonymous show counts = %v/%v, want 0/false", out["likedCount"], out["isLiked"])
	}

	favToken, _ := f.registerUser(t, "bob_fav")
	if code := postNoContentToken(t, f.ts, "/api/users/lists/favorite", favToken, map[string]any{"listId": listID}); code != http.StatusNoContent {
		t.Fatalf("favorite: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/users/lists/favorite", favToken, map[string]any{"listId": listID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_FAVORITED" {
		t.Fatalf("duplicate favorite: %d %v", resp.StatusCode, out)
	}
	resp, out = post(t, f.ts, "/api/users/lists/show", map[string]any{"listId": listID, "forPublic": true})
	if out["likedCount"] != float64(1) {
		t.Errorf("likedCount after favourite = %v, want 1", out["likedCount"])
	}
	if code := postNoContentToken(t, f.ts, "/api/users/lists/unfavorite", favToken, map[string]any{"listId": listID}); code != http.StatusNoContent {
		t.Fatalf("unfavorite: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/users/lists/unfavorite", favToken, map[string]any{"listId": listID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_FAVORITED" {
		t.Fatalf("unfavorite unfavourited: %d %v", resp.StatusCode, out)
	}

	// The list timeline shows only members' notes. Carol is not a member;
	// alice owns the list but has not added herself to it, so her note is
	// excluded too.
	resp, created := postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{"text": "alice note"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alice note: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{"text": "bob note"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bob note: %d %v", resp.StatusCode, created)
	}
	resp, created = postWithToken(t, f.ts, "/api/notes/create", tokenOf(f, t, "carol_post"), map[string]any{"text": "carol note"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("carol note: %d %v", resp.StatusCode, created)
	}
	timeline := postListWithToken(t, f.ts, "/api/notes/user-list-timeline", aliceToken, map[string]any{"listId": listID})
	texts := map[string]bool{}
	for _, item := range timeline {
		n, _ := item.(map[string]any)
		text, _ := n["text"].(string)
		texts[text] = true
	}
	if len(texts) != 1 || !texts["bob note"] {
		t.Errorf("list timeline = %v, want only the member's note", texts)
	}

	// Deleting the list clears it everywhere.
	if code := postNoContentToken(t, f.ts, "/api/users/lists/delete", aliceToken, map[string]any{"listId": listID}); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/users/lists/show", aliceToken, map[string]any{"listId": listID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_LIST" {
		t.Fatalf("show after delete: %d %v", resp.StatusCode, out)
	}
}


// errCode digs the Misskey error code out of the envelope.
func errCode(out map[string]any) any {
	if e, ok := out["error"].(map[string]any); ok {
		return e["code"]
	}
	return nil
}


// postNoContent posts a body with a bearer token and requires 204.
func postNoContentToken(t *testing.T, ts *httptest.Server, path, token string, body any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// tokenOf registers username and returns just the token.
func tokenOf(f *fedInstance, t *testing.T, username string) string {
	t.Helper()
	token, _ := f.registerUser(t, username)
	return token
}

// listOfIds coerces a response field into a []string of ids.
func listOfIds(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, _ := item.(string)
		out = append(out, s)
	}
	return out
}

func TestUserListLimitsAndErrors(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	_, otherID := f.registerUser(t, "mallory")

	// Ten lists per user, the upstream role default.
	var firstList string
	for i := 0; i < 10; i++ {
		resp, list := postWithToken(t, f.ts, "/api/users/lists/create", aliceToken, map[string]any{"name": "list"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("create %d: %d %v", i, resp.StatusCode, list)
		}
		if i == 0 {
			firstList, _ = list["id"].(string)
		}
	}
	resp, out := postWithToken(t, f.ts, "/api/users/lists/create", aliceToken, map[string]any{"name": "one more"})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "TOO_MANY_USERLISTS" {
		t.Fatalf("11th create: %d %v", resp.StatusCode, out)
	}

	// A blank name is an invalid parameter.
	resp, _ = postWithToken(t, f.ts, "/api/users/lists/create", aliceToken, map[string]any{})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("blank-name create: %d", resp.StatusCode)
	}

	// Push of an unknown user is NO_SUCH_USER.
	resp, out = postWithToken(t, f.ts, "/api/users/lists/push", aliceToken, map[string]any{"listId": firstList, "userId": "missing0000000000"})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_USER" {
		t.Fatalf("push unknown: %d %v", resp.StatusCode, out)
	}

	// Another account cannot touch alice's list: every path answers
	// NO_SUCH_LIST, not a forbidden that would confirm existence.
	malloryToken := tokenOf(f, t, "mallory2")
	resp, out = postWithToken(t, f.ts, "/api/users/lists/delete", malloryToken, map[string]any{"listId": firstList})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_LIST" {
		t.Fatalf("foreign delete: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/users/lists/push", malloryToken, map[string]any{"listId": firstList, "userId": otherID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_LIST" {
		t.Fatalf("foreign push: %d %v", resp.StatusCode, out)
	}

	// Anonymous calls that need a credential fail the handshake.
	resp, _ = post(t, f.ts, "/api/users/lists/create", map[string]any{"name": "anon"})
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Errorf("anonymous create: %d", resp.StatusCode)
	}

	// Pulling a non-member silently succeeds.
	if code := postNoContentToken(t, f.ts, "/api/users/lists/pull", aliceToken, map[string]any{"listId": firstList, "userId": otherID}); code != http.StatusNoContent {
		t.Errorf("pull non-member: %d", code)
	}
}
