package httpapi_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createUser registers a user and returns its id and native token.
func createUser(t *testing.T, ts *httptest.Server, username string) (string, string) {
	t.Helper()
	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": username,
		"password": "hunter2",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup %s: status %d, body %v", username, resp.StatusCode, body)
	}
	id, _ := body["id"].(string)
	token, _ := body["token"].(string)
	if id == "" || token == "" {
		t.Fatalf("signup %s returned no id/token: %v", username, body)
	}
	return id, token
}

// createNote posts a note and returns its id.
func createNote(t *testing.T, ts *httptest.Server, token string, params map[string]any) string {
	t.Helper()
	params["i"] = token
	resp, raw := postRaw(t, ts, "/api/notes/create", params)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/create: status %d, body %v", resp.StatusCode, raw)
	}
	body, _ := raw.(map[string]any)
	created, ok := body["createdNote"].(map[string]any)
	if !ok {
		t.Fatalf("no createdNote in %v", body)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("no note id in %v", created)
	}
	return id
}

// timelineIDs returns the ids of the notes in a timeline response.
func timelineIDs(t *testing.T, ts *httptest.Server, path, token string, extra map[string]any) []string {
	t.Helper()
	payload := map[string]any{}
	for k, v := range extra {
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
	ids := make([]string, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["id"].(string); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func contains(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func TestNotesCreateAndShow(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	id := createNote(t, ts, token, map[string]any{"text": "hello world"})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notes/show: status %d, body %v", resp.StatusCode, body)
	}
	if body["text"] != "hello world" {
		t.Errorf("text = %v, want hello world", body["text"])
	}
	if body["visibility"] != "public" {
		t.Errorf("visibility = %v, want public", body["visibility"])
	}
	if body["localOnly"] != false {
		t.Errorf("localOnly = %v, want false", body["localOnly"])
	}
	// The author is embedded.
	user, ok := body["user"].(map[string]any)
	if !ok {
		t.Fatalf("no embedded user: %v", body)
	}
	if user["username"] == nil {
		t.Error("embedded user has no username")
	}
}

func TestNotesCreateRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/notes/create", map[string]any{"text": "x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	if body["error"] == nil {
		t.Errorf("no error envelope: %v", body)
	}
}

func TestNotesCreateRejectsEmpty(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	resp, body := post(t, ts, "/api/notes/create", map[string]any{"i": token})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "NOTE_EMPTY" {
		t.Errorf("code = %v, want NOTE_EMPTY", errObj["code"])
	}
}

func TestNotesCreateVisibilityLevels(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	for _, visibility := range []string{"public", "home", "followers"} {
		id := createNote(t, ts, token, map[string]any{
			"text":       "note " + visibility,
			"visibility": visibility,
		})
		resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("show %s: status %d", visibility, resp.StatusCode)
		}
		if body["visibility"] != visibility {
			t.Errorf("visibility = %v, want %v", body["visibility"], visibility)
		}
	}
}

func TestNotesDelete(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	id := createNote(t, ts, token, map[string]any{"text": "to delete"})

	resp, _ := post(t, ts, "/api/notes/delete", map[string]any{"i": token, "noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete: status %d", resp.StatusCode)
	}

	resp, _ = post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("show after delete: status %d, want 404", resp.StatusCode)
	}
}

