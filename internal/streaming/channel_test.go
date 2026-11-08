package streaming

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// noteFor builds a packed note as the packer would emit it.
func noteFor(id, authorID, visibility string, mutate ...func(map[string]any)) PackedNote {
	n := map[string]any{
		"id":             id,
		"userId":         authorID,
		"visibility":     visibility,
		"text":           "hello",
		"cw":             nil,
		"user":           map[string]any{"id": authorID, "host": nil},
		"fileIds":        []any{},
		"mentions":       []any{},
		"visibleUserIds": []any{},
	}
	for _, m := range mutate {
		m(n)
	}
	return PackedNote(n)
}

// collector records every channel event a channel emits.
//
// Channels write synchronously through Connection.sendChannel into the
// wsConn queue, so draining the queue after HandleNote returns is
// deterministic. No write goroutine runs: the test owns the queue.
type collector struct {
	events []channelEvent
}

type channelEvent struct {
	ID   string
	Type string
	Body any
}

// drain pulls every queued channel event into the collector.
func (col *collector) drain(ws *wsConn) {
	for {
		select {
		case m := <-ws.queue:
			if m.Type != "channel" {
				continue
			}
			b, _ := m.Body.(map[string]any)
			id, _ := b["id"].(string)
			typ, _ := b["type"].(string)
			col.events = append(col.events, channelEvent{ID: id, Type: typ, Body: b["body"]})
		default:
			return
		}
	}
}

// ids returns the note ids the collector recorded.
//
// Channel payloads are PackedNote or map[string]any depending on the
// channel, so the body is normalised through a type switch.
func (col *collector) ids() []string {
	out := make([]string, 0, len(col.events))
	for _, e := range col.events {
		var body map[string]any
		switch t := e.Body.(type) {
		case PackedNote:
			body = t
		case map[string]any:
			body = t
		}
		if id, ok := body["id"].(string); ok {
			out = append(out, id)
			continue
		}
		out = append(out, "")
	}
	return out
}

// harness bundles a connection with its collector.
type harness struct {
	conn *Connection
	ws   *wsConn
	col  *collector
}

// ids drains pending events and returns the recorded note ids.
func (h *harness) ids() []string {
	h.col.drain(h.ws)
	return h.col.ids()
}

// newTestConnection builds a Connection with a fake WebSocket writer.
func newTestConnection(t *testing.T, v Viewer) *harness {
	t.Helper()
	bus := NewBus()
	ws := &wsConn{
		queue: make(chan outbound, 4096),
		done:  make(chan struct{}),
	}
	c := newConnection(t.Context(), ws, bus, v)
	t.Cleanup(func() { ws.close() })
	return &harness{conn: c, ws: ws, col: &collector{}}
}

// newChannel builds a channel of the given kind on the harness.
func newChannel(h *harness, name, id string, params map[string]json.RawMessage) Channel {
	if params == nil {
		params = map[string]json.RawMessage{}
	}
	ch := channelSpecs[name].New(h.conn, id)
	ch.Init(params)
	return ch
}

func TestChannelRegistry(t *testing.T) {
	want := map[string]struct {
		credential bool
		shareable  bool
	}{
		"main":           {true, true},
		"homeTimeline":   {true, false},
		"localTimeline":  {false, false},
		"hybridTimeline": {true, false},
		"globalTimeline": {false, false},
		// userList requires the credential outright; upstream's init
		// dereferences the viewer so anonymous subscribers crash.
		"userList": {true, false},
		// clip accepts anonymous watchers of a public clip.
		"clip": {false, false},
		// antenna is owner-only, as upstream's init existence check.
		"antenna": {true, false},
		// channel boards accept anonymous watchers, no existence check.
		"channel": {false, false},
	}
	for name, w := range want {
		spec, ok := channelSpecs[name]
		if !ok {
			t.Errorf("%s: channel is not registered", name)
			continue
		}
		if spec.RequireCredential != w.credential {
			t.Errorf("%s: requireCredential = %v, want %v", name, spec.RequireCredential, w.credential)
		}
		if spec.Shareable != w.shareable {
			t.Errorf("%s: shareable = %v, want %v", name, spec.Shareable, w.shareable)
		}
	}
	if len(channelSpecs) != len(want) {
		t.Errorf("registry has %d channels, want %d", len(channelSpecs), len(want))
	}
}

