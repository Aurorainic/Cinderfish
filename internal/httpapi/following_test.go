package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// postListWithToken posts a JSON body with a bearer token and decodes an
// array response.
func postListWithToken(t *testing.T, ts *httptest.Server, path, token string, body any) []any {
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s status = %d", path, resp.StatusCode)
	}
	var out []any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

// postWithToken posts a JSON body with a bearer token.
func postWithToken(t *testing.T, ts *httptest.Server, path, token string, body any) (*http.Response, map[string]any) {
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
	t.Cleanup(func() { resp.Body.Close() })

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return resp, decoded
}

// listOf returns a decoded response as a list, tolerating an object with
// an items key.
func listOf(v map[string]any) []any {
	if items, ok := v["items"].([]any); ok {
		return items
	}
	if list, ok := v["list"].([]any); ok {
		return list
	}
	return nil
}

func TestFollowingCreateAndDeleteLocal(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")

	resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: %d %v", resp.StatusCode, out)
	}
	if out["id"] != bobID {
		t.Errorf("follow returned id %v, want the followee %s", out["id"], bobID)
	}

	// The counters follow the relationship, in both directions.
	if resp, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show bob: %d %v", resp.StatusCode, show)
	} else if show["followersCount"] != float64(1) {
		t.Errorf("followersCount after follow = %v, want 1", show["followersCount"])
	}

	// Following twice is an error with Misskey's id and message.
	resp, out = postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("double follow status = %d, want 400", resp.StatusCode)
	}
	if e, ok := out["error"].(map[string]any); !ok || e["code"] != "ALREADY_FOLLOWING" {
		t.Errorf("double follow error = %v", out["error"])
	}

	resp, out = postWithToken(t, f.ts, "/api/following/delete", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unfollow: %d %v", resp.StatusCode, out)
	}
	if resp, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show bob after unfollow: %d %v", resp.StatusCode, show)
	} else if show["followersCount"] != float64(0) {
		t.Errorf("followersCount after unfollow = %v, want 0", show["followersCount"])
	}

	// Unfollowing what you do not follow is an error too.
	resp, out = postWithToken(t, f.ts, "/api/following/delete", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("double unfollow status = %d, want 400", resp.StatusCode)
	}
	if e, ok := out["error"].(map[string]any); !ok || e["code"] != "NOT_FOLLOWING" {
		t.Errorf("double unfollow error = %v", out["error"])
	}
}

func TestFollowingYourselfIsRejected(t *testing.T) {
	f := newFederationServer(t, "public")
	token, id := f.registerUser(t, "alice")

	resp, out := postWithToken(t, f.ts, "/api/following/create", token, map[string]any{"userId": id})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "FOLLOWEE_IS_YOURSELF" {
		t.Errorf("code = %v", e["code"])
	}
}

func TestFollowingUnknownUser(t *testing.T) {
	f := newFederationServer(t, "public")
	token, _ := f.registerUser(t, "alice")

	resp, out := postWithToken(t, f.ts, "/api/following/create", token, map[string]any{"userId": "00000000000000000000"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "NO_SUCH_USER" {
		t.Errorf("code = %v", e["code"])
	}
}

func TestMentionCreatesNotification(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")

	resp, out := postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "hello @bob, look at this",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create note: %d %v", resp.StatusCode, out)
	}

	// bob now has a mention notification, which only happens when the
	// note text was parsed for @-references.
	list := postListWithToken(t, f.ts, "/api/notifications", bobToken, map[string]any{})
	found := false
	for _, item := range list {
		n, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if n["type"] == "mention" {
			found = true
		}
	}
	if !found {
		t.Fatalf("no mention notification for %s in %v", bobID, list)
	}
}

// followListItems posts to users/followers or users/following and returns
// the decoded Following objects.
func followListItems(t *testing.T, ts *httptest.Server, path, token string, body map[string]any) []map[string]any {
	t.Helper()
	payload := map[string]any{}
	for k, v := range body {
		payload[k] = v
	}
	if token != "" {
		payload["i"] = token
	}
	resp, raw := postRaw(t, ts, path, payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: status %d, body %v", path, resp.StatusCode, raw)
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("%s did not return an array: %v", path, raw)
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("%s returned a non-object item: %v", path, item)
		}
		out = append(out, m)
	}
	return out
}

