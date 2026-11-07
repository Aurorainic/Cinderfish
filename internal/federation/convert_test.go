package federation

import (
	"strings"
	"testing"
	"time"
)

func questionObject(t *testing.T, fields map[string]any) Object {
	t.Helper()
	base := map[string]any{
		"id":           "https://remote.example/notes/1",
		"type":         "Question",
		"attributedTo": "https://remote.example/users/carol",
		"content":      "<p>pick one</p>",
		"to":           []any{Public},
	}
	for k, v := range fields {
		base[k] = v
	}
	return Object(base)
}

// TestNoteDataFromAPQuestion covers inbound Question parsing: both choice
// lists, the multi-answer flag, the closing time and the refusal cases.
func TestNoteDataFromAPQuestion(t *testing.T) {
	actor := &RemoteUser{ID: "u1", Host: "remote.example", URI: "https://remote.example/users/carol"}
	r := &Renderer{BaseURL: "https://fish.example"}

	one := questionObject(t, map[string]any{
		"oneOf": []any{
			map[string]any{"type": "Note", "name": "red"},
			map[string]any{"type": "Note", "name": "blue "},
		},
		"endTime": "2030-01-02T03:04:05Z",
	})
	data := NoteDataFromAP(one, actor, r)
	if len(data.PollChoices) != 2 || data.PollChoices[0] != "red" || data.PollChoices[1] != "blue" {
		t.Errorf("choices = %v, want [red blue]", data.PollChoices)
	}
	if data.PollMultiple {
		t.Errorf("oneOf marked multiple")
	}
	if data.PollExpiresAt == nil || !data.PollExpiresAt.Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Errorf("expiresAt = %v, want 2030-01-02T03:04:05Z", data.PollExpiresAt)
	}

	multi := questionObject(t, map[string]any{
		"anyOf": []any{
			map[string]any{"name": "a"},
			map[string]any{"name": "b"},
			map[string]any{"name": "c"},
		},
		"endTime": "2030-01-02T03:04:05.123Z",
	})
	data = NoteDataFromAP(multi, actor, r)
	if !data.PollMultiple || len(data.PollChoices) != 3 {
		t.Errorf("anyOf = %v multiple=%v, want three choices and multiple", data.PollChoices, data.PollMultiple)
	}
	if data.PollExpiresAt == nil {
		t.Errorf("fractional endTime not parsed")
	}

	// Fewer than two usable choices means no poll at all.
	tiny := NoteDataFromAP(questionObject(t, map[string]any{
		"oneOf": []any{map[string]any{"name": "only"}},
	}), actor, r)
	if len(tiny.PollChoices) != 0 {
		t.Errorf("single-choice poll accepted: %v", tiny.PollChoices)
	}

	// More than ten choices are trimmed, not rejected.
	var many []any
	for i := 0; i < 15; i++ {
		many = append(many, map[string]any{"name": string(rune('a' + i))})
	}
	trimmed := NoteDataFromAP(questionObject(t, map[string]any{"oneOf": many}), actor, r)
	if len(trimmed.PollChoices) != 10 {
		t.Errorf("choices = %d, want trimmed to 10", len(trimmed.PollChoices))
	}

	// A plain Note never grows a poll, even with oneOf present.
	plain := questionObject(t, map[string]any{
		"type":  "Note",
		"oneOf": []any{map[string]any{"name": "x"}, map[string]any{"name": "y"}},
	})
	if data := NoteDataFromAP(plain, actor, r); len(data.PollChoices) != 0 {
		t.Errorf("plain note got poll choices: %v", data.PollChoices)
	}
}

// TestRenderNoteQuestion proves a local poll federates as a Question with the
// matching choice list.
func TestRenderNoteQuestion(t *testing.T) {
	r := &Renderer{BaseURL: "https://fish.example"}
	expires := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	note := r.RenderNote(NoteData{
		ID: "n1", ActorRef: "https://fish.example/users/u1", Visibility: "public",
		Text: "pick", CreatedAt: time.Now(), PollChoices: []string{"red", "blue"},
		PollExpiresAt: &expires,
	})
	if note.Type() != "Question" {
		t.Errorf("type = %q, want Question", note.Type())
	}
	if note.String("endTime") != "2030-01-02T03:04:05Z" {
		t.Errorf("endTime = %q", note.String("endTime"))
	}
	choices := note.Objects("oneOf")
	if len(choices) != 2 || choices[0].String("name") != "red" {
		t.Errorf("oneOf = %v", choices)
	}
	if len(note.Objects("anyOf")) != 0 {
		t.Errorf("single-choice poll emitted anyOf")
	}

	multi := r.RenderNote(NoteData{
		ID: "n2", ActorRef: "https://fish.example/users/u1", Visibility: "public",
		Text: "pick many", CreatedAt: time.Now(), PollChoices: []string{"a", "b"},
		PollMultiple: true,
	})
	if len(multi.Objects("anyOf")) != 2 || len(multi.Objects("oneOf")) != 0 {
		t.Errorf("multi poll emitted the wrong choice list")
	}
}

