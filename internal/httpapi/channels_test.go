package httpapi_test

import (
	"net/http"
	"testing"
)

// The channels/* family plus the channel notes path of notes/create, per
// packages/backend/src/server/api/endpoints/channels/.
//
// A channel note is forced public and local-only, joins the timelines of
// the channel's followers, and stays out of the local/global/list
// timelines.

func TestChannelCRUDAndDefaults(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	// create: color omitted takes the DB default; allowRenoteToExternal
	// defaults true.
	resp, ch := postWithToken(t, f.ts, "/api/channels/create", aliceToken, map[string]any{"name": " general "})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, ch)
	}
	channelID, _ := ch["id"].(string)
	if channelID == "" {
		t.Fatalf("create returned no id: %v", ch)
	}
	if ch["name"] != " general " || ch["color"] != "#86b300" {
		t.Errorf("created channel = %v", ch)
	}
	if ch["isArchived"] != false || ch["notesCount"] != float64(0) || ch["usersCount"] != float64(0) {
		t.Errorf("defaults wrong in %v", ch)
	}

	// banner must be one of the caller's drive files.
	resp, out := postWithToken(t, f.ts, "/api/channels/create", aliceToken, map[string]any{
		"name": "banner", "bannerId": "missing0000000000",
	})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_FILE" {
		t.Fatalf("banner create: %d %v", resp.StatusCode, out)
	}

	// show is public; the viewer flags travel for signed-in callers.
	resp, out = postWithToken(t, f.ts, "/api/channels/show", aliceToken, map[string]any{"channelId": channelID})
	if resp.StatusCode != http.StatusOK || out["isFollowing"] != false {
		t.Fatalf("owner show: %d %v", resp.StatusCode, out)
	}
	if _, ok := out["pinnedNotes"].([]any); !ok {
		t.Errorf("show without pinnedNotes: %v", out)
	}

	// another user cannot edit; the owner can archive, which is the only
	// deletion.
	// An existing channel answers ACCESS_DENIED for a non-owner; an
	// unknown one answers NO_SUCH_CHANNEL.
	resp, out = postWithToken(t, f.ts, "/api/channels/update", bobToken, map[string]any{
		"channelId": channelID, "name": "hijacked",
	})
	if resp.StatusCode != http.StatusForbidden || errCode(out) != "ACCESS_DENIED" {
		t.Fatalf("foreign update: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/channels/update", bobToken, map[string]any{
		"channelId": "missing0000000000", "name": "ghost",
	})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_CHANNEL" {
		t.Fatalf("unknown update: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/channels/update", aliceToken, map[string]any{
		"channelId": channelID, "color": "#ff0000",
	})
	if resp.StatusCode != http.StatusOK || out["color"] != "#ff0000" {
		t.Fatalf("owner update: %d %v", resp.StatusCode, out)
	}

	// follow twice is ALREADY_FOLLOWING; unfollow is idempotent.
	if code := postNoContentToken(t, f.ts, "/api/channels/follow", bobToken, map[string]any{"channelId": channelID}); code != http.StatusNoContent {
		t.Fatalf("follow: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/channels/follow", bobToken, map[string]any{"channelId": channelID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_FOLLOWING" {
		t.Fatalf("double follow: %d %v", resp.StatusCode, out)
	}
	if got := postListWithToken(t, f.ts, "/api/channels/followed", bobToken, nil); len(got) != 1 {
		t.Errorf("followed = %d, want 1", len(got))
	}
	if code := postNoContentToken(t, f.ts, "/api/channels/unfollow", bobToken, map[string]any{"channelId": channelID}); code != http.StatusNoContent {
		t.Fatalf("unfollow: %d", code)
	}
	if got := postListWithToken(t, f.ts, "/api/channels/followed", bobToken, nil); len(got) != 0 {
		t.Errorf("followed after unfollow = %d, want 0", len(got))
	}

	// mute bookkeeping rejects a past expiry and a second mute:
	resp, out = postWithToken(t, f.ts, "/api/channels/mute/create", bobToken, map[string]any{
		"channelId": channelID, "expiresAt": 1000,
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "EXPIRES_AT_IS_PAST" {
		t.Fatalf("past expiry: %d %v", resp.StatusCode, out)
	}
	if code := postNoContentToken(t, f.ts, "/api/channels/mute/create", bobToken, map[string]any{"channelId": channelID}); code != http.StatusNoContent {
		t.Fatalf("mute: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/channels/mute/create", bobToken, map[string]any{"channelId": channelID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_MUTING_CHANNEL" {
		t.Fatalf("double mute: %d %v", resp.StatusCode, out)
	}
	if got := postListWithToken(t, f.ts, "/api/channels/mute/list", bobToken, nil); len(got) != 1 {
		t.Errorf("mute/list = %d, want 1", len(got))
	}
	if code := postNoContentToken(t, f.ts, "/api/channels/mute/delete", bobToken, map[string]any{"channelId": channelID}); code != http.StatusNoContent {
		t.Fatalf("unmute: %d", code)
	}

	// archiving hides the channel from search and owned, but show still
	// returns it.
	if code := postNoContentToken(t, f.ts, "/api/channels/update", aliceToken, map[string]any{"channelId": channelID, "isArchived": true}); code != http.StatusOK {
		t.Fatalf("archive via update must return the channel: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/channels/show", aliceToken, map[string]any{"channelId": channelID})
	if resp.StatusCode != http.StatusOK || out["isArchived"] != true {
		t.Fatalf("archived show: %d %v", resp.StatusCode, out)
	}
	if got := postListWithToken(t, f.ts, "/api/channels/owned", aliceToken, nil); len(got) != 0 {
		t.Errorf("archived channel in owned = %d, want 0", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/channels/search", bobToken, map[string]any{"query": "general"}); len(got) != 0 {
		t.Errorf("archived channel in search = %d, want 0", len(got))
	}
}

func TestChannelNotesAndTimelines(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, _ := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	resp, ch := postWithToken(t, f.ts, "/api/channels/create", aliceToken, map[string]any{"name": "board"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, ch)
	}
	channelID, _ := ch["id"].(string)

	// bob follows the board and posts into it.
	if code := postNoContentToken(t, f.ts, "/api/channels/follow", bobToken, map[string]any{"channelId": channelID}); code != http.StatusNoContent {
		t.Fatalf("follow: %d", code)
	}
	resp, created := postWithToken(t, f.ts, "/api/notes/create", bobToken, map[string]any{
		"text": "in the board", "channelId": channelID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("channel note: %d %v", resp.StatusCode, created)
	}
	noteID, _ := created["createdNote"].(map[string]any)["id"].(string)
	createdNote, _ := created["createdNote"].(map[string]any)
	// The overrides are visible: public visibility, localOnly true.
	if createdNote["visibility"] != "public" || createdNote["localOnly"] != true {
		t.Errorf("channel note not forced public/local: %v", createdNote)
	}
	if createdNote["channelId"] != channelID {
		t.Errorf("channel note channelId = %v", createdNote["channelId"])
	}

	// The channel timeline shows it; the local timeline does not.
	notes := postListWithToken(t, f.ts, "/api/channels/timeline", aliceToken, map[string]any{"channelId": channelID})
	if len(notes) != 1 {
		t.Fatalf("channels/timeline = %d, want 1", len(notes))
	}
	notes = postListWithToken(t, f.ts, "/api/notes/local-timeline", aliceToken, nil)
	for _, item := range notes {
		if n, _ := item.(map[string]any); n["id"] == noteID {
			t.Errorf("channel note leaked into the local timeline")
		}
	}

	// Home: only a channel follower sees it. Alice follows nothing.
	home := postListWithToken(t, f.ts, "/api/notes/timeline", aliceToken, map[string]any{})
	for _, item := range home {
		if n, _ := item.(map[string]any); n["id"] == noteID {
			t.Errorf("channel note reached a non-follower's home timeline")
		}
	}
	home = postListWithToken(t, f.ts, "/api/notes/timeline", bobToken, map[string]any{})
	found := false
	for _, item := range home {
		if n, _ := item.(map[string]any); n["id"] == noteID {
			found = true
		}
	}
	if !found {
		t.Errorf("channel note missing from the follower's home timeline")
	}

	// The channel counters moved.
	resp, out := postWithToken(t, f.ts, "/api/channels/show", aliceToken, map[string]any{"channelId": channelID})
	if out["notesCount"] != float64(1) || out["usersCount"] != float64(1) {
		t.Errorf("counters = %v/%v, want 1/1", out["notesCount"], out["usersCount"])
	}
	if out["isFollowing"] != false {
		t.Errorf("alice isFollowing = %v, want false", out["isFollowing"])
	}

	// Renoting a channel note out of a channel that forbids it fails; the
	// default channel allows it.
	resp, created = postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{
		"renoteId": noteID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default renote out: %d %v", resp.StatusCode, created)
	}

	// featured lists the channel after it received notes.
	if got := postListWithToken(t, f.ts, "/api/channels/featured", aliceToken, nil); len(got) != 1 {
		t.Errorf("featured = %d, want 1", len(got))
	}
	// search with an empty query lists every non-archived channel.
	if got := postListWithToken(t, f.ts, "/api/channels/search", aliceToken, map[string]any{"query": ""}); len(got) != 1 {
		t.Errorf("search empty query = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/channels/search", aliceToken, map[string]any{"query": "board"}); len(got) != 1 {
		t.Errorf("search by name = %d, want 1", len(got))
	}

	// A reply joins the parent's channel.
	resp, created = postWithToken(t, f.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "reply in board", "replyId": noteID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("reply: %d %v", resp.StatusCode, created)
	}
	reply := created["createdNote"].(map[string]any)
	if reply["channelId"] != channelID {
		t.Errorf("reply channelId = %v, want the parent's channel", reply["channelId"])
	}
	notes = postListWithToken(t, f.ts, "/api/channels/timeline", aliceToken, map[string]any{"channelId": channelID})
	if len(notes) != 2 {
		t.Errorf("channels/timeline after reply = %d, want 2", len(notes))
	}
}
