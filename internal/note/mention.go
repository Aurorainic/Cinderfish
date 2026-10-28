package note

import (
	"regexp"
	"strings"
)

// Mentions
//
// Cinderfish stores note text as MFM and needs the local users a note
// mentions so it can create mention notifications and address the note
// when it federates. Until now notes.Mentions was always empty, which
// left the Streaming API's mention event and the remote audience without
// a source; this file closes that gap.

var (
	// mentionRE matches @user and @user@host. The local part follows
	// Misskey's username rules; the host part is a dotted name.
	// The leading group anchors the @ to a word start so an email address
	// such as a@b.example is not read as a mention of b.example.
	mentionRE = regexp.MustCompile(`(^|[^A-Za-z0-9_@])@([A-Za-z0-9_]+)(?:@([A-Za-z0-9][A-Za-z0-9.\-]*[A-Za-z0-9]))?`)
)

// Mention is one @-reference in a note.
type Mention struct {
	Username string
	// Host is empty for a local mention.
	Host string
}

// ParseMentions extracts the @-references from note text.
//
// Duplicates are collapsed and the order of first appearance is kept,
// which is what makes the resulting notification order stable. A mention
// inside a code span is still extracted: MFM does not render code as
// plain text either, and Misskey resolves mentions from the same input.
func ParseMentions(text string) []Mention {
	if text == "" {
		return nil
	}
	matches := mentionRE.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[Mention]bool, len(matches))
	out := make([]Mention, 0, len(matches))
	for _, m := range matches {
		mention := Mention{Username: m[2], Host: strings.ToLower(m[3])}
		if seen[mention] {
			continue
		}
		seen[mention] = true
		out = append(out, mention)
	}
	return out
}
