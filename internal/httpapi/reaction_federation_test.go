package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/federation"
)

// TestInboundEmojiReact proves the Firefish/Misskey reaction types are
// accepted: EmojiReact and EmojiReaction create a reaction, and Undo removes
// it, the same as Like.
func TestInboundEmojiReact(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	// Alpha must reach beta to verify the signature and fetch the actor.
	routeTo(t, a, b)

	aliceToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	noteID := createNote(t, a.ts, aliceToken, map[string]any{"text": "react to me"})

	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse beta key: %v", err)
	}

	noteURI := "https://alpha.example/notes/" + noteID
	react := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#reacts/1",
		"type":     "EmojiReact",
		"actor":    bobActor,
		"object":   noteURI,
		"content":  "🍮",
	}
	if resp, body := postSigned(t, a.ts, "/inbox", react, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("EmojiReact = %d %s", resp.StatusCode, body)
	}

	reactions := postListWithToken(t, a.ts, "/api/notes/reactions", aliceToken, map[string]any{"noteId": noteID})
	if len(reactions) != 1 {
		t.Fatalf("reactions = %v, want one", reactions)
	}
	row, _ := reactions[0].(map[string]any)
	if row["type"] != "🍮" {
		t.Errorf("reaction = %v, want 🍮", row["type"])
	}

	// Undo(EmojiReact) removes it again.
	undo := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#undo/1",
		"type":     "Undo",
		"actor":    bobActor,
		"object":   react,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", undo, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Undo = %d %s", resp.StatusCode, body)
	}
	reactions = postListWithToken(t, a.ts, "/api/notes/reactions", aliceToken, map[string]any{"noteId": noteID})
	if len(reactions) != 0 {
		t.Errorf("reactions after Undo = %v, want none", reactions)
	}
}

// TestInboundCustomEmojiReaction proves a remote :name: reaction keeps its
// image: the Emoji tag is stored and note.reactionEmojis maps the shortcode.
func TestInboundCustomEmojiReaction(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)

	aliceToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	noteID := createNote(t, a.ts, aliceToken, map[string]any{"text": "react with an emoji"})

	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	react := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#reacts/emoji",
		"type":     "EmojiReact",
		"actor":    bobActor,
		"object":   "https://alpha.example/notes/" + noteID,
		"content":  ":blob:",
		"tag": []any{
			map[string]any{
				"type": "Emoji", "name": ":blob:",
				"icon": map[string]any{"type": "Image", "url": "https://beta.example/emoji/blob.png"},
			},
		},
	}
	if resp, body := postSigned(t, a.ts, "/inbox", react, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("EmojiReact = %d %s", resp.StatusCode, body)
	}

	_, shown := postWithToken(t, a.ts, "/api/notes/show", aliceToken, map[string]any{"noteId": noteID})
	reactionEmojis, _ := shown["reactionEmojis"].(map[string]any)
	if reactionEmojis["blob"] != "https://beta.example/emoji/blob.png" {
		t.Errorf("reactionEmojis = %v, want blob mapped to the remote URL", shown["reactionEmojis"])
	}
	emojis, _ := shown["emojis"].([]any)
	found := false
	for _, item := range emojis {
		if e, _ := item.(map[string]any); e["name"] == "blob" {
			found = true
		}
	}
	if !found {
		t.Errorf("emojis = %v, want the reaction emoji merged in", emojis)
	}

	var host, url string
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT host, url FROM emojis WHERE name = 'blob'`).Scan(&host, &url); err != nil {
		t.Fatalf("emoji row: %v", err)
	}
	if host != "beta.example" || url != "https://beta.example/emoji/blob.png" {
		t.Errorf("emoji row = %s/%s", host, url)
	}
}
