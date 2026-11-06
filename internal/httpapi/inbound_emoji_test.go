package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// TestInboundCustomEmoji proves a remote note's Emoji tags are stored and
// packed as note.emojis, so the renderer can show the remote images.
func TestInboundCustomEmoji(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)

	aliceToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	noteURI := "https://beta.example/notes/emoji1"
	create := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#activities/emoji1",
		"type":     "Create",
		"actor":    bobActor,
		"object": federation.Object{
			"id":           noteURI,
			"type":         "Note",
			"attributedTo": bobActor,
			"content":      "<p>:blob: hello</p>",
			"to":           []any{federation.Public},
			"published":    "2026-11-07T00:00:00Z",
			"tag": []any{
				map[string]any{
					"type": "Emoji", "name": ":blob:",
					"icon": map[string]any{"type": "Image", "url": "https://beta.example/emoji/blob.png"},
				},
			},
		},
	}
	if resp, body := postSigned(t, a.ts, "/inbox", create, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Create = %d %s", resp.StatusCode, body)
	}

	var noteID string
	if err := a.srv.TestPool().QueryRow(context.Background(),
		`SELECT id FROM notes WHERE ap_id = $1`, noteURI).Scan(&noteID); err != nil {
		t.Fatalf("stored note: %v", err)
	}
	_, shown := postWithToken(t, a.ts, "/api/notes/show", aliceToken, map[string]any{"noteId": noteID})
	emojis, _ := shown["emojis"].([]any)
	if len(emojis) != 1 {
		t.Fatalf("emojis = %v, want one", shown["emojis"])
	}
	emoji, _ := emojis[0].(map[string]any)
	if emoji["name"] != "blob" || emoji["url"] != "https://beta.example/emoji/blob.png" {
		t.Errorf("emoji = %v", emoji)
	}
}

// TestOutboundCustomEmojiTags proves a local note's custom emoji travels as
// Emoji tags with an absolute image URL, and the far side stores them.
func TestOutboundCustomEmojiTags(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	if _, err := a.srv.TestPool().Exec(context.Background(), `
INSERT INTO users (id, username, username_lower, password_hash, token, is_admin)
VALUES ('aroot', 'aroot', 'aroot', 'x', 'admintoken000000', TRUE)`); err != nil {
		t.Fatalf("insert admin: %v", err)
	}

	aliceToken, aliceID := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")

	// Upload a file and register it as the :blob: emoji.
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	a.srv.SetStorage(storage)
	uploaded, body := uploadFile(t, a.ts.URL, aliceToken, "blob.png", bytes.NewBufferString("\x89PNG\r\n\x1a\nfake"))
	if uploaded.StatusCode != http.StatusOK {
		t.Fatalf("upload: %d %v", uploaded.StatusCode, body)
	}
	fileID, _ := body["id"].(string)
	if resp, out := postWithToken(t, a.ts, "/api/admin/emoji/add", "admintoken000000", map[string]any{
		"name": "blob", "fileId": fileID,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("emoji add: %d %v", resp.StatusCode, out)
	}

	// Bob follows Alice so the Create is delivered.
	keyID, _, bobActor := actorInfo(t, b, bobID)
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	follow := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#follows/emoji",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   "https://alpha.example/users/" + aliceID,
	}
	if resp, out := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, out)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)

	if resp, out := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "look :blob: here", "visibility": "public",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("note create: %d %v", resp.StatusCode, out)
	}

	var noteID string
	deadline := time.Now().Add(8 * time.Second)
	for {
		err := b.srv.TestPool().QueryRow(context.Background(),
			`SELECT id FROM notes WHERE user_host = 'alpha.example' AND text = 'look :blob: here' AND deleted_at IS NULL`).Scan(&noteID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("note with custom emoji never delivered")
		}
		time.Sleep(200 * time.Millisecond)
	}

	_, shown := postWithToken(t, b.ts, "/api/notes/show", bobToken, map[string]any{"noteId": noteID})
	emojis, _ := shown["emojis"].([]any)
	if len(emojis) != 1 {
		t.Fatalf("delivered note emojis = %v, want one", shown["emojis"])
	}
	emoji, _ := emojis[0].(map[string]any)
	if emoji["name"] != "blob" {
		t.Errorf("emoji name = %v", emoji["name"])
	}
	if url, _ := emoji["url"].(string); !strings.HasPrefix(url, "https://alpha.example/") {
		t.Errorf("emoji url = %q, want the alpha base URL", url)
	}
}