// followItemIDs returns the follow row ids of a page.
func followItemIDs(items []map[string]any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		id, _ := item["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// TestUsersFollowListsPaginate pins the users/followers and
// users/following contract: Misskey's Following envelope with the far
// user packed inside, paged by the follow row id the way
// makePaginationQuery does.
func TestUsersFollowListsPaginate(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")

	others := make([]string, 0, 4)
	for _, name := range []string{"bob", "carol", "dave", "erin"} {
		token, id := f.registerUser(t, name)
		if resp, out := postWithToken(t, f.ts, "/api/following/create", aliceToken, map[string]any{"userId": id}); resp.StatusCode != http.StatusOK {
			t.Fatalf("alice follows %s: %d %v", name, resp.StatusCode, out)
		}
		if resp, out := postWithToken(t, f.ts, "/api/following/create", token, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
			t.Fatalf("%s follows alice: %d %v", name, resp.StatusCode, out)
		}
		others = append(others, id)
	}

	cases := []struct {
		name string
		path string
		// edge is the Following field carrying the far user; targetID is
		// the field that must name the list's owner.
		edge     string
		targetID string
	}{
		{"followers", "/api/users/followers", "follower", "followeeId"},
		{"following", "/api/users/following", "followee", "followerId"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			all := followListItems(t, f.ts, tc.path, aliceToken, map[string]any{"userId": aliceID})
			if len(all) != len(others) {
				t.Fatalf("%s returned %d rows, want %d", tc.path, len(all), len(others))
			}
			seen := map[string]bool{}
			for _, item := range all {
				if item["id"] == "" || item["createdAt"] == nil {
					t.Errorf("%s row misses the follow id or createdAt: %v", tc.path, item)
				}
				if item["followerId"] == nil || item["followeeId"] == nil {
					t.Errorf("%s row misses followerId or followeeId: %v", tc.path, item)
				}
				if item[tc.targetID] != aliceID {
					t.Errorf("%s row %s = %v, want %s", tc.path, tc.targetID, item[tc.targetID], aliceID)
				}
				user, ok := item[tc.edge].(map[string]any)
				if !ok {
					t.Fatalf("%s row has no %s user: %v", tc.path, tc.edge, item)
				}
				id, _ := user["id"].(string)
				if id == "" || user["username"] == nil {
					t.Errorf("%s embedded user is not a UserLite: %v", tc.path, user)
				}
				seen[id] = true
			}
			for _, id := range others {
				if !seen[id] {
					t.Errorf("%s is missing %s", tc.path, id)
				}
			}

			// userId defaults to the caller.
			if got := followListItems(t, f.ts, tc.path, aliceToken, map[string]any{}); len(got) != len(all) {
				t.Errorf("%s without userId returned %d rows, want the caller's %d", tc.path, len(got), len(all))
			}

			// limit keeps the newest follows.
			page := followListItems(t, f.ts, tc.path, aliceToken, map[string]any{"userId": aliceID, "limit": 2})
			want := followItemIDs(all[:2])
			if got := followItemIDs(page); !reflect.DeepEqual(got, want) {
				t.Errorf("%s limit 2 = %v, want %v", tc.path, got, want)
			}

			// untilId pages downward from a cursor.
			older := followListItems(t, f.ts, tc.path, aliceToken, map[string]any{"userId": aliceID, "untilId": all[0]["id"]})
			want = followItemIDs(all[1:])
			if got := followItemIDs(older); !reflect.DeepEqual(got, want) {
				t.Errorf("%s untilId page = %v, want %v", tc.path, got, want)
			}

			// sinceId without untilId pages upward: the oldest rows after
			// the cursor, in ascending order (makePaginationQuery).
			oldest, _ := all[len(all)-1]["id"].(string)
			newer := followListItems(t, f.ts, tc.path, aliceToken, map[string]any{"userId": aliceID, "sinceId": oldest, "limit": 2})
			second, _ := all[len(all)-2]["id"].(string)
			third, _ := all[len(all)-3]["id"].(string)
			if got, want := followItemIDs(newer), []string{second, third}; !reflect.DeepEqual(got, want) {
				t.Errorf("%s sinceId page = %v, want %v (ascending)", tc.path, got, want)
			}
		})
	}
	// Both endpoints are readable without a credential, like their
	// neighbours users/show and users/notes.
	for _, path := range []string{"/api/users/followers", "/api/users/following"} {
		if got := followListItems(t, f.ts, path, "", map[string]any{"userId": aliceID}); len(got) != len(others) {
			t.Errorf("%s as anonymous returned %d rows, want %d", path, len(got), len(others))
		}
	}
}

// TestUsersFollowListPrivacy pins the visibility gate: a private list is
// the owner's alone, a followers-only list opens to the owner and the
// owner's followers, and an administrator reads both.
func TestUsersFollowListPrivacy(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobAccountID := f.registerUser(t, "bob")

	if _, err := f.pool.Exec(context.Background(), `
UPDATE user_profiles SET followers_visibility = 'private', following_visibility = 'followers'
WHERE user_id = $1
	`, aliceID); err != nil {
		t.Fatalf("set visibility: %v", err)
	}

	// Anonymous callers are refused on both lists.
	for _, path := range []string{"/api/users/followers", "/api/users/following"} {
		resp, out := post(t, f.ts, path, map[string]any{"userId": aliceID})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s as anonymous: status %d, want 403", path, resp.StatusCode)
		}
		e, _ := out["error"].(map[string]any)
		if e["code"] != "FORBIDDEN" {
			t.Errorf("%s as anonymous: code = %v, want FORBIDDEN", path, e["code"])
		}
	}

	// The owner reads both.
	followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID})
	followListItems(t, f.ts, "/api/users/following", aliceToken, map[string]any{"userId": aliceID})

	// bob does not follow alice yet, so the followers-only list stays shut
	// and the private one never opens.
	resp, _ := postWithToken(t, f.ts, "/api/users/following", bobToken, map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("followers-only list as outsider: status %d, want 403", resp.StatusCode)
	}
	resp, _ = postWithToken(t, f.ts, "/api/users/followers", bobToken, map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("private list as outsider: status %d, want 403", resp.StatusCode)
	}

	// Following alice opens her followers-only list, nothing else.
	if resp, out := postWithToken(t, f.ts, "/api/following/create", bobToken, map[string]any{"userId": aliceID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("bob follows alice: %d %v", resp.StatusCode, out)
	}
	followListItems(t, f.ts, "/api/users/following", bobToken, map[string]any{"userId": aliceID})
	resp, _ = postWithToken(t, f.ts, "/api/users/followers", bobToken, map[string]any{"userId": aliceID})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("private list as follower: status %d, want 403", resp.StatusCode)
	}

	// An administrator reads both, as upstream's isModerator does.
	if _, err := f.pool.Exec(context.Background(), `UPDATE users SET is_admin = TRUE WHERE id = $1`, bobAccountID); err != nil {
		t.Fatalf("make bob an admin: %v", err)
	}
	followListItems(t, f.ts, "/api/users/followers", bobToken, map[string]any{"userId": aliceID})
}