func TestNotesDeleteByOtherUserFails(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")
	id := createNote(t, ts, tokenA, map[string]any{"text": "not yours"})

	resp, _ := post(t, ts, "/api/notes/delete", map[string]any{"i": tokenB, "noteId": id})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestLocalTimelineOnlyHasPublicLocalNotes(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	publicID := createNote(t, ts, token, map[string]any{"text": "public"})
	homeID := createNote(t, ts, token, map[string]any{"text": "home", "visibility": "home"})
	followersID := createNote(t, ts, token, map[string]any{"text": "followers", "visibility": "followers"})

	ids := timelineIDs(t, ts, "/api/notes/local-timeline", "", nil)
	if !contains(ids, publicID) {
		t.Error("the local timeline must contain public notes")
	}
	if contains(ids, homeID) {
		t.Error("the local timeline must not contain home visibility notes")
	}
	if contains(ids, followersID) {
		t.Error("the local timeline must not contain followers visibility notes")
	}
}

func TestGlobalTimelineRequiresPublicVisibility(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	publicID := createNote(t, ts, token, map[string]any{"text": "public"})
	homeID := createNote(t, ts, token, map[string]any{"text": "home", "visibility": "home"})

	ids := timelineIDs(t, ts, "/api/notes/global-timeline", "", nil)
	if !contains(ids, publicID) {
		t.Error("the global timeline must contain public notes")
	}
	if contains(ids, homeID) {
		t.Error("the global timeline must not contain home visibility notes")
	}
}

func TestHybridTimelineRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, _ := post(t, ts, "/api/notes/hybrid-timeline", map[string]any{})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestHybridTimelineContainsLocalPublicNotes(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	publicID := createNote(t, ts, token, map[string]any{"text": "public"})

	ids := timelineIDs(t, ts, "/api/notes/hybrid-timeline", token, nil)
	if !contains(ids, publicID) {
		t.Error("the hybrid timeline must contain public local notes")
	}
}

func TestTimelineCursorPagination(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	first := createNote(t, ts, token, map[string]any{"text": "one"})
	second := createNote(t, ts, token, map[string]any{"text": "two"})
	third := createNote(t, ts, token, map[string]any{"text": "three"})

	// Newest first.
	ids := timelineIDs(t, ts, "/api/notes/local-timeline", "", map[string]any{"limit": 10})
	if len(ids) != 3 {
		t.Fatalf("got %d notes, want 3: %v", len(ids), ids)
	}
	if ids[0] != third || ids[1] != second || ids[2] != first {
		t.Errorf("order = %v, want newest first", ids)
	}

	// untilId excludes the note itself and everything newer.
	ids = timelineIDs(t, ts, "/api/notes/local-timeline", "", map[string]any{"untilId": third})
	if contains(ids, third) {
		t.Error("untilId must exclude the note itself")
	}
	if !contains(ids, second) {
		t.Error("untilId must keep older notes")
	}

	// sinceId keeps only newer notes.
	ids = timelineIDs(t, ts, "/api/notes/local-timeline", "", map[string]any{"sinceId": first})
	if contains(ids, first) {
		t.Error("sinceId must exclude the note itself")
	}
	if !contains(ids, third) {
		t.Error("sinceId must keep newer notes")
	}
}

func TestTimelineLimit(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	for i := range 5 {
		createNote(t, ts, token, map[string]any{"text": "note"})
		_ = i
	}
	ids := timelineIDs(t, ts, "/api/notes/local-timeline", "", map[string]any{"limit": 2})
	if len(ids) != 2 {
		t.Errorf("got %d notes, want 2", len(ids))
	}
}

func TestTimelineWithReplies(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	parent := createNote(t, ts, token, map[string]any{"text": "parent"})
	reply := createNote(t, ts, token, map[string]any{"text": "reply", "replyId": parent})

	// By default replies are excluded.
	ids := timelineIDs(t, ts, "/api/notes/local-timeline", "", nil)
	if contains(ids, reply) {
		t.Error("replies must be excluded by default")
	}

	// With withReplies they appear.
	ids = timelineIDs(t, ts, "/api/notes/local-timeline", "", map[string]any{"withReplies": true})
	if !contains(ids, reply) {
		t.Error("withReplies must include replies")
	}
}

func TestReplyCountsAreTracked(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	parent := createNote(t, ts, token, map[string]any{"text": "parent"})
	createNote(t, ts, token, map[string]any{"text": "reply", "replyId": parent})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": parent})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if body["repliesCount"] != float64(1) {
		t.Errorf("repliesCount = %v, want 1", body["repliesCount"])
	}
}

func TestReactions(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")
	id := createNote(t, ts, tokenA, map[string]any{"text": "react to me"})

	resp, body := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "👍",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reactions/create: status %d, body %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show: status %d", resp.StatusCode)
	}
	if body["reactionCount"] != float64(1) {
		t.Errorf("reactionCount = %v, want 1", body["reactionCount"])
	}
	reactions, _ := body["reactions"].(map[string]any)
	if reactions["👍"] != float64(1) {
		t.Errorf("reactions = %v, want one thumbs up", reactions)
	}

	// The reaction is removed again.
	resp, _ = post(t, ts, "/api/notes/reactions/delete", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "👍",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reactions/delete: status %d", resp.StatusCode)
	}
	resp, body = post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if body["reactionCount"] != float64(0) {
		t.Errorf("reactionCount after delete = %v, want 0", body["reactionCount"])
	}
}

func TestReactionCreatesNotification(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")
	id := createNote(t, ts, tokenA, map[string]any{"text": "notify me"})

	post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "🎉",
	})

	resp, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": tokenA})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("notifications: status %d, body %v", resp.StatusCode, raw)
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		t.Fatalf("no notifications: %v", raw)
	}
	first, _ := list[0].(map[string]any)
	if first["type"] != "reaction" {
		t.Errorf("notification type = %v, want reaction", first["type"])
	}
}

