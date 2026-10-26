package account_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/account"
	"milnfish/internal/db"
)

// testPool returns a pool against the test database, skipping the test
// when no database is available.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("MILNFISH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MILNFISH_TEST_DATABASE_URL is not set; skipping database test")
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
TRUNCATE users, user_profiles, access_tokens, registration_tickets, signins
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

	code, err := svc.CreateInvite(ctx, nil)
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
