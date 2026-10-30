package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
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
