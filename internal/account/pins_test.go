package account_test

import (
	"context"
	"testing"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// TestPinnedNoteIDsOrderAndIdempotence covers the pin list storage: pins
// stack newest-first, pinning twice does not duplicate, and unpinning is
// a no-op when the note is not pinned.
func TestPinnedNoteIDsOrderAndIdempotence(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	notes := note.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// The pinned_note_ids column must exist on an already-migrated
	// database and Migrate must be replayable.
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("replay Migrate: %v", err)
	}

	ids, err := svc.PinnedNoteIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("PinnedNoteIDs empty: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("fresh pin list = %v, want empty", ids)
	}

	var created []string
	for _, text := range []string{"one", "two", "three"} {
		txt := text
		n, err := notes.Create(ctx, note.CreateParams{UserID: user.ID, Text: &txt})
		if err != nil {
			t.Fatalf("create note %q: %v", text, err)
		}
		created = append(created, n.ID)
	}

	for _, id := range created {
		if err := svc.PinNote(ctx, user.ID, id); err != nil {
			t.Fatalf("PinNote %s: %v", id, err)
		}
	}

	// Most recently pinned first.
	got, err := svc.PinnedNoteIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("PinnedNoteIDs: %v", err)
	}
	want := []string{created[2], created[1], created[0]}
	if len(got) != len(want) {
		t.Fatalf("pin list = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pin list[%d] = %s, want %s", i, got[i], want[i])
		}
	}

	// Pinning again is a no-op.
	if err := svc.PinNote(ctx, user.ID, created[1]); err != nil {
		t.Fatalf("re-pin: %v", err)
	}
	got, _ = svc.PinnedNoteIDs(ctx, user.ID)
	if len(got) != 3 {
		t.Errorf("pin list after re-pin = %v, want 3 entries", got)
	}

	// Unpinning removes the note wherever it sits.
	if err := svc.UnpinNote(ctx, user.ID, created[1]); err != nil {
		t.Fatalf("UnpinNote: %v", err)
	}
	got, _ = svc.PinnedNoteIDs(ctx, user.ID)
	if len(got) != 2 || got[0] != created[2] || got[1] != created[0] {
		t.Errorf("pin list after unpin = %v, want [%s %s]", got, created[2], created[0])
	}

	// Unpinning a note that does not exist fails like Misskey's
	// NO_SUCH_NOTE: the note id is validated before the array remove.
	if err := svc.UnpinNote(ctx, user.ID, "nosuchnote"); err != account.ErrNoSuchPinTarget {
		t.Errorf("UnpinNote unknown = %v, want ErrNoSuchPinTarget", err)
	}
}

func TestPinNoteRejectsForeignAndMissingNotes(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	notes := note.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	other, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t)+"b", "password123", "")
	if err != nil {
		t.Fatalf("Register other: %v", err)
	}

	txt := "not mine"
	foreign, err := notes.Create(ctx, note.CreateParams{UserID: other.ID, Text: &txt})
	if err != nil {
		t.Fatalf("create foreign note: %v", err)
	}

	if err := svc.PinNote(ctx, user.ID, foreign.ID); err == nil {
		t.Error("pinning another user's note must fail")
	} else if err != account.ErrNotOwnNote {
		t.Errorf("error = %v, want ErrNotOwnNote", err)
	}

	if err := svc.PinNote(ctx, user.ID, "nosuchnote"); err == nil {
		t.Error("pinning a missing note must fail")
	} else if err != account.ErrNoSuchPinTarget {
		t.Errorf("error = %v, want ErrNoSuchPinTarget", err)
	}
}

func TestPinNoteLimit(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	notes := note.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	var ids []string
	for i := 0; i < account.MaxPinnedNotes+1; i++ {
		txt := "pin target"
		n, err := notes.Create(ctx, note.CreateParams{UserID: user.ID, Text: &txt})
		if err != nil {
			t.Fatalf("create note %d: %v", i, err)
		}
		ids = append(ids, n.ID)
	}

	for _, id := range ids[:account.MaxPinnedNotes] {
		if err := svc.PinNote(ctx, user.ID, id); err != nil {
			t.Fatalf("PinNote %s: %v", id, err)
		}
	}
	if err := svc.PinNote(ctx, user.ID, ids[account.MaxPinnedNotes]); err != account.ErrPinLimitExceeded {
		t.Errorf("over-limit pin = %v, want ErrPinLimitExceeded", err)
	}
	pinned, err := svc.PinnedNoteIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("PinnedNoteIDs: %v", err)
	}
	if len(pinned) != account.MaxPinnedNotes {
		t.Errorf("pin list = %d entries, want %d", len(pinned), account.MaxPinnedNotes)
	}

	// Unpinning frees a slot.
	if err := svc.UnpinNote(ctx, user.ID, ids[0]); err != nil {
		t.Fatalf("UnpinNote: %v", err)
	}
	if err := svc.PinNote(ctx, user.ID, ids[account.MaxPinnedNotes]); err != nil {
		t.Fatalf("PinNote after unpin: %v", err)
	}
}
