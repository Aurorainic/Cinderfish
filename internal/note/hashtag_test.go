package note

import (
	"reflect"
	"testing"
)

// TestExtractHashtags pins the Misskey-compatible extraction: a tag needs a
// preceding character, trailing punctuation ends it, duplicates collapse and
// the stored form keeps the original case.
func TestExtractHashtags(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{"none", "no tags here", []string{}},
		{"simple", "hello #world", []string{"world"}},
		{"start of line", "#first post", []string{"first"}},
		{"two tags", "#one and #two", []string{"one", "two"}},
		{"case keeps first", "#Rust and #rust", []string{"Rust"}},
		{"punctuation ends", "look #tag, then more", []string{"tag"}},
		{"sentence end", "end of #sentence.", []string{"sentence"}},
		{"hash without body", "just # punctuation", []string{}},
		{"url fragment", "https://example.com/page#anchor", []string{"anchor"}},
		{"japanese tag", "これは #タグ です", []string{"タグ"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExtractHashtags(tc.text); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ExtractHashtags(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