func TestMainChannelRejectsAnonymous(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	ch := channelSpecs["main"].New(h.conn, "1")
	if ok := ch.Init(nil); ok {
		t.Fatal("main channel must reject anonymous connections")
	}
}

func TestLocalTimelineFilter(t *testing.T) {
	localPublic := noteFor("n1", "alice", "public")
	localHome := noteFor("n2", "alice", "home")
	remotePublic := noteFor("n3", "bob", "public", func(n map[string]any) {
		n["user"] = map[string]any{"id": "bob", "host": "remote.example"}
	})
	localFollowers := noteFor("n4", "alice", "followers")
	all := []PackedNote{localPublic, localHome, remotePublic, localFollowers}

	t.Run("anonymous", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		ch := newChannel(h, "localTimeline", "c", nil)
		for _, n := range all {
			ch.HandleNote(n)
		}
		if got, want := h.ids(), []string{"n1"}; !reflect.DeepEqual(got, want) {
			t.Errorf("notes = %v, want %v (public local only)", got, want)
		}
	})

	t.Run("signedIn", func(t *testing.T) {
		h := newTestConnection(t, Viewer{UserID: "me"})
		ch := newChannel(h, "localTimeline", "c", nil)
		for _, n := range all {
			ch.HandleNote(n)
		}
		// Signing in does not widen localTimeline.
		if got, want := h.ids(), []string{"n1"}; !reflect.DeepEqual(got, want) {
			t.Errorf("notes = %v, want %v", got, want)
		}
	})
}

func TestGlobalTimelineFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	ch := newChannel(h, "globalTimeline", "g", nil)

	ch.HandleNote(noteFor("n1", "bob", "public", func(n map[string]any) {
		n["user"] = map[string]any{"id": "bob", "host": "remote.example"}
	}))
	ch.HandleNote(noteFor("n2", "alice", "home"))

	if got, want := h.ids(), []string{"n1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("notes = %v, want %v (public only, remote included)", got, want)
	}
}

func TestHomeTimelineFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "me", Following: map[string]Follow{"friend": {}}})
	ch := newChannel(h, "homeTimeline", "h", nil)

	ch.HandleNote(noteFor("mine", "me", "public"))
	ch.HandleNote(noteFor("friend", "friend", "public"))
	ch.HandleNote(noteFor("stranger", "stranger", "public"))

	if got, want := h.ids(), []string{"mine", "friend"}; !reflect.DeepEqual(got, want) {
		t.Errorf("notes = %v, want %v (own + followed only)", got, want)
	}
}

func TestHybridTimelineFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "me", Following: map[string]Follow{"friend": {}}})
	ch := newChannel(h, "hybridTimeline", "hy", nil)

	// Hybrid = followed accounts plus public local notes.
	ch.HandleNote(noteFor("mine", "me", "public"))
	ch.HandleNote(noteFor("followed", "friend", "public"))
	ch.HandleNote(noteFor("localStranger", "stranger", "public"))
	ch.HandleNote(noteFor("remoteStranger", "bob", "public", func(n map[string]any) {
		n["user"] = map[string]any{"id": "bob", "host": "remote.example"}
	}))
	ch.HandleNote(noteFor("followedPrivate", "friend", "followers"))

	if got, want := h.ids(), []string{"mine", "followed", "localStranger", "followedPrivate"}; !reflect.DeepEqual(got, want) {
		t.Errorf("notes = %v, want %v", got, want)
	}
}

