package httpapi_test

import (
	"bytes"
	"net/http"
	"testing"
)

// TestDriveFilesUpdate walks drive/files/update: rename, caption, the
// sensitive flag and moving between folders, plus the ownership and
// validation branches.
func TestDriveFilesUpdate(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	resp, file := uploadFile(t, ts.URL, aliceToken, "before.bin", bytes.NewBufferString("drive edit"))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("upload: %d %v", resp.StatusCode, file)
	}
	fileID, _ := file["id"].(string)

	folderResp, folder := postWithToken(t, ts, "/api/drive/folders/create", aliceToken, map[string]any{"name": "w63 folder"})
	if folderResp.StatusCode != http.StatusOK {
		t.Fatalf("folder create: %d %v", folderResp.StatusCode, folder)
	}
	folderID, _ := folder["id"].(string)

	// Rename, caption, sensitive and the folder in one call.
	resp, updated := postWithToken(t, ts, "/api/drive/files/update", aliceToken, map[string]any{
		"fileId": fileID, "name": "after.bin", "comment": "w63 caption",
		"isSensitive": true, "folderId": folderID,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("files/update = %d %v", resp.StatusCode, updated)
	}
	if updated["name"] != "after.bin" || updated["comment"] != "w63 caption" || updated["isSensitive"] != true {
		t.Errorf("updated file = %v/%v/%v", updated["name"], updated["comment"], updated["isSensitive"])
	}

	// The change survives a re-read.
	_, shown := postWithToken(t, ts, "/api/drive/files/show", aliceToken, map[string]any{"fileId": fileID})
	if shown["name"] != "after.bin" {
		t.Errorf("files/show name = %v", shown["name"])
	}

	// A null folderId moves the file back to the root.
	resp, moved := postWithToken(t, ts, "/api/drive/files/update", aliceToken, map[string]any{
		"fileId": fileID, "folderId": nil,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clear folder = %d %v", resp.StatusCode, moved)
	}
	if moved["folderId"] != nil {
		t.Errorf("folderId after clearing = %v, want null", moved["folderId"])
	}

	// Another user cannot touch the file.
	if resp, _ := postWithToken(t, ts, "/api/drive/files/update", bobToken, map[string]any{
		"fileId": fileID, "name": "hijack.bin",
	}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("other user's update = %d, want 404", resp.StatusCode)
	}

	// Unknown file and unknown folder are distinguished.
	if resp, _ := postWithToken(t, ts, "/api/drive/files/update", aliceToken, map[string]any{
		"fileId": "00mw000000000000000", "name": "x",
	}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown file = %d, want 404", resp.StatusCode)
	}
	if resp, body := postWithToken(t, ts, "/api/drive/files/update", aliceToken, map[string]any{
		"fileId": fileID, "folderId": "00mw000000000000001",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("unknown folder = %d %v, want 400", resp.StatusCode, body)
	}

	// An empty name is refused rather than blanking the file.
	if resp, _ := postWithToken(t, ts, "/api/drive/files/update", aliceToken, map[string]any{
		"fileId": fileID, "name": "",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty name = %d, want 400", resp.StatusCode)
	}
}

// TestDriveFilesShowRequiresOwnership proves files/show needs a credential
// and answers only for the owner: another user's file is indistinguishable
// from a missing one, so its metadata does not leak through the fileIds
// other users' public notes carry.
func TestDriveFilesShowRequiresOwnership(t *testing.T) {
	ts := newTestServer(t, "open")
	_, aliceToken := createUser(t, ts, uniqueName(t)+"a")
	_, bobToken := createUser(t, ts, uniqueName(t)+"b")

	respA, fileA := uploadFile(t, ts.URL, aliceToken, "owned.png", bytes.NewBufferString("png bytes"))
	if respA.StatusCode != http.StatusOK {
		t.Fatalf("upload: status %d, body %v", respA.StatusCode, fileA)
	}
	fileID, _ := fileA["id"].(string)

	// Anonymous is 401.
	if resp, _ := post(t, ts, "/api/drive/files/show", map[string]any{"fileId": fileID}); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("anonymous show = %d, want 401", resp.StatusCode)
	}
	// The owner sees the file.
	if resp, shown := postWithToken(t, ts, "/api/drive/files/show", aliceToken, map[string]any{"fileId": fileID}); resp.StatusCode != http.StatusOK || shown["id"] != fileID {
		t.Errorf("owner show = %d %v, want 200", resp.StatusCode, shown["id"])
	}
	// Another user gets the same 404 as a missing file.
	if resp, _ := postWithToken(t, ts, "/api/drive/files/show", bobToken, map[string]any{"fileId": fileID}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign show = %d, want 404", resp.StatusCode)
	}
}
