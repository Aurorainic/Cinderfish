package httpapi_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// TestAPShowResolvesRemoteObjects proves ap/show: a URL on another instance
// resolves to a stored note or user, packed in the normal shapes.
func TestAPShowResolvesRemoteObjects(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)

	viewerToken, _ := a.signup(t, "alphaalice")
	bobToken, bobID := b.signup(t, "betabob")
	noteID := createNote(t, b.ts, bobToken, map[string]any{"text": "hello from beta"})

	// A note URL is fetched, stored and packed.
	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/notes/" + noteID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show note = %d %v", resp.StatusCode, out)
	}
	if out["type"] != "Note" {
		t.Fatalf("type = %v, want Note", out["type"])
	}
	note, _ := out["object"].(map[string]any)
	if note == nil || note["id"] == "" {
		t.Fatalf("packed note = %v", out["object"])
	}
	if note["text"] != "hello from beta" {
		t.Errorf("note text = %v", note["text"])
	}

	// An actor URL resolves to a user.
	resp, out = postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/users/" + bobID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show user = %d %v", resp.StatusCode, out)
	}
	if out["type"] != "User" {
		t.Fatalf("type = %v, want User", out["type"])
	}
	user, _ := out["object"].(map[string]any)
	if user == nil || user["username"] != "betabob" {
		t.Fatalf("packed user = %v", out["object"])
	}
}