func TestWithRenotesFalseDropsPureRenotes(t *testing.T) {
	pureRenote := noteFor("rn", "me", "public", func(n map[string]any) {
		n["renoteId"] = "orig"
		delete(n, "text")
		n["renote"] = map[string]any{"id": "orig", "userId": "friend", "visibility": "public"}
	})
	quote := noteFor("q", "me", "public", func(n map[string]any) {
		n["renoteId"] = "orig"
		n["renote"] = map[string]any{"id": "orig", "userId": "friend", "visibility": "public"}
	})

	t.Run("globalTimeline default false", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		ch := newChannel(h, "globalTimeline", "g", nil)
		ch.HandleNote(pureRenote)
		ch.HandleNote(quote)
		if got, want := h.ids(), []string{"q"}; !reflect.DeepEqual(got, want) {
			t.Errorf("notes = %v, want %v", got, want)
		}
	})

	t.Run("homeTimeline default true", func(t *testing.T) {
		h := newTestConnection(t, Viewer{UserID: "me"})
		ch := newChannel(h, "homeTimeline", "h", nil)
		ch.HandleNote(pureRenote)
		ch.HandleNote(quote)
		if got, want := h.ids(), []string{"rn", "q"}; !reflect.DeepEqual(got, want) {
			t.Errorf("notes = %v, want %v", got, want)
		}
	})

	t.Run("explicit withRenotes true", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		ch := newChannel(h, "globalTimeline", "g", map[string]json.RawMessage{
			"withRenotes": json.RawMessage("true"),
		})
		ch.HandleNote(pureRenote)
		if got, want := h.ids(), []string{"rn"}; !reflect.DeepEqual(got, want) {
			t.Errorf("notes = %v, want %v", got, want)
		}
	})
}

func TestReplyFilter(t *testing.T) {
	v := Viewer{UserID: "me", Following: map[string]Follow{"friend": {}}}

	replyToMe := noteFor("r1", "stranger", "public", func(n map[string]any) {
		n["reply"] = map[string]any{"id": "p", "userId": "me", "visibility": "public"}
	})
	replyToOther := noteFor("r2", "stranger", "public", func(n map[string]any) {
		n["reply"] = map[string]any{"id": "p", "userId": "third", "visibility": "public"}
	})
	followersReply := noteFor("r3", "stranger", "public", func(n map[string]any) {
		n["reply"] = map[string]any{"id": "p", "userId": "third", "visibility": "followers"}
	})

	// globalTimeline has no reply filter at all.
	hg := newTestConnection(t, v)
	chg := newChannel(hg, "globalTimeline", "g", nil)
	for _, n := range []PackedNote{replyToMe, replyToOther, followersReply} {
		chg.HandleNote(n)
	}
	if got, want := hg.ids(), []string{"r1", "r2", "r3"}; !reflect.DeepEqual(got, want) {
		t.Errorf("globalTimeline notes = %v, want %v", got, want)
	}

	// localTimeline applies the strict branch: only replies to me
	// survive, because nobody's follow setting enables replies.
	hl := newTestConnection(t, v)
	chl := newChannel(hl, "localTimeline", "l", nil)
	for _, n := range []PackedNote{replyToMe, replyToOther, followersReply} {
		chl.HandleNote(n)
	}
	if got, want := hl.ids(), []string{"r1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("localTimeline notes = %v, want %v", got, want)
	}

	// With withReplies enabled the loose branch applies: a
	// followers-only reply by an unfollowed author is dropped, a public
	// one passes.
	hl2 := newTestConnection(t, v)
	chl2 := newChannel(hl2, "localTimeline", "l", map[string]json.RawMessage{
		"withReplies": json.RawMessage("true"),
	})
	for _, n := range []PackedNote{replyToMe, replyToOther, followersReply} {
		chl2.HandleNote(n)
	}
	if got, want := hl2.ids(), []string{"r1", "r2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("localTimeline withReplies notes = %v, want %v", got, want)
	}
}

func TestVisibilitySpecified(t *testing.T) {
	note := noteFor("n", "alice", "specified", func(n map[string]any) {
		n["visibleUserIds"] = []any{"me"}
	})

	if isNoteVisibleForMe(note, Viewer{}) {
		t.Error("specified note must not be visible anonymously")
	}
	if !isNoteVisibleForMe(note, Viewer{UserID: "me"}) {
		t.Error("specified note must be visible to a listed user")
	}
	if isNoteVisibleForMe(note, Viewer{UserID: "other"}) {
		t.Error("specified note must not be visible to an unlisted user")
	}
	if !isNoteVisibleForMe(note, Viewer{UserID: "alice"}) {
		t.Error("specified note must be visible to its author")
	}
}

