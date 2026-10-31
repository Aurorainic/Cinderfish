package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

// fetchNotifications posts to /api/notifications with a bearer token and
// returns the decoded list.
func fetchNotifications(t *testing.T, ts *httptest.Server, token string, params map[string]any) []map[string]any {
	t.Helper()
	payload := map[string]any{"i": token}
	for k, v := range params {
		payload[k] = v
	}
	resp, raw := postRaw(t, ts, "/api/notifications", payload)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/notifications: status %d, body %v", resp.StatusCode, raw)
	}
	list, ok := raw.([]any)
	if !ok {
		t.Fatalf("/api/notifications did not return an array: %v", raw)
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("notification item is not an object: %v", item)
		}
		out = append(out, m)
	}
	return out
}

func notificationTypesOf(list []map[string]any) []string {
	out := make([]string, 0, len(list))
	for _, n := range list {
		typ, _ := n["type"].(string)
		out = append(out, typ)
	}
	return out
}

// seedTwoNotificationTypes creates one reaction and one mention notification
// for user A, newest mention last.
func seedTwoNotificationTypes(t *testing.T) (ts *httptest.Server, tokenA string) {
	t.Helper()
	ts = newTestServer(t, "open")
	nameA := uniqueName(t) + "a"
	_, tokenA = createUser(t, ts, nameA)
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	id := createNote(t, ts, tokenA, map[string]any{"text": "notify me"})
	post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "🎉",
	})
	createNote(t, ts, tokenB, map[string]any{"text": "@" + nameA + " hello"})
	return ts, tokenA
}

func TestNotificationsIncludeTypes(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)

	reactions := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{"reaction"},
	})
	if got, want := notificationTypesOf(reactions), []string{"reaction"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes reaction = %v, want %v", got, want)
	}

	mentions := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{"mention"},
	})
	if got, want := notificationTypesOf(mentions), []string{"mention"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes mention = %v, want %v", got, want)
	}

	// An explicitly empty includeTypes returns nothing.
	none := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{},
	})
	if len(none) != 0 {
		t.Errorf("empty includeTypes returned %v, want nothing", notificationTypesOf(none))
	}
}

func TestNotificationsExcludeTypes(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)

	list := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "excludeTypes": []string{"reaction"},
	})
	if got, want := notificationTypesOf(list), []string{"mention"}; !reflect.DeepEqual(got, want) {
		t.Errorf("excludeTypes reaction = %v, want %v", got, want)
	}

	all := fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got, want := notificationTypesOf(all), []string{"mention", "reaction"}; !reflect.DeepEqual(got, want) {
		t.Errorf("unfiltered list = %v, want %v", got, want)
	}
}

// seedNotificationSeries creates, in chronological order, three mention
// notes and two distinct reactions from user B aimed at user A, so the
// notification stream interleaves two types. The sleeps keep the
// millisecond-stamped ids monotonic.
func seedNotificationSeries(t *testing.T) (ts *httptest.Server, tokenA string, order []string) {
	t.Helper()
	ts = newTestServer(t, "open")
	nameA := uniqueName(t) + "a"
	_, tokenA = createUser(t, ts, nameA)
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	id := createNote(t, ts, tokenA, map[string]any{"text": "notify me"})
	time.Sleep(5 * time.Millisecond)
	for _, step := range []struct{ kind, emoji string }{
		{"mention", ""}, {"reaction", "🎉"}, {"mention", ""}, {"reaction", "👍"}, {"mention", ""},
	} {
		if step.kind == "mention" {
			createNote(t, ts, tokenB, map[string]any{"text": "@" + nameA + " ping"})
		} else {
			post(t, ts, "/api/notes/reactions/create", map[string]any{
				"i": tokenB, "noteId": id, "reaction": step.emoji,
			})
		}
		time.Sleep(5 * time.Millisecond)
		order = append(order, step.kind)
	}
	return ts, tokenA, order
}

// orderedNotifications fetches everything with markAsRead=false and returns
// the item maps newest first, matching the endpoint's ordering.
func orderedNotifications(t *testing.T, ts *httptest.Server, tokenA string) []map[string]any {
	t.Helper()
	return fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
}

func notificationIDsOf(list []map[string]any) []string {
	out := make([]string, 0, len(list))
	for _, n := range list {
		out = append(out, n["id"].(string))
	}
	return out
}

