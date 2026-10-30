package account_test

import (
	"context"
	"errors"
	"sync"
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

// TestPinNoteConcurrentDistinctNotes pins more notes than the limit from
// concurrent goroutines. The read-check-write must be serialised per user,
// so at most MaxPinnedNotes pins may win and no id may appear twice.
func TestPinNoteConcurrentDistinctNotes(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	notes := note.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	total := account.MaxPinnedNotes + 3
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		txt := "race target"
		n, err := notes.Create(ctx, note.CreateParams{UserID: user.ID, Text: &txt})
		if err != nil {
			t.Fatalf("create note %d: %v", i, err)
		}
		ids = append(ids, n.ID)
	}

	var wg sync.WaitGroup
	errs := make([]error, len(ids))
	start := make(chan struct{})
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			<-start
			errs[i] = svc.PinNote(ctx, user.ID, id)
		}(i, id)
	}
	close(start)
	wg.Wait()

	pinned, err := svc.PinnedNoteIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("PinnedNoteIDs: %v", err)
	}
	if len(pinned) > account.MaxPinnedNotes {
		t.Errorf("pin list = %d entries, want <= %d", len(pinned), account.MaxPinnedNotes)
	}

	seen := make(map[string]bool, len(pinned))
	for _, id := range pinned {
		if seen[id] {
			t.Errorf("pin list contains duplicate id %s: %v", id, pinned)
		}
		seen[id] = true
	}

	succeeded := 0
	for i, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, account.ErrPinLimitExceeded):
		default:
			t.Errorf("PinNote %s = %v, want nil or ErrPinLimitExceeded", ids[i], err)
		}
	}
	if succeeded != len(pinned) {
		t.Errorf("%d pins succeeded but the list holds %d entries", succeeded, len(pinned))
	}
	if succeeded > account.MaxPinnedNotes {
		t.Errorf("%d pins succeeded, want <= %d", succeeded, account.MaxPinnedNotes)
	}
}

// TestPinNoteConcurrentSameNote pins one note from several goroutines at
// once: the idempotence check must hold, leaving a single entry.
func TestPinNoteConcurrentSameNote(t *testing.T) {
	pool := testPool(t)
	svc := account.NewService(pool)
	notes := note.NewService(pool)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, account.RegistrationOpen, uniqueUsername(t), "password123", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	txt := "single pin"
	n, err := notes.Create(ctx, note.CreateParams{UserID: user.ID, Text: &txt})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = svc.PinNote(ctx, user.ID, n.ID)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent PinNote %d = %v, want nil", i, err)
		}
	}
	pinned, err := svc.PinnedNoteIDs(ctx, user.ID)
	if err != nil {
		t.Fatalf("PinnedNoteIDs: %v", err)
	}
	if len(pinned) != 1 || pinned[0] != n.ID {
		t.Errorf("pin list = %v, want exactly [%s]", pinned, n.ID)
	}
}
