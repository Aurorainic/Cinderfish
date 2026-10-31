package note

// IsPureRenote mirrors Misskey's isRenote(note) && !isQuote(note)
// (packages/backend/src/misc/is-renote.ts): a note carries a renote target
// and none of its own content — text, CW, a reply pointer, a poll or files.
// A note WITH any of those is a quote (isQuote), which stays replyable and
// renotable: NoteCreateService.ts rejects `isRenote && !isQuote` targets
// only (L335 for renotes, L384 for replies). The stored row must be loaded
// with its poll id and file ids (noteSelect) for the classification to be
// complete.
func (n *Note) IsPureRenote() bool {
	if n.RenoteID == nil {
		return false
	}
	if n.Text != nil || n.CW != nil || n.PollID != nil || n.ReplyID != nil || len(n.FileIDs) > 0 {
		return false
	}
	return true
}
