package note

import (
	"reflect"
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
