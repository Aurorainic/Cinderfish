package streaming

import (
	"context"
	"encoding/json"
	"sync"
)

// PackedNote is a note already rendered in Misskey's Note schema.
//
// Rendering happens in httpapi, which owns the packers; the streaming
// layer only forwards the maps it receives. That keeps the dependency
// direction httpapi -> streaming.
type PackedNote map[string]any

// Follow records one edge of the viewer's follow graph.
type Follow struct {
	// WithReplies is the viewer's per-follow "show replies" setting.
	WithReplies bool
}

// Viewer is the identity behind a stream connection.
//
// Anonymous connections have an empty UserID: localTimeline and
// globalTimeline accept them, the other channels do not.
type Viewer struct {
	UserID string
	// Following maps followee id to the viewer's follow settings.
	Following map[string]Follow
	// FollowingChannels lists the ids of the channels the viewer follows,
	// which decide whether a channel note may enter home/hybrid.
	FollowingChannels map[string]bool
}

// follows reports whether the viewer follows the given user.
func (v Viewer) follows(userID string) bool {
	_, ok := v.Following[userID]
	return ok
}

// followsChannel reports whether the viewer follows the given channel.
func (v Viewer) followsChannel(channelID string) bool {
	return v.FollowingChannels[channelID]
}

// followsWithReplies reports the viewer's "show replies" setting for the
// given user, false when the viewer does not follow them.
func (v Viewer) followsWithReplies(userID string) bool {
	f, ok := v.Following[userID]
	return ok && f.WithReplies
}

// ChannelFollowLoader loads the viewer's followed channels, which the
// home and hybrid timelines need to admit channel notes.
type ChannelFollowLoader interface {
	// LoadFollowedChannels returns the channel ids userID follows.
	LoadFollowedChannels(ctx context.Context, userID string) (map[string]bool, error)
}

// Channel is one subscription on a connection.
type Channel interface {
	// Name is the channel name clients send in "connect".
	Name() string
	// Init receives the client params. Returning false rejects the
	// subscription; the channel is then detached silently, as upstream.
	Init(params map[string]json.RawMessage) bool
	// HandleNote receives every note published on notesStream.
	HandleNote(note PackedNote)
	// HandleNoteUpdate receives events published for one specific note
	// (reactions, deletions, votes) from subNote subscriptions.
	HandleNoteUpdate(typ string, body json.RawMessage, meta NoteMeta)
	// Dispose releases subscriptions.
	Dispose()
}

// NoteMeta is the visibility envelope Misskey attaches to noteStream
// events.
type NoteMeta struct {
	ID             string
	UserID         string
	Visibility     string
	VisibleUserIDs []string
}

// BaseChannel implements the parts every channel shares.
type BaseChannel struct {
	id      string
	conn    *Connection
	unsubs  []func()
	cleanup []func()
	mu      sync.Mutex
}

// ID returns the client-supplied connection id of this channel.
func (b *BaseChannel) ID() string { return b.id }

// Connection returns the owning connection.
func (b *BaseChannel) Connection() *Connection { return b.conn }

// subscribeToBus registers a handler on the bus.
func (b *BaseChannel) subscribeToBus(subject string, h func(subject string, msg Message)) {
	cancel := b.conn.bus.Subscribe(func(s string, m Message) {
		if s == subject {
			h(s, m)
		}
	})
	b.unsubs = append(b.unsubs, cancel)
}

// Dispose cancels the bus subscriptions of the channel.
func (b *BaseChannel) Dispose() {
	b.mu.Lock()
	unsubs := b.unsubs
	b.unsubs = nil
	cleanup := b.cleanup
	b.cleanup = nil
	b.mu.Unlock()

	for _, u := range unsubs {
		u()
	}
	for _, f := range cleanup {
		f()
	}
}

// send emits a channel event.
func (b *BaseChannel) send(typ string, body any) {
	b.conn.sendChannel(b.id, typ, body)
}

// viewer returns the connection's viewer.
func (b *BaseChannel) viewer() Viewer { return b.conn.viewerSnapshot() }

// isNoteVisibleForMe reimplements Misskey's Channel.isNoteVisibleForMe.
// It must stay in sync with note.TimelineService's visibility filter.
func isNoteVisibleForMe(note PackedNote, v Viewer) bool {
	me := v.UserID
	authorID, _ := note["userId"].(string)

	visibility, _ := note["visibility"].(string)
	switch visibility {
	case "specified":
		if me == "" {
			return false
		}
		if me == authorID {
			return true
		}
		return containsString(stringSlice(note["visibleUserIds"]), me)

	case "followers":
		if me == "" {
			return false
		}
		if me == authorID {
			return true
		}
		if reply, ok := note["reply"].(map[string]any); ok {
			if rid, _ := reply["userId"].(string); rid != "" && rid == me {
				return true
			}
		}
		if containsString(stringSlice(note["mentions"]), me) {
			return true
		}
		return v.follows(authorID)

	default: // public, home
		return true
	}
}

// noteAuthorHost returns the host of the note's author, empty for local.
func noteAuthorHost(note PackedNote) string {
	user, ok := note["user"].(map[string]any)
	if !ok {
		return ""
	}
	host, _ := user["host"].(string)
	return host
}

// isPureRenote reports whether the note is a renote without its own
// content, i.e. not a quote.
func isPureRenote(note PackedNote) bool {
	renoteID, hasRenote := note["renoteId"]
	if !hasRenote || renoteID == nil {
		return false
	}
	// A pure renote carries no text of its own. Quote renotes always do.
	text, _ := note["text"].(string)
	cw, _ := note["cw"].(string)
	return text == "" && cw == ""
}

// replyOf returns the embedded reply note, if any.
func replyOf(note PackedNote) (map[string]any, bool) {
	reply, ok := note["reply"].(map[string]any)
	if !ok || reply == nil {
		return nil, false
	}
	return reply, true
}

// hasFiles reports whether the note carries at least one file.
func hasFiles(note PackedNote) bool {
	return len(stringSlice(note["fileIds"])) > 0
}

// stringSlice coerces a JSON array value into a string slice.
func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// boolParam reads a boolean channel parameter with a default.
func boolParam(params map[string]json.RawMessage, name string, def bool) bool {
	raw, ok := params[name]
	if !ok || len(raw) == 0 {
		return def
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		// Misskey uses !!(params.x ?? default), so a non-boolean truthy
		// value still enables the flag.
		if v == nil {
			return def
		}
		return true
	}
	return b
}
