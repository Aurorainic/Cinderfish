package federation_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

	policy := federation.NewPolicy(pool, "fish.example")
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
