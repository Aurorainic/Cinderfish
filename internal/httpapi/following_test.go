package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
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

// insertFollowRequest records a pending follow request directly. The API
// cannot create one yet: a local follow is immediate and a remote follow
// is only queued by the inbox, so tests arrange the row the inbox would
// leave behind. uri is the incoming Follow activity id, empty for a
// request without one.
func insertFollowRequest(t *testing.T, f *fedInstance, followerID, followeeID, uri string) string {
	t.Helper()
	id := idgen.New()
	var uriArg any
	if uri != "" {
		uriArg = uri
	}
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO follow_requests (id, follower_id, followee_id, uri) VALUES ($1,$2,$3,$4)
	`, id, followerID, followeeID, uriArg); err != nil {
		t.Fatalf("insert follow request: %v", err)
	}
	return id
}

// insertRemoteUser stores a remote actor row the way an inbound activity
// leaves it, and returns its id.
func insertRemoteUser(t *testing.T, f *fedInstance, host, username string) string {
	t.Helper()
	id := idgen.New()
	if _, err := f.pool.Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, host, uri, inbox)
VALUES ($1,$2,$3,$4,$5,$6)
	`, id, username, strings.ToLower(username), host,
		"https://"+host+"/users/"+username, "https://"+host+"/inbox"); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}
	return id
}

// queuedActivity is one row of the outbound delivery queue.
type queuedActivity struct {
	Type       string
	Actor      string
	ActivityID string
	ObjectID   string
	Inbox      string
}

// deliverQueue returns the activities queued for delivery, oldest first.
func deliverQueue(t *testing.T, f *fedInstance) []queuedActivity {
	t.Helper()
	rows, err := f.pool.Query(context.Background(), `
SELECT payload, inbox_url FROM deliver_queue ORDER BY id
	`)
	if err != nil {
		t.Fatalf("read deliver queue: %v", err)
	}
	defer rows.Close()
	var out []queuedActivity
	for rows.Next() {
		var payload []byte
		var inbox string
		if err := rows.Scan(&payload, &inbox); err != nil {
			t.Fatalf("scan deliver queue: %v", err)
		}
		var activity struct {
			ID     string         `json:"id"`
			Type   string         `json:"type"`
			Actor  string         `json:"actor"`
			Object map[string]any `json:"object"`
		}
		if err := json.Unmarshal(payload, &activity); err != nil {
			t.Fatalf("decode queued activity: %v", err)
		}
		objectID, _ := activity.Object["id"].(string)
		out = append(out, queuedActivity{
			Type: activity.Type, Actor: activity.Actor,
			ActivityID: activity.ID, ObjectID: objectID, Inbox: inbox,
		})
	}
	return out
}

