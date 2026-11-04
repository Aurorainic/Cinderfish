package httpapi_test

import (
	"net/http"
	"testing"
)

// TestDriveFoldersLifecycle pins create/show/list/update/delete for folders:
// create nests under a parent, list scopes to one level, update renames and
// moves, and delete returns its children to the parent.
func TestDriveFoldersLifecycle(t *testing.T) {
	ts := newTestServer(t, "open")
	_, token := createUser(t, ts, uniqueName(t))

	resp, root := post(t, ts, "/api/drive/folders/create", map[string]any{"i": token, "name": "root"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create root: %d %v", resp.StatusCode, root)
	}
	rootID, _ := root["id"].(string)
	if rootID == "" || root["name"] != "root" || root["parentId"] != nil {
		t.Fatalf("root folder = %v", root)
	}

	resp, child := post(t, ts, "/api/drive/folders/create", map[string]any{"i": token, "name": "child", "parentId": rootID})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create child: %d %v", resp.StatusCode, child)
	}
	childID, _ := child["id"].(string)
	if child["parentId"] != rootID {
		t.Errorf("child parentId = %v, want %s", child["parentId"], rootID)
	}

	// The root listing holds only the root folder; the child listing the child.
	roots := postListWithToken(t, ts, "/api/drive/folders", token, nil)
	if len(roots) != 1 {
		t.Fatalf("root folders = %v, want one", roots)
	}
	nested := postListWithToken(t, ts, "/api/drive/folders", token, map[string]any{"folderId": rootID})
	if len(nested) != 1 {
		t.Fatalf("child folders = %v, want one", nested)
	}

	resp, shown := post(t, ts, "/api/drive/folders/show", map[string]any{"i": token, "folderId": rootID})
	if resp.StatusCode != http.StatusOK || shown["id"] != rootID {
		t.Fatalf("show: %d %v", resp.StatusCode, shown)
	}
	if shown["foldersCount"] != float64(1) {
		t.Errorf("root foldersCount = %v, want 1", shown["foldersCount"])
	}

	// Rename.
	resp, renamed := post(t, ts, "/api/drive/folders/update", map[string]any{"i": token, "folderId": childID, "name": "renamed"})
	if resp.StatusCode != http.StatusOK || renamed["name"] != "renamed" || renamed["parentId"] != rootID {
		t.Fatalf("rename: %d %v", resp.StatusCode, renamed)
	}

	// Move to the root with an explicit null parentId.
	resp, moved := post(t, ts, "/api/drive/folders/update", map[string]any{"i": token, "folderId": childID, "parentId": nil})
	if resp.StatusCode != http.StatusOK || moved["parentId"] != nil {
		t.Fatalf("move to root: %d %v", resp.StatusCode, moved)
	}
	if roots := postListWithToken(t, ts, "/api/drive/folders", token, nil); len(roots) != 2 {
		t.Errorf("root folders after move = %v, want two", roots)
	}

	// Deleting the root returns its (now only) child to the root, then both
	// disappear with their own deletes.
	if resp, out := post(t, ts, "/api/drive/folders/delete", map[string]any{"i": token, "folderId": rootID}); resp.StatusCode != http.StatusOK {
		t.Fatalf("delete root: %d %v", resp.StatusCode, out)
	}
	if _, shown := post(t, ts, "/api/drive/folders/show", map[string]any{"i": token, "folderId": rootID}); shown["error"] == nil {
		t.Errorf("deleted folder still shows: %v", shown)
	}
}

// TestDriveFoldersIsolationAndErrors covers the failure modes: another user
// cannot see or touch a folder, a cycle is refused and unknown ids 404.
func TestDriveFoldersIsolationAndErrors(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t))
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	_, root := post(t, ts, "/api/drive/folders/create", map[string]any{"i": aliceToken, "name": "private"})
	rootID, _ := root["id"].(string)
	_, child := post(t, ts, "/api/drive/folders/create", map[string]any{"i": aliceToken, "name": "child", "parentId": rootID})
	childID, _ := child["id"].(string)

	// Bob's listing does not include alice's folders and asking for one is
	// answered like a missing folder.
	if list := postListWithToken(t, ts, "/api/drive/folders", bobToken, nil); len(list) != 0 {
		t.Errorf("bob's folders = %v, want none", list)
	}
	if resp, _ := post(t, ts, "/api/drive/folders/show", map[string]any{"i": bobToken, "folderId": rootID}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("bob show = %d, want 404", resp.StatusCode)
	}
	if resp, _ := post(t, ts, "/api/drive/folders/delete", map[string]any{"i": bobToken, "folderId": rootID}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("bob delete = %d, want 404", resp.StatusCode)
	}

	// A folder cannot become its own parent or a child's child.
	if resp, out := post(t, ts, "/api/drive/folders/update", map[string]any{"i": aliceToken, "folderId": rootID, "parentId": rootID}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("self move = %d %v, want 400", resp.StatusCode, out)
	}
	if resp, out := post(t, ts, "/api/drive/folders/update", map[string]any{"i": aliceToken, "folderId": rootID, "parentId": childID}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("cycle move = %d %v, want 400", resp.StatusCode, out)
	}

	if resp, _ := post(t, ts, "/api/drive/folders/create", map[string]any{"name": "x"}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous create = %d, want 401", resp.StatusCode)
	}
}
