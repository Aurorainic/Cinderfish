package federation

import (
	"strings"
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
	text := note.String("_misskey_content")
	if text == "" {
		if src := note.Object("source"); src != nil {
			text = src.String("content")
		}
	}
	if text == "" {
		// Fall back to the HTML body stripped of tags, so a Mastodon
		// note still produces readable text.
		text = StripHTML(note.String("content"))
	}

	tags := make([]string, 0, 4)
	for _, tag := range note.Objects("tag") {
		if tag.IsType("Hashtag") {
			if name := tag.String("name"); name != "" {
				tags = append(tags, strings.TrimPrefix(name, "#"))
			}
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
		APID:       note.Ref("id"),
		UserID:     actor.ID,
		UserHost:   actor.Host,
		Visibility: visibility,
		Text:       text,
		CW:         note.String("summary"),
		ReplyAPID:  note.Ref("inReplyTo"),
		QuoteAPID:  quoteOf(note),
		Tags:       tags,
		Sensitive:  note.Bool("sensitive"),
		CreatedAt:  publishedOrNow(note),
	}
}

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

// StripHTML removes tags from an HTML fragment, collapsing whitespace.
// It is deliberately simple: the content field of a remote note is only
// used as a text fallback.
func StripHTML(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch {
		case r == '<':
			depth++
		case r == '>':
			if depth > 0 {
				depth--
			}
		case depth == 0:
			b.WriteRune(r)
		}
	}
	out := b.String()
	out = strings.ReplaceAll(out, "&lt;", "<")
	out = strings.ReplaceAll(out, "&gt;", ">")
	out = strings.ReplaceAll(out, "&quot;", "\"")
	out = strings.ReplaceAll(out, "&#39;", "'")
	out = strings.ReplaceAll(out, "&amp;", "&")
	for _, br := range []string{"<br>", "<br/>", "<br />", "</p>"} {
		_ = br
	}
	out = strings.ReplaceAll(out, "\r\n", "\n")
	fields := strings.Fields(out)
	return strings.Join(fields, " ")
}