func TestVisibilityFollowers(t *testing.T) {
	note := noteFor("n", "alice", "followers")

	if isNoteVisibleForMe(note, Viewer{}) {
		t.Error("followers note must not be visible anonymously")
	}
	if isNoteVisibleForMe(note, Viewer{UserID: "stranger"}) {
		t.Error("followers note must not be visible to a non-follower")
	}
	if !isNoteVisibleForMe(note, Viewer{UserID: "fan", Following: map[string]Follow{"alice": {}}}) {
		t.Error("followers note must be visible to a follower")
	}
	if !isNoteVisibleForMe(note, Viewer{UserID: "alice"}) {
		t.Error("followers note must be visible to its author")
	}
}

func TestWithFilesFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	ch := newChannel(h, "globalTimeline", "g", map[string]json.RawMessage{
		"withFiles": json.RawMessage("true"),
	})
	ch.HandleNote(noteFor("plain", "alice", "public"))
	ch.HandleNote(noteFor("file", "alice", "public", func(n map[string]any) {
		n["fileIds"] = []any{"f1"}
	}))

	if got, want := h.ids(), []string{"file"}; !reflect.DeepEqual(got, want) {
		t.Errorf("notes = %v, want %v", got, want)
	}
}

// TestConnectionChannelLimits covers the connect/disconnect lifecycle.
func TestConnectionChannelLimits(t *testing.T) {
	t.Run("disconnect removes the channel", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		h.conn.connectChannel("a", "globalTimeline", nil, false)
		if len(h.conn.channels) != 1 {
			t.Fatalf("channels = %d, want 1", len(h.conn.channels))
		}
		h.conn.onDisconnect(json.RawMessage(`{"id":"a"}`))
		if len(h.conn.channels) != 0 {
			t.Errorf("channels = %d, want 0 after disconnect", len(h.conn.channels))
		}
	})

	t.Run("unknown channel is ignored", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		h.conn.connectChannel("a", "noSuchChannel", nil, false)
		if len(h.conn.channels) != 0 {
			t.Errorf("channels = %d, want 0", len(h.conn.channels))
		}
	})

	t.Run("main is deduplicated", func(t *testing.T) {
		h := newTestConnection(t, Viewer{UserID: "me"})
		h.conn.connectChannel("a", "main", nil, false)
		h.conn.connectChannel("b", "main", nil, false)
		if len(h.conn.channels) != 1 {
			t.Errorf("channels = %d, want 1 (main is shareable)", len(h.conn.channels))
		}
	})

	t.Run("cap is 32", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		for i := 0; i < 40; i++ {
			h.conn.connectChannel(string(rune('a'+i)), "globalTimeline", nil, false)
		}
		if got := len(h.conn.channels); got != maxChannelsPerConnection {
			t.Errorf("channels = %d, want %d", got, maxChannelsPerConnection)
		}
	})

	// Reconnecting an id that is already connected replaces that channel
	// rather than consuming another slot. The replacement has to happen
	// before the cap check, otherwise a full connection could never
	// reconnect one of its own channels: the old one is gone and the new
	// one is refused. Upstream Connection.connectChannel does the same.
	t.Run("replacing an id at the cap keeps a channel", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		for i := 0; i < maxChannelsPerConnection; i++ {
			h.conn.connectChannel(string(rune('a'+i)), "globalTimeline", nil, false)
		}
		if got := len(h.conn.channels); got != maxChannelsPerConnection {
			t.Fatalf("channels = %d, want %d", got, maxChannelsPerConnection)
		}

		h.conn.connectChannel("a", "globalTimeline", nil, false)
		if _, ok := h.conn.channels["a"]; !ok {
			t.Fatal("replacing a channel at the cap dropped it")
		}
		if got := len(h.conn.channels); got != maxChannelsPerConnection {
			t.Errorf("channels = %d, want %d", got, maxChannelsPerConnection)
		}
	})

	t.Run("pong replies connected", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		h.conn.connectChannel("a", "globalTimeline", nil, true)
		select {
		case m := <-h.ws.queue:
			if m.Type != "connected" {
				t.Fatalf("message type = %q, want connected", m.Type)
			}
			b, _ := m.Body.(map[string]any)
			if b["id"] != "a" {
				t.Errorf("connected id = %v, want a", b["id"])
			}
		default:
			t.Fatal("no reply was queued")
		}
	})

	t.Run("no pong means no reply", func(t *testing.T) {
		h := newTestConnection(t, Viewer{})
		h.conn.connectChannel("a", "globalTimeline", nil, false)
		select {
		case m := <-h.ws.queue:
			t.Fatalf("unexpected message %q", m.Type)
		default:
		}
	})
}

