package note

import (
	"regexp"
	"strings"
)

// hashtagPattern is Misskey's extractHashtags regex
// (packages/backend/src/misc/extract-hashtags.ts), rewritten without the
// lookbehind RE2 does not support: group 1 is the required preceding
// character, group 2 the tag body.
var hashtagPattern = regexp.MustCompile(`(^|[\s\S])#([^\s#$.,:;!?()\[\]{}'"“”‘’]+)`)

// ExtractHashtags returns the hashtags of a note body, in first-seen order
// and deduplicated case-insensitively. Misskey stores them on the note and
// feeds them to its hashtag statistics; Cinderfish derives hashtags/trend
// from the stored column the same way.
func ExtractHashtags(text string) []string {
	if !strings.Contains(text, "#") {
		return []string{}
	}

	matches := hashtagPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		tag := text[m[4]:m[5]]
		// A body that is only punctuation is not a hashtag.
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}