func TestMarkAllAsRead(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")
	id := createNote(t, ts, tokenA, map[string]any{"text": "x"})

	post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenB, "noteId": id, "reaction": "👍",
	})

	resp, _ := post(t, ts, "/api/notifications/mark-all-as-read", map[string]any{"i": tokenA})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}

	resp, raw := postRaw(t, ts, "/api/notifications", map[string]any{"i": tokenA})
	list, _ := raw.([]any)
	for _, item := range list {
		m, _ := item.(map[string]any)
		if m["isRead"] != true {
			t.Errorf("notification is not marked as read: %v", m)
		}
	}
}

func TestPoll(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	id := createNote(t, ts, token, map[string]any{
		"text": "vote please",
		"poll": map[string]any{
			"choices": []string{"a", "b"},
		},
	})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show: status %d, body %v", resp.StatusCode, body)
	}
	if body["hasPoll"] != true {
		t.Errorf("hasPoll = %v, want true", body["hasPoll"])
	}
	poll, ok := body["poll"].(map[string]any)
	if !ok {
		t.Fatalf("no poll: %v", body)
	}
	choices, _ := poll["choices"].([]any)
	if len(choices) != 2 {
		t.Errorf("choices = %v, want 2", choices)
	}
}

func TestRenote(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	original := createNote(t, ts, tokenA, map[string]any{"text": "original"})
	renote := createNote(t, ts, tokenB, map[string]any{"renoteId": original})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": renote})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show: status %d", resp.StatusCode)
	}
	if body["renoteId"] != original {
		t.Errorf("renoteId = %v, want %v", body["renoteId"], original)
	}
	embedded, ok := body["renote"].(map[string]any)
	if !ok {
		t.Fatalf("no embedded renote: %v", body)
	}
	if embedded["id"] != original {
		t.Errorf("embedded renote id = %v", embedded["id"])
	}

	// The original's renote count went up.
	resp, body = post(t, ts, "/api/notes/show", map[string]any{"noteId": original})
	if body["renoteCount"] != float64(1) {
		t.Errorf("renoteCount = %v, want 1", body["renoteCount"])
	}
}