func TestRemoteNoteMediaIsCachedAndPacked(t *testing.T) {
	makeInstance := func(domain string) (*httpapi.Server, *httptest.Server, *pgxpool.Pool) {
		pool := testPool(t)
		cfg := &config.Config{Host: "127.0.0.1", Port: 0, Domain: domain, RegistrationMode: "open", FederationMode: "public", SigninRateLimit: -1}
		srv := httpapi.NewServerWithDB(cfg, pool)
		srv.AttachStreaming(t.Logf)
		srv.AttachFederation()
		if _, err := pool.Exec(context.Background(), `INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb('public'::text)) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
			t.Fatal(err)
		}
		ts := httptest.NewServer(srv)
		t.Cleanup(ts.Close)
		return srv, ts, pool
	}
	aSrv, aTS, aPool := makeInstance("alpha.example")
	bSrv, bTS, _ := makeInstance("beta.example")
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	aSrv.SetStorage(storage)
	storageB, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	bSrv.SetStorage(storageB)
	_, viewer := post(t, aTS, "/api/signup", map[string]any{"username": "alphaalice", "password": "correct horse battery staple"})
	_, bob := post(t, bTS, "/api/signup", map[string]any{"username": "betabob", "password": "correct horse battery staple"})
	viewerToken := viewer["token"].(string)
	bobToken := bob["token"].(string)
	aSrv.SetFederationHTTPClient(&http.Client{Transport: &hostRewriteTransport{
		hosts: map[string]string{"beta.example": bTS.URL},
	}})
	imageBytes := []byte("image-payload")
	_, uploaded := uploadFile(t, bTS.URL, bobToken, "remote.png", bytes.NewBuffer(imageBytes))
	resp, created := postWithToken(t, bTS, "/api/notes/create", bobToken, map[string]any{"text": strings.Repeat("long body ", 100), "fileIds": []string{uploaded["id"].(string)}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remote note create: %d %v", resp.StatusCode, created)
	}
	noteID := created["createdNote"].(map[string]any)["id"].(string)
	resp, result := postWithToken(t, aTS, "/api/ap/show", viewerToken, map[string]any{"uri": "https://beta.example/notes/" + noteID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ap/show: %d %v", resp.StatusCode, result)
	}
	remoteNote := result["object"].(map[string]any)
	if len(remoteNote["text"].(string)) != len(strings.Repeat("long body ", 100)) {
		t.Errorf("imported text was truncated: %d bytes", len(remoteNote["text"].(string)))
	}
	files := remoteNote["files"].([]any)
	if len(files) != 1 {
		t.Fatalf("imported files = %v, want one cached attachment", files)
	}
	file := files[0].(map[string]any)
	if !strings.HasPrefix(file["url"].(string), "/media/") {
		t.Errorf("cached file url = %v, want local media URL", file["url"])
	}
	mediaResp, err := aTS.Client().Get(aTS.URL + file["url"].(string))
	if err != nil {
		t.Fatal(err)
	}
	defer mediaResp.Body.Close()
	content, err := io.ReadAll(mediaResp.Body)
	if err != nil || string(content) != "image-payload" {
		t.Errorf("cached media response = %q, err=%v", content, err)
	}
	if _, err := aPool.Exec(context.Background(), `SELECT 1 FROM drive_files WHERE id = $1`, file["id"]); err != nil {
		t.Errorf("cached media row missing: %v", err)
	}
}

func TestFederationInboundNoteKeepsCachedAttachment(t *testing.T) {
	makeInstance := func(domain string) (*httpapi.Server, *httptest.Server, *pgxpool.Pool) {
		pool := testPool(t)
		cfg := &config.Config{Host: "127.0.0.1", Port: 0, Domain: domain, RegistrationMode: "open", FederationMode: "public", SigninRateLimit: -1}
		srv := httpapi.NewServerWithDB(cfg, pool)
		srv.AttachStreaming(t.Logf)
		srv.AttachFederation()
		if _, err := pool.Exec(context.Background(), `INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb('public'::text)) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
			t.Fatal(err)
		}
		ts := httptest.NewServer(srv)
		t.Cleanup(ts.Close)
		return srv, ts, pool
	}
	aSrv, aTS, aPool := makeInstance("alpha.example")
	bSrv, bTS, _ := makeInstance("beta.example")
	aStorage, _ := media.NewLocalStorage(t.TempDir(), "/media")
	bStorage, _ := media.NewLocalStorage(t.TempDir(), "/media")
	aSrv.SetStorage(aStorage)
	bSrv.SetStorage(bStorage)
	aSrv.SetFederationHTTPClient(&http.Client{Transport: &hostRewriteTransport{hosts: map[string]string{"beta.example": bTS.URL}}})
	_, _ = post(t, aTS, "/api/signup", map[string]any{"username": "alphaalice", "password": "correct horse battery staple"})
	_, bUser := post(t, bTS, "/api/signup", map[string]any{"username": "betabob", "password": "correct horse battery staple"})
	bToken := bUser["token"].(string)
	_, uploaded := uploadFile(t, bTS.URL, bToken, "remote.png", bytes.NewBufferString("inbound-image"))
	attachmentURL := uploaded["url"].(string)
	if strings.HasPrefix(attachmentURL, "/") {
		attachmentURL = "https://beta.example" + attachmentURL
	}
	createdResp, created := postWithToken(t, bTS, "/api/notes/create", bToken, map[string]any{"text": "inbound attachment", "fileIds": []string{uploaded["id"].(string)}})
	if createdResp.StatusCode != http.StatusOK {
		t.Fatalf("remote create: %d %v", createdResp.StatusCode, created)
	}
	noteID := created["createdNote"].(map[string]any)["id"].(string)
	activity := federation.Object{
		"@context": federation.Context, "id": "https://beta.example/notes/" + noteID + "/activity", "type": "Create",
		"actor": "https://beta.example/users/" + bUser["id"].(string),
		"object": federation.Object{
			"id": "https://beta.example/notes/" + noteID, "type": "Note",
			"attributedTo": "https://beta.example/users/" + bUser["id"].(string),
			"content":      "inbound attachment", "_misskey_content": "inbound attachment",
			"to": []string{federation.Public}, "attachment": []any{map[string]any{
				"type": "Document", "mediaType": "image/png", "url": attachmentURL, "name": "remote.png",
			}},
		},
	}
	keyPair, err := bSrv.FederationLocalKey(context.Background(), bUser["id"].(string))
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := federation.ParsePrivateKeyPEM(keyPair.PrivatePEM)
	if err != nil {
		t.Fatal(err)
	}
	keyID := "https://beta.example/users/" + bUser["id"].(string) + "#main-key"
	bSrv.SetFederationHTTPClient(&http.Client{Transport: &hostRewriteTransport{hosts: map[string]string{"beta.example": bTS.URL}}})
	aSrv.SetFederationHTTPClient(&http.Client{Transport: &hostRewriteTransport{
		hosts: map[string]string{"beta.example": bTS.URL},
	}})
	if resp, body := postSigned(t, aTS, "/inbox", activity, keyID, privateKey); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("inbox create = %d %s", resp.StatusCode, body)
	}
	var localNoteID string
	if err := aPool.QueryRow(context.Background(), `SELECT id FROM notes WHERE ap_id = $1`, "https://beta.example/notes/"+noteID).Scan(&localNoteID); err != nil {
		t.Fatal(err)
	}
	_, packed := postWithToken(t, aTS, "/api/notes/show", aUserToken(t, aTS), map[string]any{"noteId": localNoteID})
	files := packed["files"].([]any)
	if len(files) != 1 {
		var rows int
		_ = aPool.QueryRow(context.Background(), `SELECT count(*) FROM drive_files WHERE user_id IS NULL`).Scan(&rows)
		t.Fatalf("inbox imported files = %v, anonymous cached drive rows=%d, want one", files, rows)
	}
}

