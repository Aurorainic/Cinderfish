package federation

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRenderPerson(t *testing.T) {
	r := NewRenderer("https://b.example/")
	p := Profile{
		ID: "abc", Username: "alice", Name: "Alice",
		Description: "hi **there**",
		PublicKey:   "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----\n",
	}
	o := r.RenderPerson(p)

	if o["id"] != "https://b.example/users/abc" {
		t.Errorf("id = %v", o["id"])
	}
	if o["type"] != "Person" {
		t.Errorf("type = %v", o["type"])
	}
	// The five collection URLs are rooted at the actor id, not at /@username.
	for _, field := range []string{"inbox", "outbox", "followers", "following", "featured"} {
		want := "https://b.example/users/abc/" + field
		if field == "featured" {
			want = "https://b.example/users/abc/collections/featured"
		}
		if got := o[field]; got != want {
			t.Errorf("%s = %v, want %s", field, got, want)
		}
	}
	if o["sharedInbox"] != "https://b.example/inbox" {
		t.Errorf("sharedInbox = %v", o["sharedInbox"])
	}
	if o["url"] != "https://b.example/@alice" {
		t.Errorf("url = %v", o["url"])
	}
	if o["preferredUsername"] != "alice" {
		t.Errorf("preferredUsername = %v", o["preferredUsername"])
	}
	if o["_misskey_summary"] != "hi **there**" {
		t.Errorf("_misskey_summary = %v", o["_misskey_summary"])
	}
	if !strings.Contains(o["summary"].(string), "<b>there</b>") {
		t.Errorf("summary did not render MFM: %v", o["summary"])
	}

	key, ok := o["publicKey"].(Object)
	if !ok {
		t.Fatalf("publicKey is %T", o["publicKey"])
	}
	if key["id"] != "https://b.example/users/abc#main-key" {
		t.Errorf("key id = %v", key["id"])
	}
	if key["owner"] != "https://b.example/users/abc" {
		t.Errorf("key owner = %v", key["owner"])
	}

	if _, ok := o["@context"]; !ok {
		t.Error("actor has no @context")
	}
}

func TestRenderPersonTypes(t *testing.T) {
	r := NewRenderer("https://b.example")
	// A username containing a dot is a system account upstream, which
	// is rendered as an Application.
	if got := r.RenderPerson(Profile{ID: "1", Username: "instance.actor"})["type"]; got != "Application" {
		t.Errorf("system account type = %v, want Application", got)
	}
	if got := r.RenderPerson(Profile{ID: "2", Username: "botty", IsBot: true})["type"]; got != "Service" {
		t.Errorf("bot type = %v, want Service", got)
	}
}

func TestRenderNoteVisibilityAudience(t *testing.T) {
	r := NewRenderer("https://b.example")
	base := NoteData{
		ID: "n1", ActorRef: "https://b.example/users/u1",
		Text: "hello", CreatedAt: time.Unix(1700000000, 0),
		MentionRefs: []string{"https://a.example/users/remote"},
	}

	cases := []struct {
		visibility string
		wantTo     []string
		wantCc     []string
	}{
		{"public", []string{Public}, []string{"https://b.example/users/u1/followers", "https://a.example/users/remote"}},
		{"home", []string{"https://b.example/users/u1/followers"}, []string{Public, "https://a.example/users/remote"}},
		{"followers", []string{"https://b.example/users/u1/followers"}, []string{"https://a.example/users/remote"}},
		{"specified", []string{"https://a.example/users/remote"}, nil},
	}
	for _, c := range cases {
		t.Run(c.visibility, func(t *testing.T) {
			n := base
			n.Visibility = c.visibility
			o := r.RenderNote(n)
			assertStrings(t, "to", o.Refs("to"), c.wantTo)
			assertStrings(t, "cc", o.Refs("cc"), c.wantCc)
		})
	}
}

func TestRenderNoteAPIDOverride(t *testing.T) {
	r := NewRenderer("https://b.example")
	// A note that came from a remote instance keeps that instance as
	// its id, which is what makes the id dereferenceable by peers.
	o := r.RenderNote(NoteData{
		ID: "local1", APID: "https://a.example/notes/xyz",
		ActorRef: "https://a.example/users/remote", ActorHost: "a.example",
		Visibility: "public", Text: "remote note",
	})
	if o["id"] != "https://a.example/notes/xyz" {
		t.Errorf("id = %v", o["id"])
	}
}

