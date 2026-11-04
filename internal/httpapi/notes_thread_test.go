package httpapi_test

import (
	"net/http"
	"testing"
)

// TestNotesConversationWalksAncestors pins notes/conversation: the reply chain
// above a note comes back nearest-first, offset skips from the near end, and
// the walk stops at a parent the viewer may not read.
func TestNotesConversationWalksAncestors(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))

	root := createNote(t, ts, aliceToken, map[string]any{"text": "root"})
	mid := createNote(t, ts, aliceToken, map[string]any{"text": "mid", "replyId": root})
	leaf := createNote(t, ts, aliceToken, map[string]any{"text": "leaf", "replyId": mid})

	conversation := postListWithToken(t, ts, "/api/notes/conversation", aliceToken, map[string]any{"noteId": leaf})
	if len(conversation) != 2 {
		t.Fatalf("conversation = %v, want root and mid", conversation)
	}
	first, _ := conversation[0].(map[string]any)
	second, _ := conversation[1].(map[string]any)
	if first["id"] != mid || second["id"] != root {
		t.Errorf("conversation order = %v, want mid then root", conversation)
	}

	// offset skips the nearest ancestor.
	skipped := postListWithToken(t, ts, "/api/notes/conversation", aliceToken, map[string]any{"noteId": leaf, "offset": 1})
	if len(skipped) != 1 {
		t.Fatalf("offset page = %v, want only the root", skipped)
	}
	only, _ := skipped[0].(map[string]any)
	if only["id"] != root {
		t.Errorf("offset page = %v, want %s", only, root)
	}

	// limit caps the walk.
	capped := postListWithToken(t, ts, "/api/notes/conversation", aliceToken, map[string]any{"noteId": leaf, "limit": 1})
	if len(capped) != 1 {
		t.Errorf("limit page = %v, want one", capped)
	}
}

// TestNotesConversationStopsAtPrivateParent proves the visibility stop: a
// followers-only parent is not leaked to an anonymous visitor through a
// public reply.
func TestNotesConversationStopsAtPrivateParent(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))

	hidden := createNote(t, ts, aliceToken, map[string]any{
		"text": "followers only", "visibility": "followers",
	})
	reply := createNote(t, ts, aliceToken, map[string]any{"text": "public reply", "replyId": hidden})

	anonymous := postListWithToken(t, ts, "/api/notes/conversation", "", map[string]any{"noteId": reply})
	if len(anonymous) != 0 {
		t.Errorf("anonymous conversation = %v, want the chain to stop", anonymous)
	}

	own := postListWithToken(t, ts, "/api/notes/conversation", aliceToken, map[string]any{"noteId": reply})
	if len(own) != 1 {
		t.Errorf("owner conversation = %v, want the private parent", own)
	}

	if resp, out := post(t, ts, "/api/notes/conversation", map[string]any{"noteId": "00000000000000000000"}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown note = %d %v, want 404", resp.StatusCode, out)
	}
}

// TestNotesChildren pins notes/children: replies to the note come back newest
// first, visibility-filtered, and the cursors paginate.
func TestNotesChildren(t *testing.T) {
	ts := newTestServer(t, "open")
	aliceID, aliceToken := createUser(t, ts, uniqueName(t))
	bobID, bobToken := createUser(t, ts, uniqueName(t)+"b")
	_, carolToken := createUser(t, ts, uniqueName(t)+"c")

	parent := createNote(t, ts, aliceToken, map[string]any{"text": "parent"})
	firstReply := createNote(t, ts, bobToken, map[string]any{"text": "first reply", "replyId": parent})
	secondReply := createNote(t, ts, bobToken, map[string]any{"text": "second reply", "replyId": parent})
	privateReply := createNote(t, ts, bobToken, map[string]any{
		"text": "followers only reply", "replyId": parent, "visibility": "followers",
	})

	// A follower-only reply to alice's note is readable by alice (she is the
	// user it answers) and by its author bob, but not by an outsider.
	all := postListWithToken(t, ts, "/api/notes/children", aliceToken, map[string]any{"noteId": parent})
	if len(all) != 3 {
		t.Fatalf("children = %v, want all three for the parent author", all)
	}
	newest, _ := all[0].(map[string]any)
	oldest, _ := all[2].(map[string]any)
	if newest["id"] != privateReply || oldest["id"] != firstReply {
		t.Errorf("children order = %v, want newest first", all)
	}

	if own := postListWithToken(t, ts, "/api/notes/children", bobToken, map[string]any{"noteId": parent}); len(own) != 3 {
		t.Errorf("bob's children = %v, want all three", own)
	}
	if outsider := postListWithToken(t, ts, "/api/notes/children", carolToken, map[string]any{"noteId": parent}); len(outsider) != 2 {
		t.Errorf("carol's children = %v, want only the two public replies", outsider)
	}

	// The untilId cursor pages back from the newest reply.
	page := postListWithToken(t, ts, "/api/notes/children", aliceToken, map[string]any{
		"noteId": parent, "limit": 1, "untilId": secondReply,
	})
	if len(page) != 1 {
		t.Fatalf("cursor page = %v, want one", page)
	}
	next, _ := page[0].(map[string]any)
	if next["id"] != firstReply {
		t.Errorf("cursor page = %v, want %s", next, firstReply)
	}
	_ = aliceID
	_ = bobID
}