func aUserToken(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	resp, out := post(t, ts, "/api/signin", map[string]any{"username": "alphaalice", "password": "correct horse battery staple"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signin alphaalice: %d %v", resp.StatusCode, out)
	}
	return out["i"].(string)
}

// TestAPShowRequiresCredential keeps the lookup authenticated.
func TestAPShowRequiresCredential(t *testing.T) {
	a := newInstance(t, "alpha")
	resp, _ := post(t, a.ts, "/api/ap/show", map[string]any{"uri": "https://beta.example/notes/x"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous ap/show = %d, want 401", resp.StatusCode)
	}
}

// TestAPShowRejectsNonEntity proves an ActivityPub collection (or any object
// that is neither a note nor an actor) is refused instead of stored.
func TestAPShowRejectsNonEntity(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	routeTo(t, a, b)
	viewerToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://beta.example/users/" + bobID + "/followers",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("collection = %d %v, want 400", resp.StatusCode, out)
	}
	if e, _ := out["error"].(map[string]any); e["code"] != "NO_SUCH_OBJECT" {
		t.Errorf("code = %v, want NO_SUCH_OBJECT", e["code"])
	}
}

// TestAPShowRejectsForeignNote proves a note whose id lives on one host but
// which claims an author on another is refused.
func TestAPShowRejectsForeignNote(t *testing.T) {
	a := newInstance(t, "alpha")
	b := newInstance(t, "beta")
	viewerToken, _ := a.signup(t, "alphaalice")
	_, bobID := b.signup(t, "betabob")

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write([]byte(`{
			"@context": "https://www.w3.org/ns/activitystreams",
			"id": "https://evil.example/notes/1",
			"type": "Note",
			"attributedTo": "https://beta.example/users/` + bobID + `",
			"content": "foreign",
			"to": ["https://www.w3.org/ns/activitystreams#Public"]
		}`))
	}))
	t.Cleanup(fake.Close)
	a.srv.SetFederationHTTPClient(&http.Client{
		Transport: &hostRewriteTransport{hosts: map[string]string{
			"evil.example": fake.URL,
			"beta.example": b.ts.URL,
		}},
		Timeout: 10 * timeSecond,
	})
	actorInfo(t, b, bobID)

	resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{
		"uri": "https://evil.example/notes/1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign note = %d %v, want 400", resp.StatusCode, out)
	}
}

// TestAPShowRefusesPrivateAddress proves the federation fetch path cannot
// be pointed at the instance's own network: a literal loopback or private
// IP is refused before any request is made.
func TestAPShowRefusesPrivateAddress(t *testing.T) {
	a := newInstance(t, "alpha")
	_, viewerToken := a.signup(t, "alphaalice")

	for _, uri := range []string{
		"http://127.0.0.1:5432/",
		"http://[::1]:5432/",
		"http://10.0.0.1/secret",
		"http://169.254.169.254/latest/meta-data/",
	} {
		resp, out := postWithToken(t, a.ts, "/api/ap/show", viewerToken, map[string]any{"uri": uri})
		if resp.StatusCode == http.StatusOK {
			t.Errorf("ap/show %q = 200 %v, want a refusal", uri, out)
		}
	}
}
