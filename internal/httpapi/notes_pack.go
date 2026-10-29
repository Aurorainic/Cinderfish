package httpapi

import (
	"context"
	"time"

	"github.com/Aurorainic/cinderfish/internal/media"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// notePackContext carries the data the packer needs besides the note
// itself, so the packer does not have to reach back into the database
// for every nested entity.
type notePackContext struct {
	ViewerID string
	// VisibleUserIDs is populated for specified-visibility notes.
	VisibleUserIDs []string
	// Author is the packed UserLite of the note's author.
	Author map[string]any
	// Reactions counts per emoji and their sum.
	Reactions     map[string]int
	ReactionTotal int
	// Files are the drive files referenced by the note's FileIDs.
	Files []*media.File
}

// packNote renders a note in Misskey's Note schema.
func packNote(n *note.Note, pc notePackContext) map[string]any {
	files := make([]any, 0, len(n.FileIDs))
	if len(n.FileIDs) > 0 {
		byID := make(map[string]*media.File, len(pc.Files))
		for _, f := range pc.Files {
			byID[f.ID] = f
		}
		for _, id := range n.FileIDs {
			if f := byID[id]; f != nil {
				files = append(files, packDriveFile(f))
			}
		}
	}

	packed := map[string]any{
		"id":             n.ID,
		"createdAt":      n.CreatedAt.UTC().Format(time.RFC3339Nano),
		"userId":         n.UserID,
		"user":           pc.Author,
		"text":           n.Text,
		"cw":             n.CW,
		"visibility":     n.Visibility,
		"localOnly":      n.LocalOnly,
		"renoteCount":    n.RenoteCount,
		"repliesCount":   n.RepliesCount,
		"reactionCount":  pc.ReactionTotal,
		"reactions":      pc.Reactions,
		"reactionEmojis": []any{},
		"emojis":         []any{},
		"fileIds":        n.FileIDs,
		"files":          files,
		"replyId":        n.ReplyID,
		"renoteId":       n.RenoteID,
		"mentions":       n.Mentions,
		"tags":           n.Tags,
		"uri":            n.URI,
		"url":            n.URL,
	}

	if n.Visibility == note.VisibilitySpecified {
		packed["visibleUserIds"] = pc.VisibleUserIDs
	}
	if n.ReactionAcceptance != nil {
		packed["reactionAcceptance"] = *n.ReactionAcceptance
	}
	return packed
}

// packNoteDetail adds the detail-only fields of the Note schema: the
// embedded reply and renote, the poll and the viewer's own reaction.
func packNoteDetail(ctx context.Context, s *Server, n *note.Note, pc notePackContext) map[string]any {
	packed := packNote(n, pc)
	packed["clippedCount"] = n.ClippedCount

	if n.ReplyID != nil {
		if parent, err := s.notes.Get(ctx, *n.ReplyID); err == nil {
			reply := packNote(parent, s.notePackContext(ctx, parent, pc.ViewerID))
			// The embedded reply is a shallow copy.
			reply["replyId"] = nil
			reply["renoteId"] = nil
			packed["reply"] = reply
		} else {
			packed["reply"] = nil
		}
	}
	if n.RenoteID != nil {
		if target, err := s.notes.Get(ctx, *n.RenoteID); err == nil {
			packed["renote"] = packNote(target, s.notePackContext(ctx, target, pc.ViewerID))
		} else {
			packed["renote"] = nil
		}
	}

	if n.PollID != nil {
		if poll, err := s.notes.Poll(ctx, n.ID); err == nil {
			counts, _ := s.notes.VoteCounts(ctx, n.ID)
			myVote := -1
			if pc.ViewerID != "" {
				myVote, _ = s.notes.MyVote(ctx, n.ID, pc.ViewerID)
			}
			choices := make([]any, 0, len(poll.Choices))
			for i, text := range poll.Choices {
				choices = append(choices, map[string]any{
					"text":    text,
					"votes":   counts[i],
					"isVoted": i == myVote,
				})
			}
			packed["hasPoll"] = true
			packed["poll"] = map[string]any{
				"multiple":   poll.Multiple,
				"expiresAt":  pollExpires(poll),
				"choices":    choices,
				"votesCount": totalVotes(counts),
			}
		}
	}

	if pc.ViewerID != "" {
		if emoji, err := s.noteReactions.MyReaction(ctx, n.ID, pc.ViewerID); err == nil && emoji != "" {
			packed["myReaction"] = emoji
		}
	}

	return packed
}

// notePackContext gathers the extra data needed to pack a note.
func (s *Server) notePackContext(ctx context.Context, n *note.Note, viewerID string) notePackContext {
	pc := notePackContext{ViewerID: viewerID}

	if author, err := s.accounts.GetByID(ctx, n.UserID); err == nil {
		pc.Author = packUserLite(author, "", loadProfile(ctx, s, n.UserID))
	} else {
		pc.Author = map[string]any{"id": n.UserID}
	}

	if n.Visibility == note.VisibilitySpecified {
		pc.VisibleUserIDs = s.notes.VisibleUserIDs(ctx, n.ID)
	}

	if len(n.FileIDs) > 0 && s.drive != nil {
		if files, err := s.drive.GetMany(ctx, n.FileIDs); err == nil {
			pc.Files = files
		}
	}

	if counts, err := s.noteReactions.Counts(ctx, n.ID); err == nil {
		pc.Reactions = counts
		for _, c := range counts {
			pc.ReactionTotal += c
		}
	} else {
		pc.Reactions = map[string]int{}
	}

	return pc
}

// unixMilli converts a millisecond timestamp to a time.
func unixMilli(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// pollExpires renders the poll expiry as a millisecond timestamp.
func pollExpires(p *note.Poll) any {
	if p.ExpiresAt == nil {
		return nil
	}
	return p.ExpiresAt.UnixMilli()
}

func totalVotes(counts map[int]int) int {
	total := 0
	for _, n := range counts {
		total += n
	}
	return total
}