// TestMainChannelEvents checks mainChannel's pass-through and mention
// visibility filtering.
func TestMainChannelEvents(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "me"})
	ch := newChannel(h, "main", "m", nil)

	note := noteFor("n1", "alice", "public")
	msg, err := NewMessage("mention", note)
	if err != nil {
		t.Fatal(err)
	}
	ch.HandleNoteUpdate("", nil, NoteMeta{})
	h.conn.bus.Publish(MainSubject("me"), msg)

	// A followers-only note from an unfollowed author must not reach the
	// mention handler.
	hidden, _ := NewMessage("mention", noteFor("n2", "stranger", "followers"))
	h.conn.bus.Publish(MainSubject("me"), hidden)

	if got, want := h.ids(), []string{"n1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("mention notes = %v, want %v", got, want)
	}
}

// TestNoteStreamVisibility checks noteUpdated delivery rules.
func TestNoteStreamVisibility(t *testing.T) {
	cases := []struct {
		name       string
		viewer     Viewer
		visibility string
		visible    []string
		want       bool
	}{
		{"public note, anonymous", Viewer{}, "public", nil, true},
		{"specified note, listed", Viewer{UserID: "me"}, "specified", []string{"me"}, true},
		{"specified note, unlisted", Viewer{UserID: "me"}, "specified", []string{"other"}, false},
		{"specified note, anonymous", Viewer{}, "specified", []string{"me"}, false},
		{"followers note, follower", Viewer{UserID: "me", Following: map[string]Follow{"alice": {}}}, "followers", nil, true},
		{"followers note, stranger", Viewer{UserID: "me"}, "followers", nil, false},
		{"own followers note", Viewer{UserID: "alice"}, "followers", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestConnection(t, tc.viewer)
			h.conn.onSubscribeNote(json.RawMessage(`{"id":"n1"}`))

			body, _ := json.Marshal(map[string]any{
				"id":             "n1",
				"userId":         "alice",
				"visibility":     tc.visibility,
				"visibleUserIds": tc.visible,
				"body":           map[string]any{"reaction": "👍"},
			})
			typ := "reacted"
			h.conn.bus.Publish(NoteSubject("n1"), Message{Type: &typ, Body: body})

			var got *outbound
			for {
				select {
				case m := <-h.ws.queue:
					if m.Type == "noteUpdated" {
						got = &m
					}
					continue
				default:
				}
				break
			}
			if tc.want && got == nil {
				t.Fatal("expected a noteUpdated event")
			}
			if !tc.want && got != nil {
				t.Fatalf("unexpected noteUpdated event: %+v", got)
			}
			if got != nil {
				b, _ := got.Body.(map[string]any)
				if b["id"] != "n1" || b["type"] != "reacted" {
					t.Errorf("noteUpdated = %+v", b)
				}
			}
		})
	}
}