// TestNotificationsCombinedIncludeAndExclude pins Misskey's include-wins
// semantics (packages/backend/src/core/NotificationService.ts L302-306):
// a non-empty includeTypes takes precedence and excludeTypes is ignored,
// whatever it names.
func TestNotificationsCombinedIncludeAndExclude(t *testing.T) {
	ts, tokenA, _ := seedNotificationSeries(t)

	unfiltered := orderedNotifications(t, ts, tokenA)

	// Include wins: exclude mention is ignored while include covers both
	// types, so the page is identical to the unfiltered one.
	list := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead":   false,
		"includeTypes": []string{"mention", "reaction"},
		"excludeTypes": []string{"mention"},
	})
	if got, want := notificationIDsOf(list), notificationIDsOf(unfiltered); !reflect.DeepEqual(got, want) {
		t.Errorf("include mention+reaction with exclude mention = %v, want %v (include wins)", got, want)
	}

	// Include-only still filters to the named types.
	mentions := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead":   false,
		"includeTypes": []string{"mention"},
	})
	for _, n := range mentions {
		if n["type"] != "mention" {
			t.Errorf("include mention-only returned a %v notification", n["type"])
		}
	}

	// Exclude-only keeps filtering out the named types.
	noMentions := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead":   false,
		"excludeTypes": []string{"mention"},
	})
	for _, n := range noMentions {
		if n["type"] == "mention" {
			t.Errorf("exclude mention returned a mention notification")
		}
	}
}

// TestNotificationsFilteredCursorPagination paginates a filtered stream and
// checks the pages join without gaps or repeats: the union of the paged
// fetches must equal one unfiltered-by-cursor fetch of the same type.
func TestNotificationsFilteredCursorPagination(t *testing.T) {
	ts, tokenA, order := seedNotificationSeries(t)

	var paged []string
	untilID := ""
	for round := 0; round < 10; round++ {
		params := map[string]any{
			"markAsRead":   false,
			"includeTypes": []string{"mention"},
			"limit":        2,
		}
		if untilID != "" {
			params["untilId"] = untilID
		}
		page := fetchNotifications(t, ts, tokenA, params)
		if len(page) == 0 {
			break
		}
		for _, n := range page {
			paged = append(paged, n["id"].(string))
		}
		untilID = page[len(page)-1]["id"].(string)
	}

	expected := make([]string, 0, len(order))
	for _, n := range orderedNotifications(t, ts, tokenA) {
		if n["type"] == "mention" {
			expected = append(expected, n["id"].(string))
		}
	}
	if !reflect.DeepEqual(paged, expected) {
		t.Errorf("paged mention stream = %v, want the full ordered mention list %v", paged, expected)
	}
	seen := map[string]bool{}
	for _, id := range paged {
		if seen[id] {
			t.Errorf("notification %s appeared twice across pages", id)
		}
		seen[id] = true
	}
}

// TestNotificationsSinceIdAscending pins Misskey's makePaginationQuery
// semantics (packages/backend/src/core/QueryService.ts L58-60): sinceId
// without untilId returns the newer remainder in ascending order.
func TestNotificationsSinceIdAscending(t *testing.T) {
	ts, tokenA, _ := seedNotificationSeries(t)

	all := orderedNotifications(t, ts, tokenA)
	if len(all) < 3 {
		t.Fatalf("seeded %d notifications, want at least 3", len(all))
	}
	oldest := all[len(all)-1]["id"].(string)

	page := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "sinceId": oldest, "limit": 10,
	})
	want := make([]string, 0, len(all)-1)
	for i := len(all) - 2; i >= 0; i-- {
		want = append(want, all[i]["id"].(string))
	}
	if got := notificationIDsOf(page); !reflect.DeepEqual(got, want) {
		t.Errorf("sinceId-only page = %v, want %v (ascending)", got, want)
	}
}

func TestNotificationsMarkAsReadDefaultsToTrue(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)

	for _, n := range fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false}) {
		if n["isRead"] != false {
			t.Fatalf("notification %v is read before any markAsRead call", n["id"])
		}
	}

	// Omitting markAsRead marks the returned notifications read.
	for _, n := range fetchNotifications(t, ts, tokenA, nil) {
		if n["isRead"] != true {
			t.Errorf("notification %v is not read after a default fetch", n["id"])
		}
	}

	// The flag is persisted, so a later markAsRead=false still sees it.
	for _, n := range fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false}) {
		if n["isRead"] != true {
			t.Errorf("notification %v reverted to unread", n["id"])
		}
	}
}

func TestNotificationsMarkAsReadOnlyTouchesReturnedTypes(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)

	// Mark only the reactions read; the mention must stay unread.
	fetchNotifications(t, ts, tokenA, map[string]any{"includeTypes": []string{"reaction"}})

	mentions := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{"mention"},
	})
	if len(mentions) != 1 {
		t.Fatalf("mention list = %v, want one notification", notificationTypesOf(mentions))
	}
	if mentions[0]["isRead"] != false {
		t.Errorf("mention notification was marked read by a reaction-only fetch")
	}
}
