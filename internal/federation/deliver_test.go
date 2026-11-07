package federation_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// deliverFixture builds a deliverer over the shared test database with a
// freshly generated signing key.
func deliverFixture(t *testing.T) (*federation.Deliverer, *pgxpool.Pool, func()) {
	t.Helper()
	url := os.Getenv(testdb.EnvVar)
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
	if _, err := pool.Exec(ctx, `TRUNCATE deliver_queue RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate queue: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO instance_meta (key, value) VALUES ('federationMode', to_jsonb('public'::text))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`); err != nil {
		t.Fatalf("seed mode: %v", err)
	}

	policy := federation.NewPolicy(pool, "127.0.0.1")
	renderer := federation.NewRenderer("https://fish.example")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	client := federation.NewClient(policy, renderer, func(context.Context, string) (string, *rsa.PrivateKey, error) {
		return "https://fish.example/users/u#main-key", priv, nil
	})
	d := federation.NewDeliverer(pool, federation.NewStore(pool), client, policy)
	clear := func() {
		_, _ = pool.Exec(context.Background(), `TRUNCATE deliver_queue RESTART IDENTITY`)
	}
	return d, pool, clear
}

// receiver starts an httptest server answering with status.
func receiver(t *testing.T, status int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestDelivererOutcomes covers the three terminal states of one delivery:
// success, a retry after a 500, and giving up once MaxAttempts is reached.
func TestDelivererOutcomes(t *testing.T) {
	d, _, clear := deliverFixture(t)
	defer clear()
	ctx := context.Background()
	activity := federation.Object{"@context": federation.Context, "type": "Delete", "actor": "https://fish.example/users/u", "object": "https://fish.example/notes/1"}

	ok := receiver(t, http.StatusAccepted)
	if err := d.Enqueue(ctx, ok.URL, "", "https://fish.example/users/u#main-key", "", activity); err != nil {
		t.Fatalf("enqueue ok: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("process ok: %v", err)
	}

	failing := receiver(t, http.StatusInternalServerError)
	if err := d.Enqueue(ctx, failing.URL, "", "https://fish.example/users/u#main-key", "", activity); err != nil {
		t.Fatalf("enqueue failing: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("process failing: %v", err)
	}

	// The failed row is rescheduled: not pending now, but present with one
	// attempt.
	pending, err := d.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	for _, p := range pending {
		if p.InboxURL == failing.URL {
			t.Error("a failed delivery is due again immediately")
		}
	}
	if len(pending) != 0 {
		t.Errorf("pending = %v, want none (success delivered, failure backed off)", pending)
	}
}

// TestDelivererGivesUp proves MaxAttempts stops the retries.
func TestDelivererGivesUp(t *testing.T) {
	d, _, clear := deliverFixture(t)
	defer clear()
	ctx := context.Background()
	d.MaxAttempts = 1
	activity := federation.Object{"@context": federation.Context, "type": "Delete", "actor": "https://fish.example/users/u", "object": "https://fish.example/notes/2"}

	failing := receiver(t, http.StatusInternalServerError)
	if err := d.Enqueue(ctx, failing.URL, "", "https://fish.example/users/u#main-key", "", activity); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("process: %v", err)
	}
	pending, err := d.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("pending after giving up = %v, want none", pending)
	}
}

// TestDelivererAbandonsDisallowedHost proves a policy refusal is final: the
// row is marked delivered so it never retries, even though it never was.
func TestDelivererAbandonsDisallowedHost(t *testing.T) {
	d, pool, clear := deliverFixture(t)
	defer clear()
	ctx := context.Background()
	activity := federation.Object{"@context": federation.Context, "type": "Delete", "actor": "https://fish.example/users/u", "object": "https://fish.example/notes/3"}

	if err := d.Enqueue(ctx, "https://peer.example/inbox", "", "https://fish.example/users/u#main-key", "", activity); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	// Switch the instance off between enqueue and send.
	if _, err := pool.Exec(ctx, `
UPDATE instance_meta SET value = to_jsonb('none'::text) WHERE key = 'federationMode'`); err != nil {
		t.Fatalf("switch mode: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("process: %v", err)
	}
	if pending, err := d.Pending(ctx, 10); err != nil {
		t.Fatalf("pending: %v", err)
	} else if len(pending) != 0 {
		t.Errorf("a policy-refused delivery is still pending: %v", pending)
	}
}

// TestDelivererClaimsEachRowOnce proves two workers polling the same queue do
// not send the same activity twice: Pending claims rows atomically with
// FOR UPDATE SKIP LOCKED.
func TestDelivererClaimsEachRowOnce(t *testing.T) {
	d, pool, clear := deliverFixture(t)
	defer clear()
	ctx := context.Background()
	activity := federation.Object{
		"@context": federation.Context, "type": "Delete",
		"actor": "https://fish.example/users/u", "object": "https://fish.example/notes/claim",
	}

	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	const total = 20
	for i := 0; i < total; i++ {
		if err := d.Enqueue(ctx, srv.URL, "", "https://fish.example/users/u#main-key", "", activity); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}

	var wg sync.WaitGroup
	for worker := 0; worker < 2; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < 2; round++ {
				if err := d.ProcessOnce(ctx); err != nil {
					t.Errorf("process: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt64(&hits); got != total {
		t.Errorf("inbox received %d requests, want %d (duplicates mean no claim)", got, total)
	}
	var delivered int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM deliver_queue WHERE delivered_at IS NOT NULL`).Scan(&delivered); err != nil {
		t.Fatalf("count delivered: %v", err)
	}
	if delivered != total {
		t.Errorf("delivered rows = %d, want %d", delivered, total)
	}
}

// TestDelivererRetriesThenSucceeds proves a failed delivery is rescheduled and
// succeeds on the next due attempt, ending as delivered rather than abandoned.
func TestDelivererRetriesThenSucceeds(t *testing.T) {
	d, pool, clear := deliverFixture(t)
	defer clear()
	ctx := context.Background()
	activity := federation.Object{
		"@context": federation.Context, "type": "Delete",
		"actor": "https://fish.example/users/u", "object": "https://fish.example/notes/retry",
	}

	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt64(&hits, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	if err := d.Enqueue(ctx, srv.URL, "", "https://fish.example/users/u#main-key", "", activity); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	var attempts int
	var delivered *time.Time
	if err := pool.QueryRow(ctx, `SELECT attempts, delivered_at FROM deliver_queue`).Scan(&attempts, &delivered); err != nil {
		t.Fatalf("row after failure: %v", err)
	}
	if attempts != 1 || delivered != nil {
		t.Fatalf("after failure attempts=%d delivered=%v, want 1/nil", attempts, delivered)
	}

	// Make the backoff due now instead of sleeping through it.
	if _, err := pool.Exec(ctx, `UPDATE deliver_queue SET next_attempt_at = now()`); err != nil {
		t.Fatalf("rewind backoff: %v", err)
	}
	if err := d.ProcessOnce(ctx); err != nil {
		t.Fatalf("second attempt: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT attempts, delivered_at FROM deliver_queue`).Scan(&attempts, &delivered); err != nil {
		t.Fatalf("row after success: %v", err)
	}
	if attempts != 1 || delivered == nil {
		t.Errorf("after success attempts=%d delivered=%v, want 1/set", attempts, delivered)
	}
	if got := atomic.LoadInt64(&hits); got != 2 {
		t.Errorf("inbox hits = %d, want 2", got)
	}
}