// requestItemIDs returns the ids of a follow request list.
func requestItemIDs(items []any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := m["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// TestFollowingRequestsListSeparatesIncomingAndOutgoing pins the two list
// endpoints: requests/list holds what other users asked of the caller,
// requests/sent what the caller asked of others, both in Misskey's
// {id, follower, followee} shape with no createdAt.
func TestFollowingRequestsListSeparatesIncomingAndOutgoing(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, bobID := f.registerUser(t, "bob")
	carolID := insertRemoteUser(t, f, "remote.example", "carol")

	incomingID := insertFollowRequest(t, f, bobID, aliceID, "")
	sentID := insertFollowRequest(t, f, aliceID, carolID, "https://remote.example/users/carol#follows/"+aliceID)

	incoming := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{})
	if len(incoming) != 1 {
		t.Fatalf("incoming requests = %v, want just the one bob sent", incoming)
	}
	item, _ := incoming[0].(map[string]any)
	if item["id"] != incomingID {
		t.Errorf("incoming id = %v, want %s", item["id"], incomingID)
	}
	if _, ok := item["createdAt"]; ok {
		t.Errorf("a follow request item must not carry createdAt: %v", item)
	}
	follower, _ := item["follower"].(map[string]any)
	followee, _ := item["followee"].(map[string]any)
	if follower["id"] != bobID || follower["username"] != "bob" {
		t.Errorf("incoming follower = %v, want bob as UserLite", item["follower"])
	}
	if followee["id"] != aliceID || followee["username"] != "alice" {
		t.Errorf("incoming followee = %v, want alice as UserLite", item["followee"])
	}

	sent := postListWithToken(t, f.ts, "/api/following/requests/sent", aliceToken, map[string]any{})
	if len(sent) != 1 {
		t.Fatalf("sent requests = %v, want just the one alice sent", sent)
	}
	item, _ = sent[0].(map[string]any)
	if item["id"] != sentID {
		t.Errorf("sent id = %v, want %s", item["id"], sentID)
	}
	follower, _ = item["follower"].(map[string]any)
	followee, _ = item["followee"].(map[string]any)
	if follower["id"] != aliceID {
		t.Errorf("sent follower = %v, want alice", item["follower"])
	}
	if followee["id"] != carolID || followee["host"] != "remote.example" {
		t.Errorf("sent followee = %v, want carol on remote.example", item["followee"])
	}

	// Each list holds one side only: bob sent the request alice received,
	// so he sees it under sent and nothing under incoming.
	if got := postListWithToken(t, f.ts, "/api/following/requests/list", bobToken, map[string]any{}); len(got) != 0 {
		t.Errorf("bob's incoming requests = %v, want none", got)
	}
	if got := requestItemIDs(postListWithToken(t, f.ts, "/api/following/requests/sent", bobToken, map[string]any{})); !reflect.DeepEqual(got, []string{incomingID}) {
		t.Errorf("bob's sent requests = %v, want the request he sent", got)
	}
}

// TestFollowingRequestsPagination pins limit and the id and date cursors.
func TestFollowingRequestsPagination(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")

	others := make([]string, 0, 4)
	for _, name := range []string{"bob", "carol", "dave", "erin"} {
		_, id := f.registerUser(t, name)
		insertFollowRequest(t, f, id, aliceID, "")
		others = append(others, id)
	}
	all := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{})
	if len(all) != len(others) {
		t.Fatalf("requests = %v, want %d", all, len(others))
	}
	ids := requestItemIDs(all)
	// Newest first: the reverse of the insertion order.
	want := ids

	page := requestItemIDs(postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{"limit": 2}))
	if !reflect.DeepEqual(page, want[:2]) {
		t.Errorf("limit 2 = %v, want %v", page, want[:2])
	}

	older := requestItemIDs(postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{"untilId": want[0]}))
	if !reflect.DeepEqual(older, want[1:]) {
		t.Errorf("untilId page = %v, want %v", older, want[1:])
	}

	// Date cursors bound created_at, not the id: the row that gets the old
	// timestamp also carries the newest id here, so an id-based filter
	// would answer each of these the other way round.
	backdated, _ := f.pool.Exec(context.Background(), `
UPDATE follow_requests SET created_at = now() - interval '2 days' WHERE id = $1
	`, want[0])
	if backdated.RowsAffected() != 1 {
		t.Fatalf("dated %s: %d rows", want[0], backdated.RowsAffected())
	}

	// A lower bound alone pages upward (makePaginationQuery), so the three
	// recent rows come back oldest first.
	since := time.Now().Add(-time.Hour).UnixMilli()
	recent := requestItemIDs(postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{"sinceDate": since}))
	want = []string{want[3], want[2], want[1]}
	if !reflect.DeepEqual(recent, want) {
		t.Errorf("sinceDate page = %v, want %v", recent, want)
	}

	// An upper bound alone stays newest first.
	until := time.Now().Add(-time.Hour).UnixMilli()
	fresh := requestItemIDs(postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, map[string]any{"untilDate": until}))
	if !reflect.DeepEqual(fresh, ids[:1]) {
		t.Errorf("untilDate page = %v, want just the backdated %v", fresh, ids[:1])
	}
}