// Misskey rejects renoting a followers-only note by anyone but its author
// and rejects renoting a specified note outright (core/NoteCreateService.ts
// L352-356, cannotRenoteDueToVisibility be9529e9-fe72-4de0-ae43-0b363c4938af).
func TestRenoteTargetVisibility(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	idB, tokenB := createUser(t, ts, uniqueName(t)+"b")

	followers := createNote(t, ts, tokenA, map[string]any{
		"text":       "followers only",
		"visibility": "followers",
	})
	specified := createNote(t, ts, tokenA, map[string]any{
		"text":           "just for you",
		"visibility":     "specified",
		"visibleUserIds": []string{idB},
	})

	resp, body := post(t, ts, "/api/notes/create", map[string]any{"i": tokenB, "renoteId": followers})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("renote followers as other: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_RENOTE_DUE_TO_VISIBILITY" {
		t.Errorf("code = %v, want CANNOT_RENOTE_DUE_TO_VISIBILITY", errObj["code"])
	}

	resp, body = post(t, ts, "/api/notes/create", map[string]any{"i": tokenB, "renoteId": specified})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("renote specified: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ = body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_RENOTE_DUE_TO_VISIBILITY" {
		t.Errorf("code = %v, want CANNOT_RENOTE_DUE_TO_VISIBILITY", errObj["code"])
	}

	// The author may still renote their own followers note.
	createNote(t, ts, tokenA, map[string]any{"renoteId": followers})
}

// A quote (renoteId + own text/cw/files/poll/reply) is NOT a pure renote:
// upstream checks isRenote(target) && !isQuote(target)
// (core/NoteCreateService.ts L335/L384, misc/is-renote.ts), so quotes
// stay replyable and renotable while pure renotes are rejected with the
// upstream error codes.
func TestQuoteReplyAndRenoteTargets(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	original := createNote(t, ts, tokenA, map[string]any{"text": "original"})
	quote := createNote(t, ts, tokenB, map[string]any{"renoteId": original, "text": "a quote"})
	pureRenote := createNote(t, ts, tokenB, map[string]any{"renoteId": original})

	// Replying to a pure renote is rejected.
	resp, body := post(t, ts, "/api/notes/create", map[string]any{"i": tokenA, "replyId": pureRenote, "text": "reply to renote"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reply to pure renote: status %d, body %v", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "CANNOT_REPLY_TO_A_PURE_RENOTE" {
		t.Errorf("code = %v, want CANNOT_REPLY_TO_A_PURE_RENOTE", errObj["code"])
	}

	// Renoting a pure renote is rejected.
	resp, body = post(t, ts, "/api/notes/create", map[string]any{"i": tokenA, "renoteId": pureRenote})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("renote of pure renote: status %d, body %v", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "CANNOT_RENOTE_TO_A_PURE_RENOTE" {
		t.Errorf("code = %v, want CANNOT_RENOTE_TO_A_PURE_RENOTE", errObj["code"])
	}

	// Replying to a quote targets the quote itself.
	reply := createNote(t, ts, tokenA, map[string]any{"replyId": quote, "text": "reply to quote"})
	resp, body = post(t, ts, "/api/notes/show", map[string]any{"noteId": reply})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show reply: status %d, body %v", resp.StatusCode, body)
	}
	if body["replyId"] != quote {
		t.Errorf("replyId = %v, want %v", body["replyId"], quote)
	}
	if parent, ok := body["reply"].(map[string]any); !ok || parent["id"] != quote {
		t.Errorf("embedded reply = %v, want the quote", body["reply"])
	}

	// Renoting a quote targets the quote itself.
	renote := createNote(t, ts, tokenA, map[string]any{"renoteId": quote})
	resp, body = post(t, ts, "/api/notes/show", map[string]any{"noteId": renote})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show renote: status %d, body %v", resp.StatusCode, body)
	}
	if body["renoteId"] != quote {
		t.Errorf("renoteId = %v, want %v", body["renoteId"], quote)
	}
}

func TestSpecifiedVisibility(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	idB, _ := createUser(t, ts, uniqueName(t)+"b")

	id := createNote(t, ts, tokenA, map[string]any{
		"text":           "just for you",
		"visibility":     "specified",
		"visibleUserIds": []string{idB},
	})

	resp, body := post(t, ts, "/api/notes/show", map[string]any{"noteId": id})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("show: status %d", resp.StatusCode)
	}
	visible, ok := body["visibleUserIds"].([]any)
	if !ok || len(visible) != 1 || visible[0] != idB {
		t.Errorf("visibleUserIds = %v, want [%s]", body["visibleUserIds"], idB)
	}
}

// Misskey refuses a reply the replier could not read and refuses to widen
// a specified note's audience
// (core/NoteCreateService.ts L373-393, core/entities/NoteEntityService.ts
// L294-344): a reply target invisible to the replier is
// CANNOT_REPLY_TO_AN_INVISIBLE_NOTE, and replying to a specified note with
// any other visibility is
// CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY.
func TestReplyTargetVisibility(t *testing.T) {
	// The follower case needs following/create, which only runs with the
	// federation store attached.
	ts := newFederationServer(t, "public").ts
	idA, tokenA := createUser(t, ts, uniqueName(t)+"a")
	idB, tokenB := createUser(t, ts, uniqueName(t)+"b")
	_, tokenC := createUser(t, ts, uniqueName(t)+"c")

	// A specified note addressed to B only.
	specified := createNote(t, ts, tokenA, map[string]any{
		"text":           "just for b",
		"visibility":     "specified",
		"visibleUserIds": []string{idB},
	})

	resp, body := post(t, ts, "/api/notes/create", map[string]any{
		"i": tokenC, "replyId": specified, "text": "let me in",
		"visibility": "specified", "visibleUserIds": []string{idA},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reply by unaddressed user: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_REPLY_TO_AN_INVISIBLE_NOTE" {
		t.Errorf("code = %v, want CANNOT_REPLY_TO_AN_INVISIBLE_NOTE", errObj["code"])
	}
	if errObj["id"] != "b98980fa-3780-406c-a935-b6d0eeee10d1" {
		t.Errorf("id = %v, want b98980fa-3780-406c-a935-b6d0eeee10d1", errObj["id"])
	}

	// The addressed user must keep the note specified.
	resp, body = post(t, ts, "/api/notes/create", map[string]any{
		"i": tokenB, "replyId": specified, "text": "public answer",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("widening reply: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ = body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY" {
		t.Errorf("code = %v, want CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY", errObj["code"])
	}
	if errObj["id"] != "ed940410-535c-4d5e-bfa3-af798671e93c" {
		t.Errorf("id = %v, want ed940410-535c-4d5e-bfa3-af798671e93c", errObj["id"])
	}

	// Staying specified is accepted.
	createNote(t, ts, tokenB, map[string]any{
		"replyId": specified, "text": "keeping it private",
		"visibility": "specified", "visibleUserIds": []string{idA},
	})

	// A followers-only note is invisible to a non-follower, visible to a
	// follower.
	followers := createNote(t, ts, tokenA, map[string]any{
		"text": "followers only", "visibility": "followers",
	})
	resp, body = post(t, ts, "/api/notes/create", map[string]any{
		"i": tokenC, "replyId": followers, "text": "not a follower",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reply by non-follower: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ = body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_REPLY_TO_AN_INVISIBLE_NOTE" {
		t.Errorf("non-follower code = %v, want CANNOT_REPLY_TO_AN_INVISIBLE_NOTE", errObj["code"])
	}

	if resp, out := post(t, ts, "/api/following/create", map[string]any{
		"i": tokenC, "userId": idA,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("follow: status %d, body %v", resp.StatusCode, out)
	}
	createNote(t, ts, tokenC, map[string]any{"replyId": followers, "text": "now I can"})

	// The author always sees their own note.
	createNote(t, ts, tokenA, map[string]any{"replyId": followers, "text": "self reply"})

	// A public target stays replyable by anyone.
	open := createNote(t, ts, tokenA, map[string]any{"text": "open book"})
	createNote(t, ts, tokenC, map[string]any{"replyId": open, "text": "sure"})
}

// Misskey rejects a reaction on a pure renote; a quote is reactable like any
// other note (core/ReactionService.ts L115-118, notes/reactions/create.ts
// cannotReactToRenote eaccdc08-ddef-43fe-908f-d108faad57f5).
func TestCannotReactToPureRenote(t *testing.T) {
	ts := newTestServer(t, "open")
	_, tokenA := createUser(t, ts, uniqueName(t)+"a")
	_, tokenB := createUser(t, ts, uniqueName(t)+"b")

	original := createNote(t, ts, tokenA, map[string]any{"text": "original"})
	pure := createNote(t, ts, tokenB, map[string]any{"renoteId": original})

	resp, body := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenA, "noteId": pure, "reaction": "👍",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("react to pure renote: status %d, body %v", resp.StatusCode, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["code"] != "CANNOT_REACT_TO_RENOTE" {
		t.Errorf("code = %v, want CANNOT_REACT_TO_RENOTE", errObj["code"])
	}
	if errObj["id"] != "eaccdc08-ddef-43fe-908f-d108faad57f5" {
		t.Errorf("id = %v, want eaccdc08-ddef-43fe-908f-d108faad57f5", errObj["id"])
	}

	// The original and a quote stay reactable.
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenA, "noteId": original, "reaction": "👍",
	}); resp.StatusCode != http.StatusOK {
		t.Errorf("react to the original: status %d, body %v", resp.StatusCode, out)
	}
	quote := createNote(t, ts, tokenB, map[string]any{"renoteId": original, "text": "a quote"})
	if resp, out := post(t, ts, "/api/notes/reactions/create", map[string]any{
		"i": tokenA, "noteId": quote, "reaction": "👍",
	}); resp.StatusCode != http.StatusOK {
		t.Errorf("react to a quote: status %d, body %v", resp.StatusCode, out)
	}
}

func TestDriveFileLifecycle(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	// Upload through the multipart endpoint.
	resp, body := uploadFile(t, ts.URL, token, "note.txt", bytes.NewBufferString("hello drive"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drive/files/create: status %d, body %v", resp.StatusCode, body)
	}
	fileID, _ := body["id"].(string)
	if fileID == "" {
		t.Fatalf("no file id: %v", body)
	}
	if body["name"] != "note.txt" {
		t.Errorf("name = %v, want note.txt", body["name"])
	}

	resp, body = post(t, ts, "/api/drive/files/show", map[string]any{"fileId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drive/files/show: status %d, body %v", resp.StatusCode, body)
	}

	resp, raw := postRaw(t, ts, "/api/drive/files", map[string]any{"i": token})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drive/files: status %d", resp.StatusCode)
	}
	list, _ := raw.([]any)
	if len(list) != 1 {
		t.Errorf("file list has %d entries, want 1", len(list))
	}

	resp, _ = post(t, ts, "/api/drive/files/delete", map[string]any{"i": token, "fileId": fileID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("drive/files/delete: status %d", resp.StatusCode)
	}
	resp, _ = post(t, ts, "/api/drive/files/show", map[string]any{"fileId": fileID})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("show after delete: status %d, want 404", resp.StatusCode)
	}
}

// uploadFile posts a multipart file to the drive endpoint.
func uploadFile(t *testing.T, baseURL, token, name string, content *bytes.Buffer) (*http.Response, map[string]any) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content.Bytes()); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/drive/files/create", &body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var decoded map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp, decoded
}
