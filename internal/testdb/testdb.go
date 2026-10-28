// Package testdb serialises database-backed test packages that share one
// PostgreSQL database.
//
// internal/account and internal/httpapi both truncate the same tables in
// their setup, and `go test ./...` runs packages in parallel, so one
// package can delete rows another package is still using. A session level
// advisory lock held for the lifetime of each test binary keeps the
// database-backed packages one at a time.
package testdb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// EnvVar names the test database URL. Database tests skip themselves when
// it is unset.
const EnvVar = "MILNFISH_TEST_DATABASE_URL"

// lockKey identifies the advisory lock guarding the shared test database.
const lockKey int64 = 0x6d696c6e66697348

// Run executes the package's tests while holding the shared-database
// lock, and returns the process exit code.
//
// Call it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
//
// With no test database configured every database test skips itself, so
// no lock is taken.
func Run(m *testing.M) int {
	url := os.Getenv(EnvVar)
	if url == "" {
		return m.Run()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testdb: connecting to the test database: %v", err)
		fmt.Fprintln(os.Stderr)
		return 1
	}
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		fmt.Fprintf(os.Stderr, "testdb: locking the test database: %v", err)
		fmt.Fprintln(os.Stderr)
		return 1
	}
	fmt.Fprintln(os.Stderr, "testdb: holding the shared test database lock")

	code := m.Run()

	// The lock is released when the session ends anyway; unlocking first
	// lets the next package start without waiting for this process.
	unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer unlockCancel()
	if _, err := conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey); err != nil {
		fmt.Fprintf(os.Stderr, "testdb: unlocking the test database: %v", err)
		fmt.Fprintln(os.Stderr)
	}
	return code
}
