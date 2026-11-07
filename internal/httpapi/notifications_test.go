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

func TestINotificationsAliasMatchesCanonical(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)
	payload := map[string]any{"i": tokenA, "markAsRead": false}

	canonicalResponse, canonical := postRaw(t, ts, "/api/notifications", payload)
	if canonicalResponse.StatusCode != http.StatusOK {
		t.Fatalf("canonical notifications status = %d, body %v", canonicalResponse.StatusCode, canonical)
	}

	aliasResponse, alias := postRaw(t, ts, "/api/i/notifications", payload)
	if aliasResponse.StatusCode != http.StatusOK {
		t.Fatalf("notifications alias status = %d, body %v", aliasResponse.StatusCode, alias)
	}
	if !reflect.DeepEqual(alias, canonical) {
		t.Errorf("notifications alias response = %v, want canonical response %v", alias, canonical)
	}
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

	// An explicitly empty includeTypes means no filter, exactly like
	// omitting it: Misskey filters only when the list is non-empty
	// (NotificationService.ts L302).
	all := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{},
	})
	if len(all) != 2 {
		t.Errorf("empty includeTypes returned %v, want both", notificationTypesOf(all))
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

// MeDetailed carries the unread notification state
// (packages/backend/src/core/entities/UserEntityService.ts L612 and L614):
// unreadNotificationsCount is the number of unread rows and
// hasUnreadNotification is unread > 0. The web shell badge and the
// "mark all as read" button both read these.
func TestIUnreadNotificationCount(t *testing.T) {
	ts := newTestServer(t, "open")
	nameA := uniqueName(t) + "a"
	_, tokenA := createUser(t, ts, nameA)
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	resp, me := post(t, ts, "/api/i", map[string]any{"i": tokenA})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/i: status %d, body %v", resp.StatusCode, me)
	}
	if me["unreadNotificationsCount"] != float64(0) {
		t.Errorf("unreadNotificationsCount = %v, want 0 for a fresh account", me["unreadNotificationsCount"])
	}
	if me["hasUnreadNotification"] != false {
		t.Errorf("hasUnreadNotification = %v, want false for a fresh account", me["hasUnreadNotification"])
	}

	id := createNote(t, ts, tokenA, map[string]any{"text": "react to me"})
	post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "🎉",
	})

	resp, me = post(t, ts, "/api/i", map[string]any{"i": tokenA})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/i after a reaction: status %d, body %v", resp.StatusCode, me)
	}
	if me["unreadNotificationsCount"] != float64(1) {
		t.Errorf("unreadNotificationsCount = %v, want 1", me["unreadNotificationsCount"])
	}
	if me["hasUnreadNotification"] != true {
		t.Errorf("hasUnreadNotification = %v, want true", me["hasUnreadNotification"])
	}

	if resp, body := post(t, ts, "/api/notifications/mark-all-as-read", map[string]any{"i": tokenA}); resp.StatusCode != http.StatusOK {
		t.Fatalf("mark-all-as-read: status %d, body %v", resp.StatusCode, body)
	}

	resp, me = post(t, ts, "/api/i", map[string]any{"i": tokenA})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/i after mark-all-as-read: status %d, body %v", resp.StatusCode, me)
	}
	if me["unreadNotificationsCount"] != float64(0) || me["hasUnreadNotification"] != false {
		t.Errorf("unread state after mark-all-as-read = %v/%v, want 0/false",
			me["unreadNotificationsCount"], me["hasUnreadNotification"])
	}
}

// notificationsOfType filters a notification list to one type.
func notificationsOfType(list []map[string]any, typ string) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, n := range list {
		if n["type"] == typ {
			out = append(out, n)
		}
	}
	return out
}