func TestRenderNoteMisskeyExtensions(t *testing.T) {
	r := NewRenderer("https://b.example")
	o := r.RenderNote(NoteData{
		ID: "n1", ActorRef: "https://b.example/users/u1",
		Visibility: "public", Text: "**bold** #tag", CW: "spoiler",
		QuoteURL: "https://a.example/notes/q",
		Tags:     []string{"tag"},
	})
	if o["_misskey_content"] != "**bold** #tag" {
		t.Errorf("_misskey_content = %v", o["_misskey_content"])
	}
	src, ok := o["source"].(Object)
	if !ok {
		t.Fatalf("source is %T", o["source"])
	}
	if src["mediaType"] != "text/x.misskeymarkdown" {
		t.Errorf("source.mediaType = %v", src["mediaType"])
	}
	if o["summary"] != "spoiler" {
		t.Errorf("summary = %v", o["summary"])
	}
	if o["quoteUrl"] != "https://a.example/notes/q" {
		t.Errorf("quoteUrl = %v", o["quoteUrl"])
	}
	if o["_misskey_quote"] != "https://a.example/notes/q" {
		t.Errorf("_misskey_quote = %v", o["_misskey_quote"])
	}

	tags := o.Objects("tag")
	found := false
	for _, tag := range tags {
		if tag.Type() == "Hashtag" && tag["name"] == "#tag" {
			found = true
		}
	}
	if !found {
		// Objects() returns []Object; Object is a map[string]any so Type()
		// works there. Re-check through the raw JSON to catch a case
		// where rendering wrote a plain map instead.
		if raw, ok := o["tag"].([]any); ok {
			for _, item := range raw {
				m, ok := item.(Object)
				if ok && m["type"] == "Hashtag" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("tag list has no Hashtag: %v", o["tag"])
	}
}

func TestRenderNoteIsValidJSON(t *testing.T) {
	r := NewRenderer("https://b.example")
	o := r.RenderNote(NoteData{
		ID: "n1", ActorRef: "https://b.example/users/u1",
		Visibility: "public", Text: "hi", CreatedAt: time.Now(),
	})
	if _, err := json.Marshal(o); err != nil {
		t.Fatalf("rendered note does not marshal: %v", err)
	}
}

func TestRenderFollowAcceptUndo(t *testing.T) {
	r := NewRenderer("https://b.example")
	f := r.RenderFollow("https://b.example/follows/1", "https://b.example/users/u1", "https://a.example/users/r1")
	if f.Ref("id") != "https://b.example/follows/1" {
		t.Errorf("follow id = %v", f.Ref("id"))
	}

	accept := r.RenderAccept("https://a.example/users/r1", f)
	if accept["type"] != "Accept" {
		t.Errorf("type = %v", accept["type"])
	}
	obj, ok := accept["object"].(Object)
	if !ok {
		t.Fatalf("Accept.object is %T, must embed the original Follow", accept["object"])
	}
	if obj.Ref("id") != "https://b.example/follows/1" {
		t.Errorf("embedded follow id = %v", obj.Ref("id"))
	}

	undo := r.RenderUndo(f, "https://b.example/users/u1")
	if undo["type"] != "Undo" {
		t.Errorf("undo type = %v", undo["type"])
	}
}

func TestRenderLikeCarriesReaction(t *testing.T) {
	r := NewRenderer("https://b.example")
	l := r.RenderLike("https://b.example/likes/1", "https://b.example/users/u1", "https://a.example/notes/n", ":blobcat:")
	if l["_misskey_reaction"] != ":blobcat:" {
		t.Errorf("_misskey_reaction = %v", l["_misskey_reaction"])
	}
	if l["content"] != ":blobcat:" {
		t.Errorf("content = %v", l["content"])
	}
}

func TestMFMToHTML(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "hello", "hello"},
		{"escapes html", "<script>", "&lt;script&gt;"},
		{"mentions", "hi @alice", "hi <a href=\"/@alice\" class=\"u-url mention\">@alice</a>"},
		{"remote mention", "@bob@a.example", "<a href=\"https://a.example/@bob\" class=\"u-url mention\">@bob@a.example</a>"},
		{"bold", "**hi**", "<b>hi</b>"},
		{"hashtag", "#go", "<a href=\"/tags/go\" rel=\"tag\">#go</a>"},
		{"url", "see https://x.example/y", "see <a href=\"https://x.example/y\" rel=\"nofollow noopener\" target=\"_blank\">https://x.example/y</a>"},
		{"inline code", "`a<b`", "<code>a&lt;b</code>"},
		{"line break", "a\nb", "a<br>\nb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MFMToHTML(c.in); got != c.want {
				t.Errorf("MFMToHTML(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMFMToHTMLCodeSpanNotScanned(t *testing.T) {
	// A URL inside inline code must stay literal, otherwise a note about
	// a link would federate as an embedded link.
	got := MFMToHTML("`https://x.example`")
	if strings.Contains(got, "<a ") {
		t.Errorf("linkified inside code: %q", got)
	}
	if got != "<code>https://x.example</code>" {
		t.Errorf("got %q", got)
	}
}

func TestMFMToHTMLFencedCode(t *testing.T) {
	got := MFMToHTML("```go\nfmt.Println(\"x\")\n```")
	if !strings.Contains(got, "<pre><code class=\"language-go\">") {
		t.Errorf("got %q", got)
	}
	if !strings.Contains(got, "fmt.Println(") {
		t.Errorf("code body lost: %q", got)
	}
}

func assertStrings(t *testing.T, name string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", name, i, got[i], want[i])
		}
	}
}
