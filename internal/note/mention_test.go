package note

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseMentions(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []Mention
	}{
		{"none", "hello world", nil},
		{"local", "hi @alice", []Mention{{Username: "alice"}}},
		{"remote", "hi @bob@other.example", []Mention{{Username: "bob", Host: "other.example"}}},
		{"mixed", "@alice and @bob@remote.example", []Mention{
			{Username: "alice"},
			{Username: "bob", Host: "remote.example"},
		}},
		{"duplicates collapse", "@alice @alice", []Mention{{Username: "alice"}}},
		// An email address is not a mention: the @ is glued to a word.
		{"email is not a mention", "mail me at a@b.example", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseMentions(c.text)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("ParseMentions(%q) = %#v, want %#v", c.text, got, c.want)
			}
		})
	}
}

// BenchmarkParseMentions measures the parse cost on the largest note the
// service accepts, so the mention limit can be judged against a worst case
// the text length allows rather than a guess.
func BenchmarkParseMentions(b *testing.B) {
	noMention := strings.Repeat("hello world ", MaxTextLength/12)

	var mentions strings.Builder
	for i := 0; mentions.Len() < MaxTextLength; i++ {
		fmt.Fprintf(&mentions, "@user%d@remote.example ", i)
	}

	cases := map[string]string{
		"no-mentions":  noMention[:MaxTextLength],
		"dense-remote": mentions.String()[:MaxTextLength],
	}
	for name, text := range cases {
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			for i := 0; i < b.N; i++ {
				ParseMentions(text)
			}
		})
	}
}
