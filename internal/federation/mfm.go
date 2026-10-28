package federation

import (
	"html"
	"regexp"
	"strings"
)

// MilnFish stores note text as MFM (Misskey Flavored Markdown) and has no
// stored HTML yet, but ActivityPub's content field must be HTML. This file
// renders the MFM subset the federation stream actually uses.
//
// A full mfm-js port is a large dependency for what a personal instance
// needs; what peers observe is that content is valid HTML with links and
// mentions marked up. Unsupported syntax is escaped rather than dropped,
// so nothing a user typed disappears from the federated copy.

var (
	// mentionRE matches a local or remote mention: @user or @user@host.
	mentionRE = regexp.MustCompile("@(" + userPat + ")(?:@(" + hostPat + "))?")
	// emojiRE matches a custom emoji shortcode: :name:.
	emojiRE = regexp.MustCompile(":([A-Za-z0-9_+\\-]+):")
	// urlRE matches a bare http(s) URL.
	urlRE = regexp.MustCompile("https?://[^\\s<>\"\x27]+")
	// hashtagRE matches a hashtag not glued to a preceding word.
	hashtagRE = regexp.MustCompile("(^|[\\s(])#([^\\s#]+)")
	// codeBlockRE matches triple-backtick fenced code.
	codeBlockRE = regexp.MustCompile("(?s)`{3}([^\\n]*)\\n(.*?)`{3}")
	// inlineCodeRE matches single-backtick inline code.
	inlineCodeRE = regexp.MustCompile("`([^`\\n]+)`")
	// boldRE matches **bold**.
	boldRE = regexp.MustCompile("\\*\\*([^*\\n]+)\\*\\*")
	// italicRE matches _italic_ delimited by whitespace or the ends.
	italicRE = regexp.MustCompile("(^|[\\s])_([^_\\n]+)_([\\s]|$)")
	// newlineRE normalises CRLF and CR to LF.
	newlineRE = strings.NewReplacer("\r\n", "\n", "\r", "\n")
)

// userPat is the shape of a local part in a mention.
const userPat = "[A-Za-z0-9_]+"

// hostPat is the shape of the host part of a remote mention; it must
// allow dots, since a host is a dotted name.
const hostPat = "[A-Za-z0-9][A-Za-z0-9.-]*[A-Za-z0-9]"

// MFMToHTML renders a subset of Misskey Flavored Markdown as HTML.
//
// Supported: mentions, hashtags, custom emoji shortcodes, bare URLs,
// **bold**, _italic_, inline code and fenced code blocks. Line breaks
// become <br>. Everything else is HTML-escaped.
//
// The order of operations matters: code spans are extracted first so
// their contents are never scanned for mentions or URLs, then the whole
// string is escaped, and only then are markers turned into tags.
func MFMToHTML(mfm string) string {
	if mfm == "" {
		return ""
	}
	s := newlineRE.Replace(mfm)

	var codeBlocks, inlineCodes []string
	s = codeBlockRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := codeBlockRE.FindStringSubmatch(m)
		class := ""
		if lang := strings.TrimSpace(sub[1]); lang != "" {
			class = " class=\"language-" + html.EscapeString(lang) + "\""
		}
		codeBlocks = append(codeBlocks, "<pre><code"+class+">"+html.EscapeString(sub[2])+"</code></pre>")
		return "\x00CODE" + itoa(len(codeBlocks)-1) + "\x00"
	})

	s = inlineCodeRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := inlineCodeRE.FindStringSubmatch(m)
		inlineCodes = append(inlineCodes, "<code>"+html.EscapeString(sub[1])+"</code>")
		return "\x00INLINE" + itoa(len(inlineCodes)-1) + "\x00"
	})

	out := html.EscapeString(s)

	out = urlRE.ReplaceAllStringFunc(out, func(u string) string {
		return "<a href=\"" + u + "\" rel=\"nofollow noopener\" target=\"_blank\">" + u + "</a>"
	})

	out = emojiRE.ReplaceAllString(out, ":<span class=\"emoji\">$1</span>:")

	out = mentionRE.ReplaceAllStringFunc(out, func(m string) string {
		sub := mentionRE.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		user, host := sub[1], sub[2]
		if host != "" {
			return "<a href=\"https://" + host + "/@" + user + "\" class=\"u-url mention\">@" + user + "@" + host + "</a>"
		}
		return "<a href=\"/@" + user + "\" class=\"u-url mention\">@" + user + "</a>"
	})

	out = hashtagRE.ReplaceAllStringFunc(out, func(m string) string {
		sub := hashtagRE.FindStringSubmatch(m)
		if len(sub) < 3 {
			return m
		}
		return sub[1] + "<a href=\"/tags/" + sub[2] + "\" rel=\"tag\">#" + sub[2] + "</a>"
	})

	out = boldRE.ReplaceAllString(out, "<b>$1</b>")
	out = italicRE.ReplaceAllString(out, "$1<i>$2</i>$3")

	out = strings.ReplaceAll(out, "\n", "<br>\n")

	for i, code := range inlineCodes {
		out = strings.ReplaceAll(out, "\x00INLINE"+itoa(i)+"\x00", code)
	}
	for i, code := range codeBlocks {
		out = strings.ReplaceAll(out, "\x00CODE"+itoa(i)+"\x00", code)
	}

	return out
}

// PlainText strips MFM markup down to plain text.
func PlainText(mfm string) string {
	s := newlineRE.Replace(mfm)
	s = strings.NewReplacer("**", "", "`", "", "__", "").Replace(s)
	return strings.TrimSpace(s)
}

// itoa formats a small non-negative int; the placeholders only ever
// hold list indices so pulling in strconv is not worth it.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte("0123456789"[n%10])
		n /= 10
	}
	return string(buf[i:])
}