// TestFollowingRequestsAccept pins the approve path: the request becomes
// a following, both counters move, and a second accept has nothing left
// to approve.
func TestFollowingRequestsAccept(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")
	insertFollowRequest(t, f, bobID, aliceID, "")

	resp, out := postWithToken(t, f.ts, "/api/following/requests/accept", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("accept: %d %v", resp.StatusCode, out)
	}
	if len(out) != 0 {
		t.Errorf("accept answered %v, want an empty object", out)
	}

	// The request is gone and the follow exists in both directions.
	if left := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, nil); len(left) != 0 {
		t.Errorf("requests after accept = %v, want none", left)
	}
	followers := followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID})
	if len(followers) != 1 || followers[0]["followerId"] != bobID {
		t.Fatalf("alice's followers = %v, want bob", followers)
	}

	// ... and the counters follow the new row.
	if _, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": aliceID}); show["followersCount"] != float64(1) {
		t.Errorf("alice's followersCount = %v, want 1", show["followersCount"])
	}
	if _, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": bobID}); show["followingCount"] != float64(1) {
		t.Errorf("bob's followingCount = %v, want 1", show["followingCount"])
	}

	// A local follower needs no ActivityPub answer.
	if queued := deliverQueue(t, f); len(queued) != 0 {
		t.Errorf("accept of a local request queued %v, want nothing", queued)
	}

	resp, out = postWithToken(t, f.ts, "/api/following/requests/accept", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("second accept: %d %v, want 400", resp.StatusCode, out)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "NO_FOLLOW_REQUEST" {
		t.Errorf("second accept code = %v, want NO_FOLLOW_REQUEST", e["code"])
	}
	if e["id"] != "bcde4f8b-0913-4614-8881-614e522fb041" {
		t.Errorf("second accept error id = %v", e["id"])
	}
}

// TestFollowingRequestsAcceptAnswersRemoteFollower pins the federation
// half of accept: the peer gets an Accept addressed to its inbox, with
// the Follow it originally sent embedded so it can match the answer.
func TestFollowingRequestsAcceptAnswersRemoteFollower(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	daveID := insertRemoteUser(t, f, "remote.example", "dave")

	followID := "https://remote.example/users/dave#follows/" + aliceID
	insertFollowRequest(t, f, daveID, aliceID, followID)

	if resp, out := postWithToken(t, f.ts, "/api/following/requests/accept", aliceToken, map[string]any{"userId": daveID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("accept: %d %v", resp.StatusCode, out)
	}

	queued := deliverQueue(t, f)
	if len(queued) != 1 {
		t.Fatalf("deliver queue = %v, want one Accept", queued)
	}
	got := queued[0]
	if got.Type != "Accept" {
		t.Errorf("queued activity = %q, want Accept", got.Type)
	}
	if want := "https://fish.example/users/" + aliceID; got.Actor != want {
		t.Errorf("Accept actor = %q, want %q", got.Actor, want)
	}
	if want := followID; got.ObjectID != want {
		t.Errorf("Accept embeds %q, want the original Follow %q", got.ObjectID, want)
	}
	if want := followID + "#accept"; got.ActivityID != want {
		t.Errorf("Accept id = %q, want %q", got.ActivityID, want)
	}
	if got.Inbox != "https://remote.example/inbox" {
		t.Errorf("Accept inbox = %q", got.Inbox)
	}

	// The follow itself was recorded, so the answer is not a promise.
	if _, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": aliceID}); show["followersCount"] != float64(1) {
		t.Errorf("alice's followersCount = %v, want 1", show["followersCount"])
	}
}

