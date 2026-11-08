package account_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/db"
)

// testPool returns a pool against the test database, skipping the test
// when no database is available.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("CINDERFISH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_DATABASE_URL is not set; skipping database test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := db.Migrate(context.Background(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Start from a clean slate so reruns do not trip unique constraints.
	if _, err := pool.Exec(context.Background(), `
TRUNCATE users, user_profiles, access_tokens, registration_tickets, signins, ip_bans, signin_failures
RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

// uniqueUsername returns a username that will not collide with other tests.
func uniqueUsername(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(t.Name()))
	return "u" + hex.EncodeToString(sum[:])[:12]
}

func TestRegisterOpen(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	user, token, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(token) != 16 {
		t.Errorf("native token length = %d, want 16", len(token))
	}
	if user.Username != name {
		t.Errorf("username = %q, want %q", user.Username, name)
	}

	got, err := svc.GetByUsername(ctx, name)
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.ID != user.ID {
		t.Errorf("id = %q, want %q", got.ID, user.ID)
	}
}

func TestRegisterDuplicateUsername(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	if _, _, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", ""); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if _, _, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", ""); err == nil {
		t.Fatal("registering a duplicate username should fail")
	}
}

func TestRegisterAdminModeRefuses(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)

	_, _, err := svc.Register(context.Background(), account.RegistrationAdmin, uniqueUsername(t), "password123", "")
	if err == nil {
		t.Fatal("admin mode must refuse self-service signup")
	}
	var regErr *account.RegistrationError
	if !asRegistrationError(err, &regErr) {
		t.Fatalf("error = %v, want a RegistrationError", err)
	}
	if regErr.Code != "REGISTRATION_DISABLED" {
		t.Errorf("code = %q, want REGISTRATION_DISABLED", regErr.Code)
	}
}

func TestRegisterInviteMode(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	// Without a code, invite mode refuses.
	if _, _, err := svc.Register(ctx, account.RegistrationInvite, uniqueUsername(t), "password123", ""); err == nil {
		t.Error("invite mode must refuse signup without a code")
	}

	invited, err := svc.CreateInvite(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	code := invited.Code
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	name := uniqueUsername(t)
	if _, _, err := svc.Register(ctx, account.RegistrationInvite, name, "password123", code); err != nil {
		t.Fatalf("Register with invite: %v", err)
	}

	// The same code must not be reusable.
	if _, _, err := svc.Register(ctx, account.RegistrationInvite, uniqueUsername(t)+"b", "password123", code); err == nil {
		t.Error("an invitation code must be single use")
	}

	// A failed signup releases the code: claim happens before the account is
	// created, so a taken username must not burn the invitation.
	retryInvited, err := svc.CreateInvite(ctx, "", nil)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	retryCode := retryInvited.Code
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if _, _, err := svc.Register(ctx, account.RegistrationInvite, name, "password123", retryCode); err == nil {
		t.Fatal("registering a taken username must fail")
	}
	var used bool
	if err := pool.QueryRow(context.Background(),
		`SELECT used_at IS NOT NULL FROM registration_tickets WHERE code = $1`, retryCode).Scan(&used); err != nil {
		t.Fatalf("read invite: %v", err)
	}
	if used {
		t.Error("a failed signup consumed the invitation code")
	}
	if _, _, err := svc.Register(ctx, account.RegistrationInvite, uniqueUsername(t)+"c", "password123", retryCode); err != nil {
		t.Errorf("reusing the released code: %v", err)
	}
}

func TestInviteManagement(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	admin, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register admin: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET is_admin = true WHERE id = $1`, admin.ID); err != nil {
		t.Fatalf("promote: %v", err)
	}

	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	invited, err := svc.CreateInvite(ctx, admin.ID, &expires)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	code := invited.Code
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if len(code) != 16 {
		t.Errorf("invite code length = %d, want 16", len(code))
	}

	// The creator is recorded on the ticket.
	invites, err := svc.ListInvites(ctx, account.InviteFilter{Type: "all", Sort: "+createdAt"})
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("invite count = %d, want 1", len(invites))
	}
	inv := invites[0]
	if inv.Code != code {
		t.Errorf("code = %q, want %q", inv.Code, code)
	}
	if inv.Used {
		t.Error("a fresh invite must not be used")
	}
	if inv.ExpiresAt == nil || !inv.ExpiresAt.Equal(expires) {
		t.Errorf("expiresAt = %v, want %v", inv.ExpiresAt, expires)
	}
	if inv.CreatedBy == nil || inv.CreatedBy.ID != admin.ID {
		t.Fatalf("createdBy = %+v, want admin %s", inv.CreatedBy, admin.ID)
	}
	if inv.CreatedBy.Username != admin.Username {
		t.Errorf("createdBy username = %q, want %q", inv.CreatedBy.Username, admin.Username)
	}

	// Filter by type sees the same ticket as unused, none as used.
	used, err := svc.ListInvites(ctx, account.InviteFilter{Type: "used"})
	if err != nil {
		t.Fatalf("ListInvites used: %v", err)
	}
	if len(used) != 0 {
		t.Errorf("used filter returned %d invites, want 0", len(used))
	}

	// A signup claims the ticket and the used-by side of the pack follows.
	claimant, _, err := svc.Register(ctx, account.RegistrationInvite, uniqueUsername(t)+"x", "password123", code)
	if err != nil {
		t.Fatalf("Register with invite: %v", err)
	}
	invites, err = svc.ListInvites(ctx, account.InviteFilter{Type: "used"})
	if err != nil {
		t.Fatalf("ListInvites used: %v", err)
	}
	if len(invites) != 1 || !invites[0].Used {
		t.Fatalf("used filter = %+v, want one used invite", invites)
	}
	if invites[0].UsedBy == nil || invites[0].UsedBy.ID != claimant.ID {
		t.Errorf("usedBy = %+v, want claimant %s", invites[0].UsedBy, claimant.ID)
	}
	if invites[0].UsedAt == nil {
		t.Error("usedAt must be set once claimed")
	}

	// An expired filter only matches tickets whose deadline has passed.
	expiredInv, err := svc.CreateInvite(ctx, admin.ID, ptrTime(time.Now().Add(-time.Hour)))
	if err != nil {
		t.Fatalf("CreateInvite expired: %v", err)
	}
	expiredCode := expiredInv.Code
	if err != nil {
		t.Fatalf("CreateInvite expired: %v", err)
	}
	expired, err := svc.ListInvites(ctx, account.InviteFilter{Type: "expired"})
	if err != nil {
		t.Fatalf("ListInvites expired: %v", err)
	}
	if len(expired) != 1 || expired[0].Code != expiredCode {
		t.Fatalf("expired filter = %+v, want the expired invite", expired)
	}

	// Deleting works, and a second delete reports the ticket gone.
	if err := svc.DeleteInvite(ctx, invites[0].ID); err != nil {
		t.Fatalf("DeleteInvite: %v", err)
	}
	if err := svc.DeleteInvite(ctx, invites[0].ID); err == nil {
		t.Error("deleting a missing invite must fail")
	}
	all, err := svc.ListInvites(ctx, account.InviteFilter{Type: "all"})
	if err != nil {
		t.Fatalf("ListInvites after delete: %v", err)
	}
	if len(all) != 1 || all[0].Code != expiredCode {
		t.Errorf("remaining invites = %+v, want only the expired one", all)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestReservedUsernames(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	svc.SetReservedUsernames(func() []string { return []string{"Admin", "root"} })

	// Signup refuses a preserved name, case-insensitively (Misskey's
	// SignupService compares lowercased entries).
	_, _, err := svc.Register(ctx, account.RegistrationOpen, "aDmIn", "password123", "")
	if err == nil {
		t.Fatal("signup with a preserved username must fail")
	}
	var regErr *account.RegistrationError
	if !asRegistrationError(err, &regErr) || regErr.Code != "USED_USERNAME" {
		t.Fatalf("error = %v, want USED_USERNAME", err)
	}

	// username/available agrees.
	available, err := svc.UsernameAvailable(ctx, "ROOT")
	if err != nil {
		t.Fatalf("UsernameAvailable: %v", err)
	}
	if available {
		t.Error("a preserved username must report unavailable")
	}

	// A name not on the list still works.
	if _, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", ""); err != nil {
		t.Errorf("registering an unreserved name: %v", err)
	}

	// The admin create path deliberately ignores the list: reserving a name
	// and then issuing it is how an administrator uses it.
	if _, err := svc.Create(ctx, "root", "password123"); err != nil {
		t.Errorf("admin Create of a preserved name: %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	if _, _, err := svc.Register(ctx, account.RegistrationOpen, name, "correct horse", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}

	user, err := svc.Authenticate(ctx, name, "correct horse")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if user.Username != name {
		t.Errorf("username = %q", user.Username)
	}

	if _, err := svc.Authenticate(ctx, name, "wrong"); err == nil {
		t.Error("Authenticate should reject a wrong password")
	}
	if _, err := svc.Authenticate(ctx, "nosuchuser", "x"); err == nil {
		t.Error("Authenticate should reject an unknown user")
	}
}

func TestTokenResolve(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	tokens := account.NewTokenService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	user, nativeToken, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resolved, accessToken, err := tokens.Resolve(ctx, nativeToken)
	if err != nil {
		t.Fatalf("Resolve native token: %v", err)
	}
	if resolved.ID != user.ID {
		t.Errorf("resolved user = %q, want %q", resolved.ID, user.ID)
	}
	if accessToken != nil {
		t.Error("a native token must not resolve to an access token")
	}

	miauthToken := "miauth-" + uniqueUsername(t)
	if _, err := tokens.Issue(ctx, user.ID, miauthToken, "session", "app", "desc", "icon", []string{"read"}); err != nil {
		t.Fatalf("Issue: %v", err)
	}
	resolved, accessToken, err = tokens.Resolve(ctx, miauthToken)
	if err != nil {
		t.Fatalf("Resolve miauth token: %v", err)
	}
	if resolved.ID != user.ID {
		t.Errorf("resolved user = %q", resolved.ID)
	}
	if accessToken == nil || accessToken.UserID != user.ID {
		t.Errorf("access token = %+v", accessToken)
	}

	if _, _, err := tokens.Resolve(ctx, "definitely-not-a-token"); err == nil {
		t.Error("Resolve should reject an unknown token")
	}
}

func TestUsernameAvailable(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	available, err := svc.UsernameAvailable(ctx, name)
	if err != nil {
		t.Fatalf("UsernameAvailable: %v", err)
	}
	if !available {
		t.Fatal("a fresh username must be available")
	}

	if _, _, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", ""); err != nil {
		t.Fatalf("Register: %v", err)
	}
	available, err = svc.UsernameAvailable(ctx, name)
	if err != nil {
		t.Fatalf("UsernameAvailable: %v", err)
	}
	if available {
		t.Error("a taken username must not be available")
	}
}

func asRegistrationError(err error, target **account.RegistrationError) bool {
	if e, ok := err.(*account.RegistrationError); ok {
		*target = e
		return true
	}
	return false
}

func TestSetAdminAndSuspend(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, "mod", "password123", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.IsAdmin {
		t.Fatal("a registered user must not be an administrator")
	}

	if err := svc.SetAdmin(ctx, user.ID, true); err != nil {
		t.Fatalf("set admin: %v", err)
	}
	updated, err := svc.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !updated.IsAdmin {
		t.Error("user is not an administrator after SetAdmin")
	}

	if err := svc.SetSuspended(ctx, user.ID, true); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	updated, _ = svc.GetByID(ctx, user.ID)
	if !updated.IsSuspended {
		t.Error("user is not suspended after SetSuspended")
	}

	if err := svc.SetAdmin(ctx, "missing", true); err != account.ErrUserNotFound {
		t.Errorf("SetAdmin on a missing user = %v, want ErrUserNotFound", err)
	}
}

func TestIPBans(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	admin, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t)+"a", "password123", "")
	if err != nil {
		t.Fatalf("register admin: %v", err)
	}

	// A bare address is normalised and matches exactly.
	ban, err := svc.CreateIPBan(ctx, " 192.0.2.7 ", "spam", nil, admin.ID)
	if err != nil {
		t.Fatalf("CreateIPBan: %v", err)
	}
	if ban.IP != "192.0.2.7" {
		t.Errorf("normalised ip = %q, want 192.0.2.7", ban.IP)
	}
	if banned, err := svc.IsIPBanned(ctx, "192.0.2.7"); err != nil || !banned {
		t.Errorf("exact match = %v %v, want banned", banned, err)
	}
	if banned, _ := svc.IsIPBanned(ctx, "192.0.2.8"); banned {
		t.Error("a neighbouring address must not be banned")
	}

	// A CIDR rule covers its network only.
	if _, err := svc.CreateIPBan(ctx, "198.51.100.0/24", "", nil, admin.ID); err != nil {
		t.Fatalf("CreateIPBan CIDR: %v", err)
	}
	if banned, _ := svc.IsIPBanned(ctx, "198.51.100.23"); !banned {
		t.Error("an address inside the CIDR must be banned")
	}
	if banned, _ := svc.IsIPBanned(ctx, "198.51.101.1"); banned {
		t.Error("an address outside the CIDR must not be banned")
	}

	// An expired rule no longer applies, but stays listed.
	past := time.Now().Add(-time.Minute)
	if _, err := svc.CreateIPBan(ctx, "203.0.113.9", "", &past, admin.ID); err != nil {
		t.Fatalf("CreateIPBan expired: %v", err)
	}
	if banned, _ := svc.IsIPBanned(ctx, "203.0.113.9"); banned {
		t.Error("an expired rule must not ban")
	}
	list, err := svc.ListIPBans(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListIPBans: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("bans = %d, want 3", len(list))
	}

	// Deletion works and a second delete reports the row gone.
	if err := svc.DeleteIPBan(ctx, ban.ID); err != nil {
		t.Fatalf("DeleteIPBan: %v", err)
	}
	if err := svc.DeleteIPBan(ctx, ban.ID); err != account.ErrIPBanNotFound {
		t.Errorf("second delete = %v, want ErrIPBanNotFound", err)
	}

	// Invalid rules are refused.
	for _, rule := range []string{"", "not-an-ip", "300.1.1.1", "10.0.0.0/33"} {
		if _, err := svc.CreateIPBan(ctx, rule, "", nil, admin.ID); err == nil {
			t.Errorf("rule %q must be refused", rule)
		}
	}
}

func TestAdminSigninHistory(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t)+"s", "password123", "")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := svc.RecordSignin(ctx, user.ID, "192.0.2.10", true); err != nil {
		t.Fatalf("record ok: %v", err)
	}
	if err := svc.RecordSignin(ctx, user.ID, "192.0.2.10", false); err != nil {
		t.Fatalf("record fail: %v", err)
	}

	records, err := svc.AdminSigninHistory(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("AdminSigninHistory: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	// Newest first; the failed attempt was recorded last.
	if records[0].Success {
		t.Error("newest record should be the failed attempt")
	}
	if records[0].Username != user.Username || records[0].IP != "192.0.2.10" {
		t.Errorf("record = %+v, want user %s from 192.0.2.10", records[0], user.Username)
	}
}
