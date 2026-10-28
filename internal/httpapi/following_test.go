package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
