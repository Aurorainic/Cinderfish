package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/media"
)

// The gallery/* family plus users/gallery/posts and i/gallery/*, per
// packages/backend/src/server/api/endpoints/gallery/

// galleryUpload uploads one PNG-shaped file and returns its id, the way
// the editor's drive picker would deliver.
func galleryUpload(t *testing.T, f *fedInstance, token, name string) string {
	t.Helper()
	file := uploadTyped(t, f.ts.URL, token, name, "image/png", []byte{0x89, 'P', 'N', 'G'})
	id, _ := file["id"].(string)
	if id == "" {
		t.Fatalf("upload %s returned no id: %v", name, file)
	}
	return id
}

func TestGalleryCRUDAndLikes(t *testing.T) {
	f := newFederationServer(t, "public")
	// Gallery posts reference drive files, so the federation fixture gains
	// a local storage the uploads land in.
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	f.srv.SetStorage(storage)
	aliceToken, aliceID := f.registerUser(t, "alice")
	bobToken, _ := f.registerUser(t, "bob")

	fileID := galleryUpload(t, f, aliceToken, "pic.png")
	fileB := galleryUpload(t, f, bobToken, "bobs.png")

	// create with a foreign file id drops it; the post then has no files,
	// which is the recorded INVALID_PARAM divergence.
	resp, out := postWithToken(t, f.ts, "/api/gallery/posts/create", aliceToken, map[string]any{
		"title": "stolen", "fileIds": []string{fileB},
	})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) == nil {
		t.Fatalf("foreign-file create: %d %v", resp.StatusCode, out)
	}

	resp, post := postWithToken(t, f.ts, "/api/gallery/posts/create", aliceToken, map[string]any{
		"title": "shots", "description": "my pictures",
		"fileIds": []string{fileID}, "isSensitive": false,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create: %d %v", resp.StatusCode, post)
	}
	postID, _ := post["id"].(string)
	if post["title"] != "shots" || post["likedCount"] != float64(0) {
		t.Errorf("created post = %v", post)
	}
	if files, ok := post["files"].([]any); !ok || len(files) != 1 {
		t.Errorf("packed files = %v, want the drive file", post["files"])
	}

	// update renames and clears the description explicitly.
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/update", aliceToken, map[string]any{
		"postId": postID, "title": "shots 2", "description": nil,
	})
	if resp.StatusCode != http.StatusOK || out["title"] != "shots 2" {
		t.Fatalf("update: %d %v", resp.StatusCode, out)
	}
	if _, present := out["description"]; present && out["description"] != nil {
		t.Errorf("cleared description should be null, got %v", out["description"])
	}

	// files order is preserved across reads.
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/update", aliceToken, map[string]any{
		"postId": postID, "fileIds": []string{fileID, fileID},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update files: %d %v", resp.StatusCode, out)
	}
	if ids, ok := out["fileIds"].([]any); !ok || len(ids) != 1 {
		t.Errorf("duplicate fileIds should deduplicate to 1, got %v", out["fileIds"])
	}

	// Foreign update is a silent no-op upstream; Cinderfish answers the
	// ownership question upfront with not-found.
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/update", bobToken, map[string]any{
		"postId": postID, "title": "hijack",
	})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_POST" {
		t.Fatalf("foreign update: %d %v", resp.StatusCode, out)
	}

	// Likes: your own post is YOUR_POST; bob can like once.
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/like", aliceToken, map[string]any{"postId": postID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "YOUR_POST" {
		t.Fatalf("self like: %d %v", resp.StatusCode, out)
	}
	if code := postNoContentToken(t, f.ts, "/api/gallery/posts/like", bobToken, map[string]any{"postId": postID}); code != http.StatusNoContent {
		t.Fatalf("like: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/like", bobToken, map[string]any{"postId": postID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "ALREADY_LIKED" {
		t.Fatalf("double like: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/show", bobToken, map[string]any{"postId": postID})
	if out["likedCount"] != float64(1) || out["isLiked"] != true {
		t.Errorf("after like = %v/%v, want 1/true", out["likedCount"], out["isLiked"])
	}
	likes := postListWithToken(t, f.ts, "/api/i/gallery/likes", bobToken, nil)
	if len(likes) != 1 {
		t.Fatalf("i/gallery/likes = %d, want 1", len(likes))
	}

	// The listings see the post.
	if got := postListWithToken(t, f.ts, "/api/gallery/posts", aliceToken, nil); len(got) != 1 {
		t.Errorf("gallery/posts = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/gallery/featured", aliceToken, nil); len(got) != 1 {
		t.Errorf("featured = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/gallery/popular", aliceToken, nil); len(got) != 1 {
		t.Errorf("popular = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/users/gallery/posts", bobToken, map[string]any{"userId": aliceID}); len(got) != 1 {
		t.Errorf("users/gallery/posts = %d, want 1", len(got))
	}
	if got := postListWithToken(t, f.ts, "/api/i/gallery/posts", aliceToken, nil); len(got) != 1 {
		t.Errorf("i/gallery/posts = %d, want 1", len(got))
	}

	// Unlike then unlike again is NOT_LIKED.
	if code := postNoContentToken(t, f.ts, "/api/gallery/posts/unlike", bobToken, map[string]any{"postId": postID}); code != http.StatusNoContent {
		t.Fatalf("unlike: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/unlike", bobToken, map[string]any{"postId": postID})
	if resp.StatusCode != http.StatusBadRequest || errCode(out) != "NOT_LIKED" {
		t.Fatalf("second unlike: %d %v", resp.StatusCode, out)
	}
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/show", bobToken, map[string]any{"postId": postID})
	if out["likedCount"] != float64(0) {
		t.Errorf("likedCount after unlike = %v, want 0", out["likedCount"])
	}

	// Foreign delete is NO_SUCH_POST; the owner deletes.
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/delete", bobToken, map[string]any{"postId": postID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_POST" {
		t.Fatalf("foreign delete: %d %v", resp.StatusCode, out)
	}
	if code := postNoContentToken(t, f.ts, "/api/gallery/posts/delete", aliceToken, map[string]any{"postId": postID}); code != http.StatusNoContent {
		t.Fatalf("owner delete: %d", code)
	}
	resp, out = postWithToken(t, f.ts, "/api/gallery/posts/show", bobToken, map[string]any{"postId": postID})
	if resp.StatusCode != http.StatusNotFound || errCode(out) != "NO_SUCH_POST" {
		t.Fatalf("show after delete: %d %v", resp.StatusCode, out)
	}
	_ = aliceID
}