// TestNoteSubscriptionRefCount checks subNote/unsubNote counting.
func TestNoteSubscriptionRefCount(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	h.conn.onSubscribeNote(json.RawMessage(`{"id":"n1"}`))
	h.conn.onSubscribeNote(json.RawMessage(`{"id":"n1"}`))
	if got := h.conn.noteSubs["n1"]; got != 2 {
		t.Fatalf("refcount = %d, want 2", got)
	}
	h.conn.onUnsubscribeNote(json.RawMessage(`{"id":"n1"}`))
	if got := h.conn.noteSubs["n1"]; got != 1 {
		t.Fatalf("refcount = %d, want 1", got)
	}
	h.conn.onUnsubscribeNote(json.RawMessage(`{"id":"n1"}`))
	if _, ok := h.conn.noteSubs["n1"]; ok {
		t.Fatal("subscription should be gone")
	}
	if _, ok := h.conn.noteUnsubs["n1"]; ok {
		t.Fatal("bus listener should be released")
	}
}

// TestNoteSubscriptionCap checks LRU eviction at the cap and that the
// listener table never drifts from the subscription table.
func TestNoteSubscriptionCap(t *testing.T) {
	h := newTestConnection(t, Viewer{})

	// Re-subscribe to the same notes repeatedly: counts must rise without
	// leaking extra listeners, and the LRU order must not accumulate
	// stale entries that get evicted as if they were live.
	for round := 0; round < 3; round++ {
		for i := 0; i < maxSubscribingNotesPerConnection; i++ {
			id, _ := json.Marshal(map[string]string{"id": string(rune(0x4e00 + i))})
			h.conn.onSubscribeNote(id)
		}
	}
	if got := len(h.conn.noteSubs); got != maxSubscribingNotesPerConnection {
		t.Fatalf("subscriptions = %d, want %d", got, maxSubscribingNotesPerConnection)
	}
	if got := len(h.conn.noteUnsubs); got != maxSubscribingNotesPerConnection {
		t.Fatalf("bus listeners = %d, want %d (one per subscribed note)", got, maxSubscribingNotesPerConnection)
	}

	// Pushing past the cap evicts one live subscription per new one.
	for i := 0; i < 10; i++ {
		id, _ := json.Marshal(map[string]string{"id": "extra-" + string(rune('a'+i))})
		h.conn.onSubscribeNote(id)
	}
	if got := len(h.conn.noteSubs); got != maxSubscribingNotesPerConnection {
		t.Errorf("subscriptions = %d, want %d", got, maxSubscribingNotesPerConnection)
	}
	if got := len(h.conn.noteUnsubs); got != maxSubscribingNotesPerConnection {
		t.Errorf("bus listeners = %d, want %d", got, maxSubscribingNotesPerConnection)
	}

	// Every subscription must have a live listener.
	for id := range h.conn.noteSubs {
		if _, ok := h.conn.noteUnsubs[id]; !ok {
			t.Fatalf("note %q has no bus listener", id)
		}
	}
	for id := range h.conn.noteUnsubs {
		if _, ok := h.conn.noteSubs[id]; !ok {
			t.Fatalf("listener for %q has no subscription", id)
		}
	}
}

func TestBoolParam(t *testing.T) {
	cases := []struct {
		raw  string
		def  bool
		want bool
	}{
		{`true`, false, true},
		{`false`, true, false},
		{`null`, true, true},
		{`"x"`, false, true},
	}
	for _, tc := range cases {
		got := boolParam(map[string]json.RawMessage{"p": json.RawMessage(tc.raw)}, "p", tc.def)
		if got != tc.want {
			t.Errorf("boolParam(%s, def=%v) = %v, want %v", tc.raw, tc.def, got, tc.want)
		}
	}
	if got := boolParam(nil, "p", true); got != true {
		t.Errorf("missing param should use the default")
	}
}

// stubListLoader is a ListLoader with a fixed membership map.
type stubListLoader struct {
	members map[string]ListMembership
}

func (s stubListLoader) LoadListMembers(_ context.Context, _ string, _ string) (map[string]ListMembership, bool, error) {
	return s.members, true, nil
}

func TestUserListChannelFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice", Following: map[string]Follow{}})
	h.conn.lists = stubListLoader{members: map[string]ListMembership{
		"bob": {},
	}}

	listIDJSON, _ := json.Marshal("list1")
	ch := newChannel(h, "userList", "u1", map[string]json.RawMessage{"listId": listIDJSON})
	if ch.Name() != "userList" {
		t.Fatalf("channel name = %q", ch.Name())
	}

	// A member's note passes; a stranger's does not.
	ch.HandleNote(noteFor("n1", "bob", "public"))
	ch.HandleNote(noteFor("n2", "carol", "public"))

	// The list owner is not a member unless added, so their note stays out.
	ch.HandleNote(noteFor("n3", "alice", "public"))

	var col collector
	col.drain(h.ws)
	if len(col.events) != 1 || col.events[0].Type != "note" {
		t.Fatalf("events = %+v, want one note", col.events)
	}
}

// An anonymous connection's userList subscription is refused, matching
// the credential requirement every other personal channel has.
func TestChannelRejectsAnonymousUserList(t *testing.T) {
	h := newTestConnection(t, Viewer{})
	listIDJSON, _ := json.Marshal("list1")
	ch := channelSpecs["userList"].New(h.conn, "u2")
	if ch.Init(map[string]json.RawMessage{"listId": listIDJSON}) {
		t.Error("anonymous subscription must be rejected")
	}
}

// A missing listId param and an unknown loader both reject the
// subscription.
func TestUserListChannelRejectsBadParams(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice"})
	if channelSpecs["userList"].New(h.conn, "u1").Init(nil) {
		t.Error("missing listId must reject")
	}
	h2 := newTestConnection(t, Viewer{UserID: "alice"})
	// No loader installed: Init must refuse rather than let the filter
	// pass every note.
	listIDJSON, _ := json.Marshal("list1")
	if channelSpecs["userList"].New(h2.conn, "u2").Init(map[string]json.RawMessage{"listId": listIDJSON}) {
		t.Error("missing loader must reject")
	}
}

// stubClipLoader is a ClipLoader with a fixed visibility answer.
type stubClipLoader struct {
	visible bool
}

func (s stubClipLoader) ClipVisible(_ context.Context, _, _ string) (bool, error) {
	return s.visible, nil
}

func TestClipChannelForwardsEvents(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice"})
	h.conn.clipLoader = stubClipLoader{visible: true}

	clipIDJSON, _ := json.Marshal("clip1")
	ch := newChannel(h, "clip", "c1", map[string]json.RawMessage{"clipId": clipIDJSON})
	if ch.Name() != "clip" {
		t.Fatalf("channel name = %q", ch.Name())
	}

	forwarded, _ := NewMessage("updated", nil)
	h.conn.bus.Publish(ClipSubject("clip1"), forwarded)
	deleted, _ := NewMessage("deleted", nil)
	h.conn.bus.Publish(ClipSubject("clip1"), deleted)
	// Other subjects and types must not surface.
	h.conn.bus.Publish(ClipSubject("other"), forwarded)
	notePing, _ := NewMessage("unrelated", nil)
	h.conn.bus.Publish(ClipSubject("clip1"), notePing)

	var col collector
	col.drain(h.ws)
	if len(col.events) != 2 {
		t.Fatalf("events = %+v, want updated and deleted", col.events)
	}
	if col.events[0].Type != "updated" || col.events[1].Type != "deleted" {
		t.Errorf("event types = %q, %q", col.events[0].Type, col.events[1].Type)
	}
}

func TestClipChannelRejects(t *testing.T) {
	// An invisible clip rejects the subscription.
	h := newTestConnection(t, Viewer{UserID: "alice"})
	h.conn.clipLoader = stubClipLoader{visible: false}
	clipIDJSON, _ := json.Marshal("clip1")
	if channelSpecs["clip"].New(h.conn, "c1").Init(map[string]json.RawMessage{"clipId": clipIDJSON}) {
		t.Error("invisible clip must reject")
	}

	// A missing clipId param rejects.
	h2 := newTestConnection(t, Viewer{UserID: "alice"})
	h2.conn.clipLoader = stubClipLoader{visible: true}
	if channelSpecs["clip"].New(h2.conn, "c2").Init(nil) {
		t.Error("missing clipId must reject")
	}

	// A missing loader rejects rather than letting events flow.
	h3 := newTestConnection(t, Viewer{UserID: "alice"})
	if channelSpecs["clip"].New(h3.conn, "c3").Init(map[string]json.RawMessage{"clipId": clipIDJSON}) {
		t.Error("missing loader must reject")
	}
}

