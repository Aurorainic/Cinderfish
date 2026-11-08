package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// TestTwoInstancePersonUpdate proves that a local profile change reaches the
// copy a remote instance cached. Before the fix the actor document carried no
// icon at all and i/update sent no Update activity, so a new avatar or display
// name never left the instance.
//
// Beta caches Alice's actor by receiving one of her notes before the profile
// change. The final users/show on beta therefore answers from that cached row
// rather than refetching, so a passing assertion means the Update activity was
// applied, not that a fresh fetch papered over the gap.
func TestTwoInstancePersonUpdate(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	routeTo(t, b, a)

	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("local storage: %v", err)
	}
	a.srv.SetStorage(storage)

	aliceToken, aliceID := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")
	aliceActorURI := "https://alpha.example/users/" + aliceID
	keyID, _, bobActor := actorInfo(t, b, bobID)

	if resp, body := postWithToken(t, a.ts, "/api/i/update", aliceToken, map[string]any{
		"name": "Alice Before",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("initial i/update = %d %v", resp.StatusCode, body)
	}

	// Bob follows Alice so that Alice has a remote follower to notify.
	kp, err := b.srv.FederationLocalKey(context.Background(), bobID)
	if err != nil {
		t.Fatalf("beta key: %v", err)
	}
	priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
	if err != nil {
		t.Fatalf("parse beta key: %v", err)
	}
	follow := federation.Object{
		"@context": federation.Context,
		"id":       bobActor + "#follows/personupdate",
		"type":     "Follow",
		"actor":    bobActor,
		"object":   aliceActorURI,
	}
	if resp, body := postSigned(t, a.ts, "/inbox", follow, keyID, priv); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("follow = %d %s", resp.StatusCode, body)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	a.srv.StartFederationWorkers(ctx)

	// A note delivered to beta is what makes it cache Alice's actor.
	if resp, body := postWithToken(t, a.ts, "/api/notes/create", aliceToken, map[string]any{
		"text": "hello before the rename", "visibility": "public",
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("note create = %d %v", resp.StatusCode, body)
	}
	waitForRemoteProfileName(t, b, aliceActorURI, "Alice Before")
	cached := fetchRemoteProfile(t, b)
	if cached["avatarUrl"] != nil && cached["avatarUrl"] != "" {
		t.Fatalf("beta already cached an avatar before the change: %v", cached["avatarUrl"])
	}

	resp, file := uploadFile(t, a.ts.URL, aliceToken, "avatar.png", bytes.NewBufferString("png bytes"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("avatar upload = %d %v", resp.StatusCode, file)
	}
	avatarID, _ := file["id"].(string)
	if avatarID == "" {
		t.Fatalf("avatar upload returned no id: %v", file)
	}

	if resp, body := postWithToken(t, a.ts, "/api/i/update", aliceToken, map[string]any{
		"name":        "Alice Renamed",
		"description": "new bio",
		"avatarId":    avatarID,
	}); resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update = %d %v", resp.StatusCode, body)
	}

	// The actor document itself must publish an absolute icon URL; a peer
	// cannot fetch the /media path the filesystem backend stores.
	resp, body := getPlain(t, a.ts, "/users/"+aliceID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("alpha actor = %d %s", resp.StatusCode, body)
	}
	var person map[string]any
	if err := json.Unmarshal(body, &person); err != nil {
		t.Fatalf("decode actor: %v", err)
	}
	icon, _ := person["icon"].(map[string]any)
	if iconURL, _ := icon["url"].(string); !strings.HasPrefix(iconURL, "https://alpha.example/media/") {
		t.Errorf("actor icon url = %q, want an absolute https://alpha.example/media/ URL", iconURL)
	}

	// Beta applies the queued Update to the profile it cached for Alice.
	waitForRemoteProfileName(t, b, aliceActorURI, "Alice Renamed")
	updated := fetchRemoteProfile(t, b)
	avatarURL, _ := updated["avatarUrl"].(string)
	if !strings.HasPrefix(avatarURL, "https://alpha.example/media/") {
		t.Errorf("beta avatarUrl = %q, want the absolute alpha media URL", avatarURL)
	}
	if updated["description"] != "new bio" {
		t.Errorf("beta description = %v, want new bio", updated["description"])
	}
	_ = bobID
}

// waitForRemoteProfileName polls beta's users/show until the cached copy of the
// remote account carries want. It answers from stored rows, so it observes what
// delivery left behind rather than triggering a refetch.
func waitForRemoteProfileName(t *testing.T, b *instance, actorURI, want string) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for {
		if name, _ := fetchRemoteProfile(t, b)["name"].(string); name == want {
			return
		}
		if time.Now().After(deadline) {
			got, _ := fetchRemoteProfile(t, b)["name"].(string)
			t.Fatalf("beta never cached name %q for %s (still %q)", want, actorURI, got)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// fetchRemoteProfile reads beta's cached view of Alice through users/show.
func fetchRemoteProfile(t *testing.T, b *instance) map[string]any {
	t.Helper()
	resp, out := post(t, b.ts, "/api/users/show", map[string]any{
		"username": "alphaalice",
		"host":     "alpha.example",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("beta users/show = %d %v", resp.StatusCode, out)
	}
	return out
}
