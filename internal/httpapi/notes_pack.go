package httpapi

import (
	"context"
	"strings"
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
	// ReactionEmojiURLs maps custom reaction shortcodes to their images.
	ReactionEmojiURLs map[string]string
	// Channel is the packed channel reference of a channel note.
	Channel map[string]any
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
		"updatedAt":      nilOrTime(n.UpdatedAt),
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
		"reactionEmojis": packReactionEmojis(pc.ReactionEmojiURLs),
		"emojis":         packNoteEmojis(n.Emojis, pc.ReactionEmojiURLs),
		"fileIds":        n.FileIDs,
		"files":          files,
		"replyId":        n.ReplyID,
		"renoteId":       n.RenoteID,
		"mentions":       n.Mentions,
		"tags":           n.Tags,
		"uri":            n.URI,
		"url":            n.URL,
		"channelId":      n.ChannelID,
	}
	// The embedded channel reference travels with the note, as upstream's
	// NoteEntityService attaches it.
	packed["channel"] = pc.Channel

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
			if visible, verr := s.notes.VisibleTo(ctx, parent, pc.ViewerID); verr == nil && visible {
				reply := packNote(parent, s.notePackContext(ctx, parent, pc.ViewerID))
				// The embedded reply is a shallow copy.
				reply["replyId"] = nil
				reply["renoteId"] = nil
				packed["reply"] = reply
			} else {
				// A parent the viewer may not read is omitted, like a
				// deleted or unreachable one upstream
				// (core/entities/NotificationEntityService.ts): replyId
				// stays, the embedded object is null.
				packed["reply"] = nil
			}
		} else {
			packed["reply"] = nil
		}
	}
	if n.RenoteID != nil {
		if target, err := s.notes.Get(ctx, *n.RenoteID); err == nil {
			if visible, verr := s.notes.VisibleTo(ctx, target, pc.ViewerID); verr == nil && visible {
				packed["renote"] = packNote(target, s.notePackContext(ctx, target, pc.ViewerID))
			} else {
				packed["renote"] = nil
			}
		} else {
			packed["renote"] = nil
		}
	}

	if n.PollID != nil {
		if poll, err := s.notes.Poll(ctx, n.ID); err == nil {
			counts, _ := s.notes.VoteCounts(ctx, n.ID)
			myVote := -1
			voted := map[int]bool{}
			if pc.ViewerID != "" {
				myVote, _ = s.notes.MyVote(ctx, n.ID, pc.ViewerID)
				if mine, err := s.notes.MyVotes(ctx, n.ID, pc.ViewerID); err == nil {
					for _, c := range mine {
						voted[c] = true
					}
				}
			}
			choices := make([]any, 0, len(poll.Choices))
			for i, text := range poll.Choices {
				choices = append(choices, map[string]any{
					"text":    text,
					"votes":   counts[i],
					"isVoted": i == myVote || voted[i],
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

	if n.ChannelID != nil && s.accounts != nil {
		if ch, err := s.accounts.GetChannel(ctx, *n.ChannelID); err == nil {
			pc.Channel = packChannelRef(ch)
		}
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

	// Custom reactions carry their own image; the renderer needs it because a
	// remote emoji is not in the local registry.
	if s.emojis != nil {
		names := make([]string, 0, len(pc.Reactions))
		for reaction := range pc.Reactions {
			name, _ := reactionShortcode(reaction)
			if name != "" {
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			if urls, err := s.emojis.LookupAll(ctx, names); err == nil && len(urls) > 0 {
				pc.ReactionEmojiURLs = urls
			}
		}
	}

	return pc
}

// unixMilli converts a millisecond timestamp to a time.
func unixMilli(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// packNoteEmojis renders the custom emoji a note references. The renderer
// looks a shortcode up here before falling back to the instance registry, so
// a remote note keeps its own emoji images.
func packNoteEmojis(emojis []note.Emoji, reactionEmojis map[string]string) []any {
	out := make([]any, 0, len(emojis))
	seen := make(map[string]bool, len(emojis))
	for _, e := range emojis {
		out = append(out, map[string]any{"name": e.Name, "url": e.URL})
		seen[e.Name] = true
	}
	// The reaction views hand the note's emoji list to the icon renderer, so
	// custom reactions are merged in when the text does not already use them.
	for name, url := range reactionEmojis {
		if seen[name] {
			continue
		}
		out = append(out, map[string]any{"name": name, "url": url})
	}
	return out
}

// packReactionEmojis renders note.reactionEmojis as a name to URL map.
func packReactionEmojis(urls map[string]string) map[string]any {
	out := make(map[string]any, len(urls))
	for name, url := range urls {
		out[name] = url
	}
	return out
}

// reactionShortcode extracts the name from :name: or :name@host:.
func reactionShortcode(reaction string) (string, string) {
	if !strings.HasPrefix(reaction, ":") || !strings.HasSuffix(reaction, ":") || len(reaction) < 3 {
		return "", ""
	}
	inner := strings.Trim(reaction, ":")
	name, host, _ := strings.Cut(inner, "@")
	return name, host
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

// nilOrTime renders an optional timestamp as RFC3339Nano, or JSON null.
func nilOrTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