// TestNoteDataFromAPEmojis covers custom emoji tags: colons are stripped,
// icon.url wins, href is the fallback, duplicates collapse and an entry with
// no image is dropped.
func TestNoteDataFromAPEmojis(t *testing.T) {
	actor := &RemoteUser{ID: "u1", Host: "remote.example", URI: "https://remote.example/users/carol"}
	r := &Renderer{BaseURL: "https://fish.example"}
	note := Object{
		"id":           "https://remote.example/notes/2",
		"type":         "Note",
		"attributedTo": actor.URI,
		"content":      "<p>:blob: :cat:</p>",
		"to":           []any{Public},
		"tag": []any{
			map[string]any{"type": "Emoji", "name": ":blob:", "icon": map[string]any{"type": "Image", "url": "https://remote.example/emoji/blob.png"}},
			map[string]any{"type": "Emoji", "name": "blob", "icon": map[string]any{"url": "https://remote.example/emoji/blob2.png"}},
			map[string]any{"type": "Emoji", "name": ":cat:", "href": "https://remote.example/emoji/cat.png"},
			map[string]any{"type": "Emoji", "name": ":broken:"},
			map[string]any{"type": "Hashtag", "name": "#tag"},
		},
	}
	data := NoteDataFromAP(note, actor, r)
	if len(data.Emojis) != 2 {
		t.Fatalf("emojis = %v, want blob and cat", data.Emojis)
	}
	if data.Emojis[0].Name != "blob" || data.Emojis[0].URL != "https://remote.example/emoji/blob.png" {
		t.Errorf("first emoji = %+v", data.Emojis[0])
	}
	if data.Emojis[1].Name != "cat" || data.Emojis[1].URL != "https://remote.example/emoji/cat.png" {
		t.Errorf("second emoji = %+v", data.Emojis[1])
	}
	if len(data.Tags) != 1 || data.Tags[0] != "tag" {
		t.Errorf("tags = %v", data.Tags)
	}
}

func TestNoteDataFromAPKeepsLongTextAndAttachments(t *testing.T) {
	actor := &RemoteUser{ID: "u1", Host: "remote.example", URI: "https://remote.example/users/carol"}
	longText := strings.Repeat("long text ", 100)
	note := Object{
		"id": "https://remote.example/notes/long", "type": "Note", "attributedTo": actor.URI,
		"_misskey_content": longText, "content": "<p>short rendered fallback</p>", "to": []any{Public},
		"attachment": []any{
			map[string]any{"type": "Image", "url": "https://cdn.remote.example/media/photo.png", "mediaType": "image/png"},
			map[string]any{"type": "Document", "url": "https://cdn.remote.example/media/file.bin", "mediaType": "application/octet-stream"},
			map[string]any{"type": "Image", "url": "javascript:alert(1)"},
		},
	}
	data := NoteDataFromAP(note, actor, NewRenderer("https://fish.example"))
	if data.Text != longText {
		t.Errorf("text length = %d, want original long body length %d", len(data.Text), len(longText))
	}
	if len(data.Attachments) != 2 {
		t.Fatalf("attachments = %+v, want two safe media URLs", data.Attachments)
	}
	if data.Attachments[0].URL != "https://cdn.remote.example/media/photo.png" || data.Attachments[0].ContentType != "image/png" {
		t.Errorf("image attachment = %+v", data.Attachments[0])
	}
}

func TestNoteDataFromAPPreservesParagraphBreaks(t *testing.T) {
	actor := &RemoteUser{ID: "u1", Host: "remote.example", URI: "https://remote.example/users/carol"}
	data := NoteDataFromAP(Object{
		"id": "https://remote.example/notes/paragraphs", "type": "Note", "attributedTo": actor.URI,
		"content": "<p>first paragraph</p><p>second paragraph</p>", "to": []any{Public},
	}, actor, NewRenderer("https://fish.example"))
	if data.Text != "first paragraph\n\nsecond paragraph" {
		t.Errorf("fallback text = %q", data.Text)
	}
}