// TestUsersFollowListNeedsTargetWithoutCredential pins the userId default:
// it names the caller, so an anonymous request without one is invalid.
func TestUsersFollowListNeedsTargetWithoutCredential(t *testing.T) {
	f := newFederationServer(t, "public")
	for _, path := range []string{"/api/users/followers", "/api/users/following"} {
		resp, out := post(t, f.ts, path, map[string]any{})
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", path, resp.StatusCode)
		}
		e, _ := out["error"].(map[string]any)
		if e["code"] != "INVALID_PARAM" {
			t.Errorf("%s: code = %v, want INVALID_PARAM", path, e["code"])
		}
	}
}

// TestUsersFollowListUnknownUser pins NO_SUCH_USER with each endpoint's
// own Misskey error id.
func TestUsersFollowListUnknownUser(t *testing.T) {
	f := newFederationServer(t, "public")
	cases := []struct {
		path string
		id   string
	}{
		{"/api/users/followers", "27fa5435-88ab-43de-9360-387de88727cd"},
		{"/api/users/following", "63e4aba4-4156-4e53-be25-c9559e42d71b"},
	}
	for _, tc := range cases {
		resp, out := post(t, f.ts, tc.path, map[string]any{"userId": "00000000000000000000"})
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d, want 404", tc.path, resp.StatusCode)
		}
		e, _ := out["error"].(map[string]any)
		if e["code"] != "NO_SUCH_USER" {
			t.Errorf("%s: code = %v, want NO_SUCH_USER", tc.path, e["code"])
		}
		if e["id"] != tc.id {
			t.Errorf("%s: error id = %v, want %s", tc.path, e["id"], tc.id)
		}
	}
}
