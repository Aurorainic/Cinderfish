package httpapi_test

import (
	"net/http"
	"testing"
	"time"
)

// TestAdminInviteLifecycle covers admin/invite/create, admin/invite/list and
// admin/invite/delete: the packed InviteCode shape, the type and sort
// filters, and the deletion branches (missing id, already-used ticket).
func TestAdminInviteLifecycle(t *testing.T) {
	ts, _ := adminTestServer(t, "invite")

	// Non-admin credentials are refused before anything else.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/create", "user-token-not-admin", nil); resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Fatalf("create as non-admin = %d, want 401/403", resp.StatusCode)
	}

	// Default count is one; expiresAt accepts RFC3339.
	resp, created := postRawWithToken(t, ts, "/api/admin/invite/create", "admintoken000000", map[string]any{
		"expiresAt": time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create = %d", resp.StatusCode)
	}
	codes, _ := created.([]any)
	if len(codes) != 1 {
		t.Fatalf("create returned %d codes, want 1", len(codes))
	}
	first, _ := codes[0].(map[string]any)
	code1, _ := first["code"].(string)
	if len(code1) != 16 {
		t.Errorf("code length = %d, want 16", len(code1))
	}
	if first["used"] != false || first["usedAt"] != nil {
		t.Errorf("fresh invite used = %v usedAt = %v, want false/nil", first["used"], first["usedAt"])
	}
	if first["expiresAt"] == nil {
		t.Error("expiresAt must round-trip")
	}
	creator, _ := first["createdBy"].(map[string]any)
	if creator == nil || creator["username"] != "root" {
		t.Errorf("createdBy = %v, want the issuing admin", first["createdBy"])
	}

	// A batch creates that many tickets.
	_, batch := postRawWithToken(t, ts, "/api/admin/invite/create", "admintoken000000", map[string]any{"count": 2})
	batchCodes, _ := batch.([]any)
	if len(batchCodes) != 2 {
		t.Fatalf("batch create returned %d codes, want 2", len(batchCodes))
	}

	// An invalid date format answers the upstream error id.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/create", "admintoken000000", map[string]any{
		"expiresAt": "not-a-date",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid expiresAt = %d, want 400", resp.StatusCode)
	}
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/create", "admintoken000000", map[string]any{
		"count": 101,
	}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("count 101 = %d, want 400", resp.StatusCode)
	}

	// list: three tickets. Tickets created in the same millisecond share an
	// id prefix, so their relative order follows the random suffix exactly
	// as upstream's id sort does; only the row count is asserted here.
	list := postListWithToken(t, ts, "/api/admin/invite/list", "admintoken000000", nil)
	if len(list) != 3 {
		t.Fatalf("list = %d rows, want 3", len(list))
	}

	// list: the unused filter matches everything so far.
	unused := postListWithToken(t, ts, "/api/admin/invite/list", "admintoken000000", map[string]any{"type": "unused"})
	if len(unused) != 3 {
		t.Fatalf("unused filter = %d rows, want 3", len(unused))
	}

	// list: limit and offset.
	paged := postListWithToken(t, ts, "/api/admin/invite/list", "admintoken000000", map[string]any{"limit": 2, "offset": 1})
	if len(paged) != 2 {
		t.Fatalf("paged list = %d rows, want 2", len(paged))
	}

	// Redeem one code through the public signup, then the used filter and
	// the usedBy side must reflect it.
	_, signupAny := postRaw(t, ts, "/api/signup", map[string]any{
		"username": "inviteuser1", "password": "password123", "invitationCode": code1,
	})
	signup, _ := signupAny.(map[string]any)
	if _, ok := signup["token"].(string); !ok {
		t.Fatalf("signup with invite failed: %v", signup)
	}
	used := postListWithToken(t, ts, "/api/admin/invite/list", "admintoken000000", map[string]any{"type": "used"})
	if len(used) != 1 {
		t.Fatalf("used filter = %d rows, want 1", len(used))
	}
	usedRow, _ := used[0].(map[string]any)
	if usedRow["code"] != code1 || usedRow["used"] != true || usedRow["usedAt"] == nil {
		t.Errorf("used row = %v, want code1/used/usedAt", usedRow)
	}
	usedBy, _ := usedRow["usedBy"].(map[string]any)
	if usedBy == nil || usedBy["username"] != "inviteuser1" {
		t.Errorf("usedBy = %v, want inviteuser1", usedRow["usedBy"])
	}

	// Deleting the used ticket works (admins may clear used codes), then a
	// second delete answers the upstream not-found id.
	inviteID, _ := usedRow["id"].(string)
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/delete", "admintoken000000", map[string]any{"inviteId": inviteID}); resp.StatusCode != http.StatusOK {
		t.Errorf("delete = %d, want 200", resp.StatusCode)
	}
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/delete", "admintoken000000", map[string]any{"inviteId": inviteID}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", resp.StatusCode)
	}
	after := postListWithToken(t, ts, "/api/admin/invite/list", "admintoken000000", nil)
	if len(after) != 2 {
		t.Errorf("list after delete = %d rows, want 2", len(after))
	}

	// Missing inviteId is a parameter error.
	if resp, _ := postRawWithToken(t, ts, "/api/admin/invite/delete", "admintoken000000", nil); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("delete without inviteId = %d, want 400", resp.StatusCode)
	}
}
