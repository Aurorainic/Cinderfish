package federation

import (
	"html"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"
)

// NoteDataFromAP converts a received ActivityPub note into the data the
// store persists. The author is taken from the actor that delivered the
// activity, which the caller has already checked against attributedTo.
//
// Visibility is derived from the audience exactly the way Misskey's
// ApAudienceService does: a to/cc containing the Public collection is
// public; otherwise followers-only when the audience is the author's
// followers collection, and specified when it is a list of users.
func NoteDataFromAP(note Object, actor *RemoteUser, r *Renderer) RemoteNote {
	poll := pollOf(note)
	text := note.String("_misskey_content")
	if text == "" {
		if src := note.Object("source"); src != nil {
			text = src.String("content")
		}
	}
	if text == "" {
		text = TextFromHTML(note.String("content"))
	}

	tags := make([]string, 0, 4)
	emojis := make([]EmojiRef, 0, 2)
	seenEmoji := map[string]bool{}
	for _, tag := range note.Objects("tag") {
		switch {
		case tag.IsType("Hashtag"):
			if name := tag.String("name"); name != "" {
				tags = append(tags, strings.TrimPrefix(name, "#"))
			}
		case tag.IsType("Emoji"):
			name := strings.Trim(tag.String("name"), ":")
			url := tag.Object("icon").String("url")
			if url == "" {
				url = tag.String("href")
			}
			if name == "" || url == "" || seenEmoji[name] {
				continue
			}
			seenEmoji[name] = true
			emojis = append(emojis, EmojiRef{Name: name, URL: url})
		}
	}

	attachments := make([]RemoteAttachment, 0)
	for _, attachment := range note.Objects("attachment") {
		if !attachment.IsType("Image") && !attachment.IsType("Document") && !attachment.IsType("Video") && !attachment.IsType("Audio") {
			continue
		}
		fileURL := attachment.Ref("url")
		if fileURL == "" {
			fileURL = attachment.Ref("href")
		}
		parsed, err := url.Parse(fileURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			continue
		}
		name := path.Base(parsed.Path)
		if name == "." || name == "/" || name == "" {
			name = "remote-media"
		}
		contentType := attachment.String("mediaType")
		if contentType == "" {
			contentType = attachment.String("mimeType")
		}
		attachments = append(attachments, RemoteAttachment{
			URL: fileURL, Name: name, ContentType: contentType, Sensitive: note.Bool("sensitive"),
		})
		if len(attachments) == 16 {
			break
		}
	}

	visibility := "public"
	to := note.Refs("to")
	cc := note.Refs("cc")
	if !containsRef(append(to, cc...), Public) {
		followers := actor.URI + "/followers"
		if containsRef(to, followers) {
			visibility = "followers"
		} else {
			visibility = "specified"
		}
	}

	return RemoteNote{
		APID:          note.Ref("id"),
		UserID:        actor.ID,
		UserHost:      actor.Host,
		Visibility:    visibility,
		Text:          text,
		CW:            note.String("summary"),
		ReplyAPID:     note.Ref("inReplyTo"),
		QuoteAPID:     quoteOf(note),
		Tags:          tags,
		Emojis:        emojis,
		Sensitive:     note.Bool("sensitive"),
		Attachments:   attachments,
		CreatedAt:     publishedOrNow(note),
		PollChoices:   poll.PollChoices,
		PollMultiple:  poll.PollMultiple,
		PollExpiresAt: poll.PollExpiresAt,
	}
}

// pollOf extracts a Misskey/Mastodon Question: its choices, whether it allows
// several answers, and the closing time. A Question with fewer than two usable
// choices is stored as a plain note instead of an unusable poll.
func pollOf(note Object) RemoteNote {
	if !note.IsType("Question") {
		return RemoteNote{}
	}
	items := note.Objects("oneOf")
	multiple := false
	if len(items) == 0 {
		items = note.Objects("anyOf")
		multiple = len(items) > 0
	}
	choices := make([]string, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.String("name"))
		if name != "" {
			choices = append(choices, name)
		}
	}
	// Misskey renders at most ten choices; trimming here keeps a hostile
	// peer from storing a thousand-row poll.
	if len(choices) > maxRemotePollChoices {
		choices = choices[:maxRemotePollChoices]
	}
	if len(choices) < 2 {
		return RemoteNote{}
	}
	var expiresAt *time.Time
	if end := note.String("endTime"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			expiresAt = &t
		} else if t, err := time.Parse(time.RFC3339Nano, end); err == nil {
			expiresAt = &t
		}
	}
	return RemoteNote{PollChoices: choices, PollMultiple: multiple, PollExpiresAt: expiresAt}
}

// maxRemotePollChoices mirrors Misskey's ten-choice limit.
const maxRemotePollChoices = 10

// quoteOf returns the quoted note URL, preferring Misskey's own
// extension over the Mastodon-compatible one.
func quoteOf(note Object) string {
	if q := note.String("_misskey_quote"); q != "" {
		return q
	}
	return note.String("quoteUrl")
}

func containsRef(refs []string, want string) bool {
	for _, r := range refs {
		if r == want {
			return true
		}
	}
	return false
}

// TextFromHTML converts common ActivityPub HTML blocks to readable text while
// retaining paragraph and line breaks for clients that provide no source.
func TextFromHTML(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '<' {
			end := strings.IndexByte(s[i:], '>')
			if end < 0 {
				break
			}
			tag := strings.ToLower(strings.TrimSpace(s[i+1 : i+end]))
			closing := strings.HasPrefix(tag, "/")
			if closing {
				tag = strings.TrimPrefix(tag, "/")
			}
			fields := strings.Fields(strings.TrimSuffix(tag, "/"))
			if len(fields) == 0 {
				i += end + 1
				continue
			}
			tag = fields[0]
			if closing && (tag == "p" || tag == "div" || tag == "li") || !closing && (tag == "br" || tag == "p" || tag == "div" || tag == "li") {
				b.WriteString("\n")
			}
			i += end + 1
			continue
		}
		r := rune(s[i])
		if r >= 0x80 {
			decoded, size := utf8.DecodeRuneInString(s[i:])
			r = decoded
			i += size
		} else {
			i++
		}
		b.WriteRune(r)
	}
	out := html.UnescapeString(b.String())
	lines := strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
