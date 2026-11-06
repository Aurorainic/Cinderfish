package httpapi_test

import (
	"context"
	"net/http"
	"testing"
)

// TestSignupWithInviteCode walks the invite-only registration through the API:
// a valid code creates the account, the code is single use, and a failed
// attempt releases it again.
func TestSignupWithInviteCode(t *testing.T) {
	ts, srv := adminTestServer(t, "invite")
	pool := srv.TestPool()

	if _, err := pool.Exec(context.Background(),
		`INSERT INTO registration_tickets (id, code) VALUES ('w63t1', 'w63invite')`); err != nil {
		t.Fatalf("seed invite: %v", err)
	}

	resp, created := post(t, ts, "/api/signup", map[string]any{
		"username": "inviteduser", "password": "password123", "invitationCode": "w63invite",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("signup with invite = %d %v", resp.StatusCode, created)
	}
	if token, _ := created["token"].(string); token == "" {
		t.Fatalf("signup returned no token: %v", created)
	}

	// The code is spent.
	resp, body := post(t, ts, "/api/signup", map[string]any{
		"username": "seconduser", "password": "password123", "invitationCode": "w63invite",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("reused invite = %d %v, want 400", resp.StatusCode, body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != "INVALID_INVITE" {
		t.Errorf("reuse error = %v, want INVALID_INVITE", body["error"])
	}

	// A failed attempt (taken username) leaves the code usable.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO registration_tickets (id, code) VALUES ('w63t2', 'w63retry')`); err != nil {
		t.Fatalf("seed retry invite: %v", err)
	}
	resp, body = post(t, ts, "/api/signup", map[string]any{
		"username": "inviteduser", "password": "password123", "invitationCode": "w63retry",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("taken username = %d %v, want 400", resp.StatusCode, body)
	}
	var used bool
	if err := pool.QueryRow(context.Background(),
		`SELECT used_at IS NOT NULL FROM registration_tickets WHERE code = 'w63retry'`).Scan(&used); err != nil {
		t.Fatalf("read retry invite: %v", err)
	}
	if used {
		t.Fatal("the failed attempt consumed the code")
	}
	resp, body = post(t, ts, "/api/signup", map[string]any{
		"username": "retrieduser", "password": "password123", "invitationCode": "w63retry",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("retry with released code = %d %v", resp.StatusCode, body)
	}
}
