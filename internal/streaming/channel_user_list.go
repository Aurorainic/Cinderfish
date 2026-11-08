package streaming

import (
	"encoding/json"
)

// newUserListChannel builds the userList channel.
func newUserListChannel(c *Connection, id string) Channel {
	return &userListChannel{timelineChannel: timelineChannel{BaseChannel: BaseChannel{id: id, conn: c}}}
}

// userListChannel carries the notes of one user list's members, plus the
// list's membership changes. Misskey models this by subscribing every
// connection to notesStream and filtering against the list's membership;
// this channel does the same and keeps its member map fresh from the
// userListStream subject's userAdded/userRemoved events rather than
// upstream's five-second poll.
type userListChannel struct {
	timelineChannel
	listID  string
	members map[string]ListMembership
}

func (u *userListChannel) Name() string { return "userList" }

func (u *userListChannel) Init(params map[string]json.RawMessage) bool {
	v := u.conn.viewerSnapshot()
	if v.UserID == "" {
		return false
	}
	raw, ok := params["listId"]
	if !ok || len(raw) == 0 {
		return false
	}
	var listID string
	if err := json.Unmarshal(raw, &listID); err != nil || listID == "" {
		return false
	}
	u.listID = listID

	if !u.loadMembers() {
		return false
	}

	// Upstream's userList channel defaults withRenotes to true.
	u.withRenotes = boolParam(params, "withRenotes", true)
	u.withFiles = boolParam(params, "withFiles", false)
	u.filter = u.onNote

	// Membership changes reload the map; upstream polls every five
	// seconds instead, but a single-process instance can be exact.
	u.subscribeToBus(UserListSubject(listID), u.onListEvent)
	return true
}

// loadMembers refreshes the channel's member map. It returns false when
// the list is missing or not owned by the viewer, which rejects the
// subscription.
func (u *userListChannel) loadMembers() bool {
	if u.conn.lists == nil {
		return false
	}
	v := u.conn.viewerSnapshot()
	members, ok, err := u.conn.lists.LoadListMembers(u.conn.ctx, u.listID, v.UserID)
	if err != nil || !ok {
		return false
	}
	u.members = members
	return true
}

// onListEvent handles userAdded/userRemoved on the list's subject: it
// reloads the member map and forwards the event to the client, which is
// what Misskey's channel does with the same events.
func (u *userListChannel) onListEvent(_ string, msg Message) {
	if msg.Type == nil {
		return
	}
	switch *msg.Type {
	case "userAdded", "userRemoved":
		u.loadMembers()
		u.sendRaw(*msg.Type, msg.Body)
	}
}

// sendRaw forwards an event whose body is already encoded JSON.
func (u *userListChannel) sendRaw(typ string, body json.RawMessage) {
	u.conn.ws.send("channel", map[string]any{"id": u.id, "type": typ, "body": body})
}

func (u *userListChannel) onNote(note PackedNote) {
	v := u.viewer()

	if !u.filesAllowed(note) {
		return
	}
	// Channel posts are only visible to followers of that channel, which
	// Cinderfish does not model yet.
	if hasChannelID(note) {
		return
	}

	authorID := noteAuthorID(note)
	membership, isMember := u.members[authorID]
	if !isMember {
		return
	}
	if !isNoteVisibleForMe(note, v) {
		return
	}
	if !u.replyAllowed(note, membership.WithReplies) {
		return
	}
	if !u.renoteAllowed(note) {
		return
	}
	if u.isNoteMutedOrBlocked(note) {
		return
	}

	u.emitNote(note)
}
