package streaming

import (
	"encoding/json"
)

// newHomeTimelineChannel builds the homeTimeline channel.
func newHomeTimelineChannel(c *Connection, id string) Channel {
	return &homeTimelineChannel{timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// homeTimelineChannel carries notes from the accounts the viewer
// follows, plus the viewer's own notes. Misskey models this by
// subscribing every connection to notesStream and filtering here.
type homeTimelineChannel struct {
	timelineChannel
}

func (h *homeTimelineChannel) Name() string { return "homeTimeline" }

func (h *homeTimelineChannel) Init(params map[string]json.RawMessage) bool {
	if h.conn.viewerSnapshot().UserID == "" {
		return false
	}
	h.withRenotes = boolParam(params, "withRenotes", true)
	h.withFiles = boolParam(params, "withFiles", false)
	// homeTimeline has no withReplies param: replies are controlled per
	// follow through the follow setting.
	h.withReplies = false
	h.filter = h.onNote
	return true
}

func (h *homeTimelineChannel) onNote(note PackedNote) {
	v := h.viewer()
	authorID := noteAuthorID(note)
	isMe := authorID == v.UserID

	if !h.filesAllowed(note) {
		return
	}

	// A channel post is only visible to followers of that channel, which
	// MilnFish does not model yet.
	if hasChannelID(note) {
		return
	}
	if !isMe && !v.follows(authorID) {
		return
	}

	if !isNoteVisibleForMe(note, v) {
		return
	}
	if !h.replyAllowed(note, v.followsWithReplies(authorID)) {
		return
	}
	if !h.renoteAllowed(note) {
		return
	}
	if h.isNoteMutedOrBlocked(note) {
		return
	}

	h.emitNote(note)
}

// newLocalTimelineChannel builds the localTimeline channel.
func newLocalTimelineChannel(c *Connection, id string) Channel {
	return &localTimelineChannel{timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// localTimelineChannel carries public notes by local authors. It is the
// one timeline that accepts anonymous connections.
type localTimelineChannel struct {
	timelineChannel
}

func (l *localTimelineChannel) Name() string { return "localTimeline" }

func (l *localTimelineChannel) Init(params map[string]json.RawMessage) bool {
	l.withRenotes = boolParam(params, "withRenotes", false)
	l.withReplies = boolParam(params, "withReplies", false)
	l.withFiles = boolParam(params, "withFiles", false)
	l.filter = l.onNote
	// Misskey checks the ltlAvailable user policy here. MilnFish grants
	// it unconditionally; per-user policies arrive with roles.
	return true
}

func (l *localTimelineChannel) onNote(note PackedNote) {
	if !l.filesAllowed(note) {
		return
	}

	// Local author, public visibility, no channel: upstream's filter.
	if noteAuthorHost(note) != "" {
		return
	}
	if visibilityOf(note) != "public" {
		return
	}
	if hasChannelID(note) {
		return
	}

	v := l.viewer()

	// The reply filter only narrows signed-in viewers, as upstream. It
	// uses the note's per-follow setting when the viewer follows its
	// author, and the channel's withReplies param otherwise.
	if v.UserID != "" {
		if _, isReply := replyOf(note); isReply {
			if !l.replyAllowed(note, v.followsWithReplies(noteAuthorID(note))) {
				return
			}
		}
	}

	if !l.renoteAllowed(note) {
		return
	}
	if l.isNoteMutedOrBlocked(note) {
		return
	}

	l.emitNote(note)
}

// newGlobalTimelineChannel builds the globalTimeline channel.
func newGlobalTimelineChannel(c *Connection, id string) Channel {
	return &globalTimelineChannel{timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// globalTimelineChannel carries every public note the instance knows
// about, remote or local.
type globalTimelineChannel struct {
	timelineChannel
}

func (g *globalTimelineChannel) Name() string { return "globalTimeline" }

func (g *globalTimelineChannel) Init(params map[string]json.RawMessage) bool {
	g.withRenotes = boolParam(params, "withRenotes", false)
	g.withFiles = boolParam(params, "withFiles", false)
	g.filter = g.onNote
	return true
}

func (g *globalTimelineChannel) onNote(note PackedNote) {
	if !g.filesAllowed(note) {
		return
	}
	if visibilityOf(note) != "public" {
		return
	}
	if hasChannelID(note) {
		return
	}
	if !g.renoteAllowed(note) {
		return
	}
	if g.isNoteMutedOrBlocked(note) {
		return
	}
	g.emitNote(note)
}

// newHybridTimelineChannel builds the hybridTimeline channel.
func newHybridTimelineChannel(c *Connection, id string) Channel {
	return &hybridTimelineChannel{timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// hybridTimelineChannel merges the home and local timelines: the notes
// of followed accounts plus every public local note.
type hybridTimelineChannel struct {
	timelineChannel
}

func (h *hybridTimelineChannel) Name() string { return "hybridTimeline" }

func (h *hybridTimelineChannel) Init(params map[string]json.RawMessage) bool {
	if h.conn.viewerSnapshot().UserID == "" {
		return false
	}
	h.withRenotes = boolParam(params, "withRenotes", false)
	h.withReplies = boolParam(params, "withReplies", false)
	h.withFiles = boolParam(params, "withFiles", false)
	h.filter = h.onNote
	return true
}

func (h *hybridTimelineChannel) onNote(note PackedNote) {
	v := h.viewer()
	authorID := noteAuthorID(note)
	isMe := authorID == v.UserID

	if !h.filesAllowed(note) {
		return
	}

	// Channel posts require following the channel, not modelled yet.
	if hasChannelID(note) {
		return
	}

	// Visible when it is mine, by someone I follow, or a public local
	// note.
	if !(isMe || v.follows(authorID) || (noteAuthorHost(note) == "" && visibilityOf(note) == "public")) {
		return
	}

	if !isNoteVisibleForMe(note, v) {
		return
	}
	if h.isNoteMutedOrBlocked(note) {
		return
	}
	if !h.replyAllowed(note, v.followsWithReplies(authorID)) {
		return
	}
	if !h.renoteAllowed(note) {
		return
	}

	h.emitNote(note)
}

// noteAuthorID returns the id of the note's author.
func noteAuthorID(note PackedNote) string {
	id, _ := note["userId"].(string)
	return id
}
