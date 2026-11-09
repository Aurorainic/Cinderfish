package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// The pages/* family, per packages/backend/src/server/api/endpoints/pages/.
// The port stores the full content model; the viewer executes only the
// four static block types, exactly as upstream's client at c05e844 does.

func pageUpload(t *testing.T, f *fedInstance, token, name string) string {
	t.Helper()
	file := uploadTyped(t, f.ts.URL, token, name, "image/png", []byte{0x89, 'P', 'N', 'G'})
	id, _ := file["id"].(string)
	if id == "" {
		t.Fatalf("upload %s returned no id: %v", name, file)
	}
	return id
}

func TestPagesCRUDShowAndLikes(t *testing.T) {
	f := newFederationServer(t, "public")
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	f.srv.SetStorage(storage)
	bannerID := pageUpload(t, f, aliceToken, "banner.png")

	content := []any{
		map[string]any{"type": "text", "text": "hello from the page"},
		map[string]any{"type": "section", "children": []any{
			map[string]any{"type": "image", "fileId": bannerID},
		}},
		map[string]any{"type": "note", "note": "note0000000000000"},
		map[string]any{"type": "button", "text": "dynamic"},
	}

	// create with an attachment and every static block type.
	resp, page := postWithToken(t, f.ts, "/api/pages/create", aliceToken, map[string]any{
		"title": "My Page", "name": "my-page", "summary": "about",
		"content": content, "variables": []any{}, "script": "",
		"eyeCatchingImageId": bannerID, "font": "serif", "alignCenter": true,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, page)
	}
	pageID, _ := page["id"].(string)
	if page["name"] != "my-page" || page["font"] != "serif" {
		t.Errorf("created page = %v", page)
	}
	// attachedFiles carries the image block's drive file.
	if files, ok := page["attachedFiles"].([]any); !ok || len(files) != 1 {
		t.Errorf("attachedFiles = %v, want the banner", page["attachedFiles"])
	}
	if page["eyeCatchingImageId"] != bannerID {
		t.Errorf("eyeCatchingImageId = %v", page["eyeCatchingImageId"])
	}

	// The slug is unique per user: NAME_ALREADY_EXISTS.
	resp, out := postWithToken(t, f.ts, "/api/pages/create", aliceToken, map[string]any{
		"title": "dup", "name": "my-page", "content": []any{}, "variables": []any{}, "script": "",
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NAME_ALREADY_EXISTS" {
		t.Fatalf("duplicate name: %d %v", resp.StatusCode, out)
	}
	// Another user may use the same slug.
	resp, _ = postWithToken(t, f.ts, "/api/pages/create", bobToken, map[string]any{
		"title": "also my-page", "name": "my-page", "content": []any{}, "variables": []any{}, "script": "",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("same slug other user: %d", resp.StatusCode)
	}

	// The eye-catching image must be the caller's.
	resp, out = postWithToken(t, f.ts, "/api/pages/create", bobToken, map[string]any{
		"title": "foreign banner", "name": "foreign-banner", "content": []any{},
		"variables": []any{}, "script": "", "eyeCatchingImageId": bannerID,
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NO_SUCH_FILE" {
		t.Fatalf("foreign banner: %d %v", resp.StatusCode, out)
	}

	// show by id and by {name, username}.
	resp, out = postWithToken(t, f.ts, "/api/pages/show", aliceToken, map[string]any{"pageId": pageID})
	if resp.StatusCode != http.StatusOK || out["id"] != pageID {
		t.Fatalf("show by id: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/show", bobToken, map[string]any{
		"name": "my-page", "username": "alice",
	})
	if resp.StatusCode != http.StatusOK || out["id"] != pageID {
		t.Fatalf("show by name: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/show", bobToken, map[string]any{"pageId": "missing00000000"})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_PAGE" {
		t.Fatalf("show unknown: %d %v", resp.StatusCode, out)
	}

	// update merges; upstream returns nothing, so the status is 204.
	if code := postNoContentToken(t, f.ts, "/api/pages/update", aliceToken, map[string]any{
		"pageId": pageID, "summary": nil, "title": "My Page 2",
	}); code != http.StatusNoContent {
		t.Fatalf("update: %d", code)
	}
	// A name change to the same slug is allowed (it is the owner's own);
	// the uniqueness collision is tested above. Foreign update answers
	// not-found: the ownership question is answered upfront.
	resp, out = postWithToken(t, f.ts, "/api/pages/update", bobToken, map[string]any{
		"pageId": pageID, "title": "hijack",
	})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_PAGE" {
		t.Fatalf("foreign update: %d %v", resp.StatusCode, out)
	}

	// Listings.
	if got := postListWithToken(t, f.ts, "/api/i/pages", aliceToken, nil); len(got) != 1 {
		t.Errorf("i/pages = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/users/pages", bobToken, map[string]any{"userId": aliceID}); len(got) != 1 {
		t.Errorf("users/pages = %d, want 1", len(got))
	}

	// Likes: your own page is YOUR_PAGE; bob likes once, then ALREADY_LIKED.
	resp, out = postWithToken(t, f.ts, "/api/pages/like", aliceToken, map[string]any{"pageId": pageID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "YOUR_PAGE" {
		t.Fatalf("self like: %d %v", resp.StatusCode, out)
	}
	if code := postNoContentToken(t, f.ts, "/api/pages/like", bobToken, map[string]any{"pageId": pageID}); code != http.StatusNoContent {
		t.Fatalf("like: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/like", bobToken, map[string]any{"pageId": pageID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_LIKED" {
		t.Fatalf("double like: %d %v", resp.StatusCode, out)
	}
	// featured now lists the page.
	if got := postListWithToken(t, f.ts, "/api/pages/featured", aliceToken, nil); len(got) != 1 {
		t.Errorf("featured = %d, want 1", len(got))
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/show", bobToken, map[string]any{"pageId": pageID})
	if out["isLiked"] != true || out["likedCount"] != float64(1) {
		t.Errorf("after like = %v/%v", out["isLiked"], out["likedCount"])
	}
	likes := postListWithToken(t, f.ts, "/api/i/page-likes", bobToken, nil)
	if len(likes) != 1 {
		t.Fatalf("i/page-likes = %d, want 1", len(likes))
	}

	// Unlike twice: the second is NOT_LIKED.
	if code := postNoContentToken(t, f.ts, "/api/pages/unlike", bobToken, map[string]any{"pageId": pageID}); code != http.StatusNoContent {
		t.Fatalf("unlike: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/unlike", bobToken, map[string]any{"pageId": pageID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NOT_LIKED" {
		t.Fatalf("second unlike: %d %v", resp.StatusCode, out)
	}

	// Pinning: a foreign page answers NO_SUCH_PAGE with i/update's id;
	// the owner's pin travels through users/show and i.
	resp, out = postWithToken(t, f.ts, "/api/i/update", bobToken, map[string]any{"pinnedPageId": pageID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NO_SUCH_PAGE" {
		t.Fatalf("foreign pin: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/i/update", aliceToken, map[string]any{"pinnedPageId": pageID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pin: %d %v", resp.StatusCode, out)
	}
	if out["pinnedPageId"] != pageID {
		t.Errorf("i pinnedPageId = %v", out["pinnedPageId"])
	}
	resp, out = postWithToken(t, f.ts, "/api/users/show", bobToken, map[string]any{"userId": aliceID})
	if out["pinnedPageId"] != pageID {
		t.Errorf("users/show pinnedPageId = %v", out["pinnedPageId"])
	}
	if _, ok := out["pinnedPage"].(map[string]any); !ok {
		t.Errorf("users/show pinnedPage missing: %v", out["pinnedPage"])
	}

	// Deletion clears the pin through the FK's SET NULL.
	if code := postNoContentToken(t, f.ts, "/api/pages/delete", aliceToken, map[string]any{"pageId": pageID}); code != http.StatusNoContent {
		t.Fatalf("delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/users/show", bobToken, map[string]any{"userId": aliceID})
	if out["pinnedPageId"] != nil {
		t.Errorf("pinnedPageId after delete = %v, want nil", out["pinnedPageId"])
	}
	resp, out = postWithToken(t, f.ts, "/api/pages/show", bobToken, map[string]any{"pageId": pageID})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("show after delete = %d", resp.StatusCode)
	}
}