// New notes notify their reply target and the author of the renoted note
// (packages/backend/src/core/NoteCreateService.ts L856-895). The
// notification carries the new note's id, and a reply that also mentions
// its target yields one "reply" notification, not a reply and a mention
// (NotificationManager.push, L85-103).
func TestReplyAndRenoteNotifications(t *testing.T) {
	ts := newTestServer(t, "open")
	nameA := uniqueName(t) + "a"
	_, tokenA := createUser(t, ts, nameA)
	idB, tokenB := createUser(t, ts, uniqueName(t)+"b")

	parent := createNote(t, ts, tokenA, map[string]any{"text": "parent"})

	replyID := createNote(t, ts, tokenB, map[string]any{"text": "a reply", "replyId": parent})
	replies := notificationsOfType(fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false}), "reply")
	if len(replies) != 1 {
		t.Fatalf("reply notifications = %d, want 1", len(replies))
	}
	if replies[0]["noteId"] != replyID {
		t.Errorf("reply notification noteId = %v, want %s", replies[0]["noteId"], replyID)
	}
	if replies[0]["userId"] != idB {
		t.Errorf("reply notification userId = %v, want %s", replies[0]["userId"], idB)
	}

	// A reply that mentions the same user is still a single reply.
	createNote(t, ts, tokenB, map[string]any{
		"text": "@" + nameA + " mention and reply", "replyId": parent,
	})
	list := fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got := len(notificationsOfType(list, "reply")); got != 2 {
		t.Errorf("reply notifications = %d, want 2", got)
	}
	if got := len(notificationsOfType(list, "mention")); got != 0 {
		t.Errorf("mention notifications = %d, want 0 (the reply reason wins)", got)
	}

	// Replying to one's own note notifies nobody.
	createNote(t, ts, tokenA, map[string]any{"text": "talking to myself", "replyId": parent})
	list = fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got := len(notificationsOfType(list, "reply")); got != 2 {
		t.Errorf("self reply produced a notification: reply count = %d, want 2", got)
	}

	// A pure renote notifies the original author as "renote".
	renoteID := createNote(t, ts, tokenB, map[string]any{"renoteId": parent})
	list = fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	renotes := notificationsOfType(list, "renote")
	if len(renotes) != 1 {
		t.Fatalf("renote notifications = %d, want 1", len(renotes))
	}
	if renotes[0]["noteId"] != renoteID {
		t.Errorf("renote notification noteId = %v, want %s", renotes[0]["noteId"], renoteID)
	}

	// A quote is its own type (misc/is-renote.ts isQuote).
	quoteID := createNote(t, ts, tokenB, map[string]any{"renoteId": parent, "text": "quoting"})
	list = fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	quotes := notificationsOfType(list, "quote")
	if len(quotes) != 1 {
		t.Fatalf("quote notifications = %d, want 1", len(quotes))
	}
	if quotes[0]["noteId"] != quoteID {
		t.Errorf("quote notification noteId = %v, want %s", quotes[0]["noteId"], quoteID)
	}

	// Renoting one's own note notifies nobody.
	createNote(t, ts, tokenA, map[string]any{"renoteId": parent})
	list = fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got := len(notificationsOfType(list, "renote")); got != 1 {
		t.Errorf("self renote produced a notification: renote count = %d, want 1", got)
	}
}

// A specified note only notifies the users it is addressed to
// (NotificationManager.notify, packages/backend/src/core/NoteCreateService.ts
// L105-142), so a reply whose audience excludes its own target does not
// notify that target.
func TestNotificationRespectsNoteVisibility(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")
	idC, tokenC := createUser(t, ts, uniqueName(t)+"c")

	parent := createNote(t, ts, tokenA, map[string]any{"text": "parent"})
	createNote(t, ts, tokenB, map[string]any{
		"replyId": parent, "text": "for c only",
		"visibility": "specified", "visibleUserIds": []string{idC},
	})

	if got := fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false}); len(got) != 0 {
		t.Errorf("reply target got %d notifications for a note it cannot read: %v",
			len(got), notificationTypesOf(got))
	}
	if got := fetchNotifications(t, ts, tokenC, map[string]any{"markAsRead": false}); len(got) != 0 {
		t.Errorf("addressed bystander got %d notifications: %v", len(got), notificationTypesOf(got))
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

// The HTTP notification list must not carry a spurious body field: the
// upstream Notification type only has body on app-type notifications,
// which this server never stores.
func TestNotificationHasNoBodyField(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)
	list := fetchNotifications(t, ts, tokenA, map[string]any{})
	if len(list) == 0 {
		t.Fatal("no notifications")
	}
	for _, n := range list {
		if _, present := n["body"]; present {
			t.Errorf("notification carries body field: %v", n)
		}
	}
}

