package account_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/media"
)

// testDrive attaches a local-storage drive service to the account service
// so avatar files can be resolved in tests.
func testDrive(t *testing.T, pool *pgxpool.Pool, svc *account.Service) *media.DriveService {
	t.Helper()
	storage, err := media.NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("local storage: %v", err)
	}
	drive := media.NewDriveService(pool, storage)
	svc.AttachDrive(drive)
	return drive
}

func strPtr(s string) *string { return &s }

func TestGetProfileDefaults(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	name := uniqueUsername(t)
	user, _, err := svc.Register(ctx, account.RegistrationOpen, name, "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	prof, err := svc.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if prof.Name != "" || prof.Description != "" || prof.AvatarURL != "" {
		t.Errorf("fresh profile = %+v, want zero value", prof)
	}

	// A user without a profile row must also read as the zero value
	// instead of failing.
	if _, err := pool.Exec(ctx, `DELETE FROM user_profiles WHERE user_id = $1`, user.ID); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
	prof, err = svc.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetProfile without row: %v", err)
	}
	if prof == nil || *prof != (account.Profile{}) {
		t.Errorf("profile without row = %+v, want zero value", prof)
	}
}

func TestUpdateProfileTextFields(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := svc.UpdateProfile(ctx, user.ID, strPtr("Tuna"), strPtr("self intro"), nil); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	prof, err := svc.GetProfile(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if prof.Name != "Tuna" || prof.Description != "self intro" {
		t.Errorf("profile = %+v, want name/description set", prof)
	}

	// A nil field is left untouched; an empty string clears the field.
	if err := svc.UpdateProfile(ctx, user.ID, strPtr("Maguro"), nil, nil); err != nil {
		t.Fatalf("UpdateProfile partial: %v", err)
	}
	prof, _ = svc.GetProfile(ctx, user.ID)
	if prof.Name != "Maguro" || prof.Description != "self intro" {
		t.Errorf("partial update = %+v, want name changed, description kept", prof)
	}

	if err := svc.UpdateProfile(ctx, user.ID, strPtr(""), strPtr(""), nil); err != nil {
		t.Fatalf("UpdateProfile clear: %v", err)
	}
	prof, _ = svc.GetProfile(ctx, user.ID)
	if prof.Name != "" || prof.Description != "" {
		t.Errorf("cleared profile = %+v, want empty strings", prof)
	}
}

func TestUpdateProfileAvatar(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	drive := testDrive(t, pool, svc)

	other, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t)+"b", "password123", "")
	if err != nil {
		t.Fatalf("Register other: %v", err)
	}

	file, err := drive.Save(ctx, user.ID, "avatar.png", bytes.NewBufferString("png"), "image/png")
	if err != nil {
		t.Fatalf("drive.Save: %v", err)
	}

	if err := svc.UpdateProfile(ctx, user.ID, nil, nil, strPtr(file.ID)); err != nil {
		t.Fatalf("UpdateProfile avatar: %v", err)
	}
	prof, _ := svc.GetProfile(ctx, user.ID)
	if prof.AvatarURL != file.URL {
		t.Errorf("avatarUrl = %q, want %q", prof.AvatarURL, file.URL)
	}

	// An empty avatarId clears the URL.
	if err := svc.UpdateProfile(ctx, user.ID, nil, nil, strPtr("")); err != nil {
		t.Fatalf("UpdateProfile clear avatar: %v", err)
	}
	prof, _ = svc.GetProfile(ctx, user.ID)
	if prof.AvatarURL != "" {
		t.Errorf("avatarUrl after clear = %q, want empty", prof.AvatarURL)
	}

	// Re-setting the avatar keeps the other fields intact.
	if err := svc.UpdateProfile(ctx, user.ID, strPtr("Tuna"), nil, strPtr(file.ID)); err != nil {
		t.Fatalf("UpdateProfile combined: %v", err)
	}
	prof, _ = svc.GetProfile(ctx, user.ID)
	if prof.Name != "Tuna" || prof.AvatarURL != file.URL {
		t.Errorf("combined update = %+v", prof)
	}

	// An unknown file id is an error.
	if err := svc.UpdateProfile(ctx, user.ID, nil, nil, strPtr("nosuchfile")); err == nil {
		t.Error("UpdateProfile with a missing avatar file must fail")
	}

	// The other user's profile must not have been touched.
	otherProf, err := svc.GetProfile(ctx, other.ID)
	if err != nil {
		t.Fatalf("GetProfile other: %v", err)
	}
	if otherProf.AvatarURL != "" || otherProf.Name != "" {
		t.Errorf("other profile = %+v, want untouched", otherProf)
	}
}
