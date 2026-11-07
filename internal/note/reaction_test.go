package note

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/testdb"
)

// TestMain serialises this package against the other database-backed test
// packages, which share one test database.
func TestMain(m *testing.M) {
	os.Exit(testdb.Run(m))
}

// notificationFixture owns two users and the notifications seeded for them.
// Every id carries a prefix derived from the test name so this package can
// share the test database with other packages whose tests run in parallel.
type notificationFixture struct {
	pool     *pgxpool.Pool
	prefix   string
	readerID string
	otherID  string
}

func newNotificationFixture(t *testing.T) *notificationFixture {
	t.Helper()
	url := os.Getenv("CINDERFISH_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("CINDERFISH_TEST_DATABASE_URL is not set; skipping database test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if _, err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sum := sha256.Sum256([]byte(t.Name()))
	prefix := "nt" + hex.EncodeToString(sum[:])[:10]
	f := &notificationFixture{
		pool:     pool,
		prefix:   prefix,
		readerID: prefix + "-reader",
		otherID:  prefix + "-other",
	}
	// Deleting the users cascades to their notifications. Clearing first
	// recovers from an earlier run that failed before its cleanup ran, and
	// the cleanup is registered before the inserts for the same reason.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id LIKE $1`, prefix+"-%"); err != nil {
		t.Fatalf("clear stale fixture rows: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`DELETE FROM users WHERE id LIKE $1`, prefix+"-%"); err != nil {
			t.Errorf("cleanup fixture users: %v", err)
		}
	})
	for _, id := range []string{f.readerID, f.otherID} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO users (id, username, username_lower) VALUES ($1, $2, lower($2))`, id, id); err != nil {
			t.Fatalf("insert user %s: %v", id, err)
		}
	}
	return f
}

func (f *notificationFixture) service() *NotificationService {
	return NewNotificationService(f.pool)
}

func (f *notificationFixture) seed(t *testing.T, id, recipientID, notifType string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(),
		`INSERT INTO notifications (id, recipient_id, type) VALUES ($1, $2, $3)`,
		f.prefix+"-"+id, recipientID, notifType); err != nil {
		t.Fatalf("insert notification %s: %v", id, err)
	}
}

func (f *notificationFixture) list(t *testing.T, include, exclude []string) []*Notification {
	t.Helper()
	list, err := f.service().List(context.Background(), f.readerID, 20, "", "", include, exclude, false)
	if err != nil {
		t.Fatalf("List(include=%v, exclude=%v): %v", include, exclude, err)
	}
	return list
}

func (f *notificationFixture) ids(list []*Notification) []string {
	out := make([]string, 0, len(list))
	for _, n := range list {
		out = append(out, strings.TrimPrefix(n.ID, f.prefix+"-"))
	}
	return out
}

func TestNotificationListIncludeTypes(t *testing.T) {
	f := newNotificationFixture(t)
	f.seed(t, "01", f.readerID, "reaction")
	f.seed(t, "02", f.readerID, "mention")
	f.seed(t, "03", f.readerID, "renote")
	f.seed(t, "04", f.readerID, "reaction")

	if got, want := f.ids(f.list(t, []string{"reaction"}, nil)), []string{"04", "01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes reaction = %v, want %v", got, want)
	}
	if got, want := f.ids(f.list(t, []string{"mention"}, nil)), []string{"02"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes mention = %v, want %v", got, want)
	}
	if got, want := f.ids(f.list(t, []string{"reaction", "renote"}, nil)), []string{"04", "03", "01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("includeTypes reaction+renote = %v, want %v", got, want)
	}

	// An explicitly empty includeTypes means no filter, exactly like nil:
	// Misskey checks length > 0 before filtering (NotificationService.ts
	// L302), so [] returns everything.
	if got := f.list(t, []string{}, nil); len(got) != 4 {
		t.Errorf("empty includeTypes returned %v, want all four", f.ids(got))
	}
	// nil leaves the type unfiltered.
	if got := f.list(t, nil, nil); len(got) != 4 {
		t.Errorf("nil includeTypes returned %d notifications, want 4", len(got))
	}
}

func TestNotificationListExcludeTypes(t *testing.T) {
	f := newNotificationFixture(t)
	f.seed(t, "01", f.readerID, "reaction")
	f.seed(t, "02", f.readerID, "mention")
	f.seed(t, "03", f.readerID, "renote")

	if got, want := f.ids(f.list(t, nil, []string{"reaction"})), []string{"03", "02"}; !reflect.DeepEqual(got, want) {
		t.Errorf("excludeTypes reaction = %v, want %v", got, want)
	}
	if got, want := f.ids(f.list(t, nil, []string{"mention", "renote"})), []string{"01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("excludeTypes mention+renote = %v, want %v", got, want)
	}
}

// TestNotificationListIncludeAndExclude pins Misskey's include-wins
// semantics (packages/backend/src/core/NotificationService.ts L302-306):
// a non-empty includeTypes takes precedence and excludeTypes is ignored.
func TestNotificationListIncludeAndExclude(t *testing.T) {
	f := newNotificationFixture(t)
	f.seed(t, "01", f.readerID, "reaction")
	f.seed(t, "02", f.readerID, "mention")
	f.seed(t, "03", f.readerID, "renote")
	f.seed(t, "04", f.readerID, "reaction")

	// Include covers both types, so the exclude clause is ignored.
	got := f.ids(f.list(t, []string{"reaction", "mention"}, []string{"mention"}))
	want := []string{"04", "02", "01"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("include reaction+mention, exclude mention = %v, want %v (include wins)", got, want)
	}

	// Exclude still applies on its own.
	got = f.ids(f.list(t, nil, []string{"renote"}))
	if want := []string{"04", "02", "01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("exclude renote alone = %v, want %v", got, want)
	}
}

// TestNotificationListSinceIdAscending pins Misskey's makePaginationQuery
// semantics (packages/backend/src/core/QueryService.ts L58-60): a sinceId
// without untilId pages upward in ascending id order, the direction used by
// "load newer" cursors.
func TestNotificationListSinceIdAscending(t *testing.T) {
	f := newNotificationFixture(t)
	f.seed(t, "01", f.readerID, "mention")
	f.seed(t, "02", f.readerID, "reaction")
	f.seed(t, "03", f.readerID, "mention")
	f.seed(t, "04", f.readerID, "renote")

	// sinceId alone: the page holds everything after it, oldest first.
	got, err := f.service().List(context.Background(), f.readerID, 20, f.prefix+"-01", "", nil, nil, false)
	if err != nil {
		t.Fatalf("List(sinceId): %v", err)
	}
	if want := []string{"02", "03", "04"}; !reflect.DeepEqual(f.ids(got), want) {
		t.Errorf("sinceId-only List = %v, want %v (ascending)", f.ids(got), want)
	}

	// sinceId with untilId pages between the two cursors, newest first.
	got, err = f.service().List(context.Background(), f.readerID, 20, f.prefix+"-01", f.prefix+"-04", nil, nil, false)
	if err != nil {
		t.Fatalf("List(sinceId,untilId): %v", err)
	}
	if want := []string{"03", "02"}; !reflect.DeepEqual(f.ids(got), want) {
		t.Errorf("sinceId+untilId List = %v, want %v (descending)", f.ids(got), want)
	}
}

func TestNotificationMarkRead(t *testing.T) {
	f := newNotificationFixture(t)
	ctx := context.Background()
	f.seed(t, "01", f.readerID, "reaction")
	f.seed(t, "02", f.readerID, "mention")
	f.seed(t, "03", f.otherID, "reaction")

	list := f.list(t, nil, nil)
	if got := f.ids(list); !reflect.DeepEqual(got, []string{"02", "01"}) {
		t.Fatalf("List returned %v, want the reader's two notifications", got)
	}
	ids := make([]string, 0, len(list))
	for _, n := range list {
		if n.IsRead {
			t.Fatalf("notification %s is read before marking", n.ID)
		}
		ids = append(ids, n.ID)
	}

	if err := f.service().MarkRead(ctx, f.readerID, ids); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	for _, n := range f.list(t, nil, nil) {
		if !n.IsRead {
			t.Errorf("notification %s still unread after MarkRead", n.ID)
		}
	}

	// The other user's notification is untouched, even when its id is
	// passed for the wrong recipient.
	other, err := f.service().Get(ctx, f.prefix+"-03")
	if err != nil {
		t.Fatalf("Get other: %v", err)
	}
	if other.IsRead {
		t.Errorf("notification of another recipient was marked read")
	}
	if err := f.service().MarkRead(ctx, f.readerID, []string{f.prefix + "-03"}); err != nil {
		t.Fatalf("MarkRead other: %v", err)
	}
	other, err = f.service().Get(ctx, f.prefix+"-03")
	if err != nil {
		t.Fatalf("Get other: %v", err)
	}
	if other.IsRead {
		t.Errorf("MarkRead crossed the recipient boundary")
	}

	// An empty id list is a no-op, not an error.
	if err := f.service().MarkRead(ctx, f.readerID, nil); err != nil {
		t.Errorf("MarkRead with no ids: %v", err)
	}
}

// CreateForLocal mirrors Misskey's local-only notification fan-out
// (core/NoteCreateService.ts L863/L884: a target with a host gets no
// notification row; ActivityPub delivers its copy instead).
func TestNotificationCreateForLocal(t *testing.T) {
	f := newNotificationFixture(t)
	ctx := context.Background()

	remoteID := f.prefix + "-remote"
	if _, err := f.pool.Exec(ctx,
		`INSERT INTO users (id, username, username_lower, host) VALUES ($1, $2, lower($2), 'remote.example')`,
		remoteID, remoteID); err != nil {
		t.Fatalf("insert remote user: %v", err)
	}

	id, created, err := f.service().CreateForLocal(ctx, f.otherID, "reply", &f.readerID, nil, nil)
	if err != nil {
		t.Fatalf("CreateForLocal(local): %v", err)
	}
	if !created || id == "" {
		t.Fatalf("CreateForLocal(local) = (%q, %v), want a created row", id, created)
	}
	if _, err := f.service().Get(ctx, id); err != nil {
		t.Errorf("created notification is not readable: %v", err)
	}

	_, created, err = f.service().CreateForLocal(ctx, remoteID, "reply", &f.readerID, nil, nil)
	if err != nil {
		t.Fatalf("CreateForLocal(remote): %v", err)
	}
	if created {
		t.Error("CreateForLocal recorded a notification for a remote user")
	}
	var count int
	if err := f.pool.QueryRow(ctx,
		`SELECT count(*) FROM notifications WHERE recipient_id = $1`, remoteID).Scan(&count); err != nil {
		t.Fatalf("count remote notifications: %v", err)
	}
	if count != 0 {
		t.Errorf("remote recipient has %d notification rows, want 0", count)
	}
}
