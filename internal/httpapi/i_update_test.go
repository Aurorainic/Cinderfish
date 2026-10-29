package httpapi_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func TestIUpdateRequiresCredential(t *testing.T) {
	ts := newTestServer(t, "open")
	resp, body := post(t, ts, "/api/i/update", map[string]any{"name": "Tuna"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body %v", resp.StatusCode, body)
	}
}

func TestIUpdateProfileFields(t *testing.T) {
	ts := newTestServer(t, "open")
	userID, token := createUser(t, ts, uniqueName(t))

	resp, body := post(t, ts, "/api/i/update", map[string]any{
		"i": token, "name": "Tuna", "description": "self intro",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update: status %d, body %v", resp.StatusCode, body)
	}
	if body["name"] != "Tuna" || body["description"] != "self intro" {
		t.Errorf("i/update response name/description = %v/%v", body["name"], body["description"])
	}

	// The change is reflected in api/i.
	_, me := post(t, ts, "/api/i", map[string]any{"i": token})
	if me["name"] != "Tuna" || me["description"] != "self intro" {
		t.Errorf("api/i name/description = %v/%v", me["name"], me["description"])
	}

	// And in users/show.
	resp, show := post(t, ts, "/api/users/show", map[string]any{"userId": userID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("users/show: status %d, body %v", resp.StatusCode, show)
	}
	if show["name"] != "Tuna" || show["description"] != "self intro" {
		t.Errorf("users/show name/description = %v/%v", show["name"], show["description"])
	}

	// A partial update only touches the given field.
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "description": "updated",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("partial i/update: status %d, body %v", resp.StatusCode, body)
	}
	if body["name"] != "Tuna" || body["description"] != "updated" {
		t.Errorf("partial update kept name %v, set description %v", body["name"], body["description"])
	}
}

func TestIUpdateAvatar(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))
	_, otherToken := createUser(t, ts, uniqueName(t)+"b")

	resp, file := uploadFile(t, ts.URL, token, "me.png", bytes.NewBufferString("png bytes"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload own avatar: status %d, body %v", resp.StatusCode, file)
	}
	fileID, _ := file["id"].(string)
	fileURL, _ := file["url"].(string)

	resp, body := post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": fileID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update with avatar: status %d, body %v", resp.StatusCode, body)
	}
	if body["avatarUrl"] != fileURL {
		t.Errorf("avatarUrl = %v, want %v", body["avatarUrl"], fileURL)
	}

	// Another user's drive file is refused.
	resp, otherFile := uploadFile(t, ts.URL, otherToken, "other.png", bytes.NewBufferString("other"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload other avatar: status %d, body %v", resp.StatusCode, otherFile)
	}
	otherID, _ := otherFile["id"].(string)
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": otherID,
	})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("i/update with a foreign avatar: status %d, want 403; body %v", resp.StatusCode, body)
	}

	// An unknown file id is refused too.
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": "nosuchfile",
	})
	if resp.StatusCode == 0 || resp.StatusCode < http.StatusBadRequest {
		t.Fatalf("i/update with an unknown avatar: status %d, body %v", resp.StatusCode, body)
	}

	// An empty avatarId clears the avatar again.
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": "",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update clearing avatar: status %d, body %v", resp.StatusCode, body)
	}
	if body["avatarUrl"] != nil {
		t.Errorf("avatarUrl after clear = %v, want nil", body["avatarUrl"])
	}

	// An explicit JSON null clears the avatar as well, which is what
	// Misskey's own client sends. Set it once more first.
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": fileID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("re-set avatar: status %d, body %v", resp.StatusCode, body)
	}
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "avatarId": nil,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("i/update null avatar: status %d, body %v", resp.StatusCode, body)
	}
	if body["avatarUrl"] != nil {
		t.Errorf("avatarUrl after null clear = %v, want nil", body["avatarUrl"])
	}
}

func TestIUpdateNullClearsTextFields(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	// Set the fields first.
	resp, body := post(t, ts, "/api/i/update", map[string]any{
		"i": token, "name": "Tuna", "description": "self intro",
	})
	if resp.StatusCode != http.StatusOK || body["name"] != "Tuna" {
		t.Fatalf("setup i/update: status %d, body %v", resp.StatusCode, body)
	}

	// Misskey's client sends explicit null to clear a field; the same
	// request must clear both.
	resp, body = post(t, ts, "/api/i/update", map[string]any{
		"i": token, "name": nil, "description": nil,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("null clear: status %d, body %v", resp.StatusCode, body)
	}
	if body["name"] != nil || body["description"] != nil {
		t.Errorf("after null clear name=%v description=%v, want both nil", body["name"], body["description"])
	}
}

func TestIUpdateLengthCaps(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	// 50 and 500 characters are the limits; one more is refused.
	okName := strings.Repeat("あ", 50)
	tooLongName := strings.Repeat("あ", 51)
	okDescription := strings.Repeat("い", 500)
	tooLongDescription := strings.Repeat("い", 501)

	resp, body := post(t, ts, "/api/i/update", map[string]any{"i": token, "name": okName})
	if resp.StatusCode != http.StatusOK || body["name"] != okName {
		t.Fatalf("50-char name: status %d, body %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/i/update", map[string]any{"i": token, "description": okDescription})
	if resp.StatusCode != http.StatusOK || body["description"] != okDescription {
		t.Fatalf("500-char description: status %d, body %v", resp.StatusCode, body)
	}

	resp, body = post(t, ts, "/api/i/update", map[string]any{"i": token, "name": tooLongName})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("51-char name: status %d, body %v", resp.StatusCode, body)
	}
	resp, body = post(t, ts, "/api/i/update", map[string]any{"i": token, "description": tooLongDescription})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("501-char description: status %d, body %v", resp.StatusCode, body)
	}

	// The rejected values must not have been persisted.
	_, me := post(t, ts, "/api/i", map[string]any{"i": token})
	if me["name"] != okName || me["description"] != okDescription {
		t.Errorf("stored name/description = %v/%v, want the last accepted values", me["name"], me["description"])
	}
}