// TestNotificationTypeMuting covers mutingNotificationTypes: the muted kind
// disappears from the list and the unread count, an explicit includeTypes still
// wins, and the setting validates its values.
func TestNotificationTypeMuting(t *testing.T) {
	ts, tokenA := seedTwoNotificationTypes(t)

	// Before muting both kinds are visible.
	all := fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got := len(all); got != 2 {
		t.Fatalf("notifications before muting = %d, want 2", got)
	}
	_, me := post(t, ts, "/api/i", map[string]any{"i": tokenA})
	if me["unreadNotificationsCount"] != float64(2) {
		t.Fatalf("unread before muting = %v, want 2", me["unreadNotificationsCount"])
	}

	// Mute reactions.
	resp, updated := post(t, ts, "/api/i/update", map[string]any{
		"i": tokenA, "mutingNotificationTypes": []string{"reaction"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update muting = %d %v", resp.StatusCode, updated)
	}
	if muted, _ := updated["mutingNotificationTypes"].([]any); len(muted) != 1 || muted[0] != "reaction" {
		t.Errorf("mutingNotificationTypes = %v", updated["mutingNotificationTypes"])
	}

	visible := fetchNotifications(t, ts, tokenA, map[string]any{"markAsRead": false})
	if got, want := notificationTypesOf(visible), []string{"mention"}; !reflect.DeepEqual(got, want) {
		t.Errorf("list after muting = %v, want %v", got, want)
	}
	_, me = post(t, ts, "/api/i", map[string]any{"i": tokenA})
	if me["unreadNotificationsCount"] != float64(1) {
		t.Errorf("unread after muting = %v, want 1", me["unreadNotificationsCount"])
	}

	// An explicit includeTypes still returns the muted kind.
	included := fetchNotifications(t, ts, tokenA, map[string]any{
		"markAsRead": false, "includeTypes": []string{"reaction"},
	})
	if got, want := notificationTypesOf(included), []string{"reaction"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes after muting = %v, want %v", got, want)
	}

	// Unknown types are refused; duplicates are collapsed; clearing works.
	if resp, _ := post(t, ts, "/api/i/update", map[string]any{
		"i": tokenA, "mutingNotificationTypes": []string{"carrier-pigeon"},
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown type status = %d, want 400", resp.StatusCode)
	}
	resp, deduped := post(t, ts, "/api/i/update", map[string]any{
		"i": tokenA, "mutingNotificationTypes": []string{"reaction", "reaction", "reply"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dedupe update = %d %v", resp.StatusCode, deduped)
	}
	if muted, _ := deduped["mutingNotificationTypes"].([]any); len(muted) != 2 {
		t.Errorf("deduped types = %v, want two", deduped["mutingNotificationTypes"])
	}
	resp, cleared := post(t, ts, "/api/i/update", map[string]any{"i": tokenA, "mutingNotificationTypes": nil})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear muting = %d %v", resp.StatusCode, cleared)
	}
	// The packer renders an empty list as [] rather than null.
	if muted, _ := cleared["mutingNotificationTypes"].([]any); len(muted) != 0 {
		t.Errorf("cleared muting = %v, want an empty list", cleared["mutingNotificationTypes"])
	}
}