// stubAntennaLoader is an AntennaLoader with a fixed ownership answer.
type stubAntennaLoader struct {
	owned bool
}

func (s stubAntennaLoader) AntennaOwned(_ context.Context, _, _ string) (bool, error) {
	return s.owned, nil
}

func TestAntennaChannelForwardsNotes(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice"})
	h.conn.antennaLoader = stubAntennaLoader{owned: true}

	antennaIDJSON, _ := json.Marshal("ant1")
	ch := newChannel(h, "antenna", "a1", map[string]json.RawMessage{"antennaId": antennaIDJSON})
	if ch.Name() != "antenna" {
		t.Fatalf("channel name = %q", ch.Name())
	}

	packed := noteFor("n1", "bob", "public")
	noteMsg, _ := NewMessage("note", packed)
	h.conn.bus.Publish(AntennaSubject("ant1"), noteMsg)
	// Foreign subjects must not surface.
	h.conn.bus.Publish(AntennaSubject("other"), noteMsg)

	var col collector
	col.drain(h.ws)
	if len(col.events) != 1 || col.events[0].Type != "note" {
		t.Fatalf("events = %+v, want one note", col.events)
	}
}

func TestAntennaChannelRejects(t *testing.T) {
	// A foreign antenna rejects the subscription.
	h := newTestConnection(t, Viewer{UserID: "alice"})
	h.conn.antennaLoader = stubAntennaLoader{owned: false}
	antennaIDJSON, _ := json.Marshal("ant1")
	if channelSpecs["antenna"].New(h.conn, "a1").Init(map[string]json.RawMessage{"antennaId": antennaIDJSON}) {
		t.Error("foreign antenna must reject")
	}

	// A missing loader rejects rather than letting events flow.
	h2 := newTestConnection(t, Viewer{UserID: "alice"})
	if channelSpecs["antenna"].New(h2.conn, "a2").Init(map[string]json.RawMessage{"antennaId": antennaIDJSON}) {
		t.Error("missing loader must reject")
	}

	// A missing antennaId param rejects.
	h3 := newTestConnection(t, Viewer{UserID: "alice"})
	h3.conn.antennaLoader = stubAntennaLoader{owned: true}
	if channelSpecs["antenna"].New(h3.conn, "a3").Init(nil) {
		t.Error("missing antennaId must reject")
	}
}

func TestChannelBoardStreamFilter(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice", Following: map[string]Follow{}})

	boardID, _ := json.Marshal("board1")
	ch := newChannel(h, "channel", "b1", map[string]json.RawMessage{"channelId": boardID})
	if ch.Name() != "channel" {
		t.Fatalf("channel name = %q", ch.Name())
	}

	// A note of the board passes; another board's does not.
	in := noteFor("n1", "bob", "public", func(n map[string]any) {
		n["channelId"] = "board1"
		n["channel"] = map[string]any{"id": "board1", "name": "board"}
	})
	out := noteFor("n2", "bob", "public", func(n map[string]any) {
		n["channelId"] = "board2"
	})
	ch.HandleNote(in)
	ch.HandleNote(out)
	// An ordinary note has no channel and never enters a board.
	ch.HandleNote(noteFor("n3", "bob", "public"))

	var col collector
	col.drain(h.ws)
	if len(col.events) != 1 || col.events[0].Type != "note" {
		t.Fatalf("events = %+v, want one note", col.events)
	}
}

func TestChannelBoardStreamRejects(t *testing.T) {
	h := newTestConnection(t, Viewer{UserID: "alice"})
	if channelSpecs["channel"].New(h.conn, "b1").Init(nil) {
		t.Error("missing channelId must reject")
	}
	h2 := newTestConnection(t, Viewer{UserID: "alice"})
	bad, _ := json.Marshal(42)
	if channelSpecs["channel"].New(h2.conn, "b2").Init(map[string]json.RawMessage{"channelId": bad}) {
		t.Error("non-string channelId must reject")
	}
}