// TestFollowingRequestsReject pins the deny path: the request goes away,
// no follow appears, and the peer is told through a Reject.
func TestFollowingRequestsReject(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	_, bobID := f.registerUser(t, "bob")
	insertFollowRequest(t, f, bobID, aliceID, "")

	resp, out := postWithToken(t, f.ts, "/api/following/requests/reject", aliceToken, map[string]any{"userId": bobID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reject: %d %v", resp.StatusCode, out)
	}
	if len(out) != 0 {
		t.Errorf("reject answered %v, want an empty object", out)
	}

	if left := postListWithToken(t, f.ts, "/api/following/requests/list", aliceToken, nil); len(left) != 0 {
		t.Errorf("requests after reject = %v, want none", left)
	}
	if followers := followListItems(t, f.ts, "/api/users/followers", aliceToken, map[string]any{"userId": aliceID}); len(followers) != 0 {
		t.Errorf("followers after reject = %v, want none", followers)
	}
	if _, show := post(t, f.ts, "/api/users/show", map[string]any{"userId": aliceID}); show["followersCount"] != float64(0) {
		t.Errorf("alice's followersCount = %v, want 0", show["followersCount"])
	}

	// A stale button still succeeds: Misskey treats a missing request as
	// a no-op in reject.
	if resp, out := postWithToken(t, f.ts, "/api/following/requests/reject", aliceToken, map[string]any{"userId": bobID}); resp.StatusCode != http.StatusOK {
		t.Errorf("second reject: %d %v, want 200", resp.StatusCode, out)
	}

	// A remote follower is answered with the original Follow wrapped in a
	// Reject.
	erinID := insertRemoteUser(t, f, "remote.example", "erin")
	followID := "https://remote.example/users/erin#follows/" + aliceID
	insertFollowRequest(t, f, erinID, aliceID, followID)
	if resp, out := postWithToken(t, f.ts, "/api/following/requests/reject", aliceToken, map[string]any{"userId": erinID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("reject remote: %d %v", resp.StatusCode, out)
	}
	queued := deliverQueue(t, f)
	if len(queued) != 1 {
		t.Fatalf("deliver queue = %v, want one Reject", queued)
	}
	if queued[0].Type != "Reject" || queued[0].ObjectID != followID {
		t.Errorf("queued activity = %v, want a Reject of the original Follow", queued[0])
	}
}

// TestFollowingRequestsCancel pins the withdraw path: an outgoing request
// disappears, the followee is returned as UserLite, and a second cancel
// reports that there was nothing to cancel.
func TestFollowingRequestsCancel(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	daveID := insertRemoteUser(t, f, "remote.example", "dave")
	followID := "https://remote.example/users/dave#follows/" + aliceID
	insertFollowRequest(t, f, aliceID, daveID, followID)

	resp, out := postWithToken(t, f.ts, "/api/following/requests/cancel", aliceToken, map[string]any{"userId": daveID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel: %d %v", resp.StatusCode, out)
	}
	if out["id"] != daveID || out["username"] != "dave" {
		t.Errorf("cancel returned %v, want dave as UserLite", out)
	}

	if left := postListWithToken(t, f.ts, "/api/following/requests/sent", aliceToken, nil); len(left) != 0 {
		t.Errorf("sent requests after cancel = %v, want none", left)
	}
	// The peer is told the Follow is withdrawn.
	queued := deliverQueue(t, f)
	if len(queued) != 1 || queued[0].Type != "Undo" {
		t.Errorf("deliver queue = %v, want one Undo", queued)
	}

	resp, out = postWithToken(t, f.ts, "/api/following/requests/cancel", aliceToken, map[string]any{"userId": daveID})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("second cancel: %d %v, want 400", resp.StatusCode, out)
	}
	e, _ := out["error"].(map[string]any)
	if e["code"] != "FOLLOW_REQUEST_NOT_FOUND" {
		t.Errorf("second cancel code = %v, want FOLLOW_REQUEST_NOT_FOUND", e["code"])
	}
	if e["id"] != "089b125b-d338-482a-9a09-e2622ac9f8d4" {
		t.Errorf("second cancel error id = %v", e["id"])
	}
}

// TestFollowingRequestsUnknownUser pins NO_SUCH_USER with each endpoint's
// own Misskey error id.
func TestFollowingRequestsUnknownUser(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")

	cases := []struct {
		path string
		id   string
	}{
		{"/api/following/requests/accept", "66ce1645-d66c-46bb-8b79-96739af885bd"},
		{"/api/following/requests/reject", "abc2ffa6-25b2-4380-ba99-321ff3a94555"},
		{"/api/following/requests/cancel", "4e68c551-fc4c-4e46-bb41-7d4a37bf9dab"},
	}
	for _, tc := range cases {
		resp, out := postWithToken(t, f.ts, tc.path, aliceToken, map[string]any{"userId": "00000000000000000000"})
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

// TestFollowingRequestsNeedCredential pins requireCredential on all five
// endpoints: an anonymous caller gets the neighbours' 401.
func TestFollowingRequestsNeedCredential(t *testing.T) {
	f := newFederationServer(t, "public")
	_, aliceID := f.registerUser(t, "alice")

	paths := []string{
		"/api/following/requests/list",
		"/api/following/requests/sent",
		"/api/following/requests/accept",
		"/api/following/requests/reject",
		"/api/following/requests/cancel",
	}
	for _, path := range paths {
		resp, out := post(t, f.ts, path, map[string]any{"userId": aliceID})
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", path, resp.StatusCode)
		}
		if resp.Header.Get("WWW-Authenticate") == "" {
			t.Errorf("%s: 401 without WWW-Authenticate", path)
		}
		e, _ := out["error"].(map[string]any)
		if e["code"] != "AUTHENTICATION_FAILED" {
			t.Errorf("%s: code = %v, want AUTHENTICATION_FAILED", path, e["code"])
		}
	}
}
