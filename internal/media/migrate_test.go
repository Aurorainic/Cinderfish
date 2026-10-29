package media

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// labelledStorage renames a backend so a test can move between two local
// directories while the drive_files CHECK constraint still sees
// 'local' and 's3'.
type labelledStorage struct {
	Storage
	name string
}

func (l labelledStorage) Backend() string { return l.name }

func migratePool(t *testing.T) *pgxpool.Pool {
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
	if _, err := pool.Exec(ctx, `TRUNCATE users, drive_files CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES ('u1', 'alice', 'x')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return pool
}

func putFile(t *testing.T, ctx context.Context, s Storage, key, body string) {
	t.Helper()
	if _, err := s.Put(ctx, key, bytes.NewReader([]byte(body)), int64(len(body)), "text/plain"); err != nil {
		t.Fatalf("put %s: %v", key, err)
	}
}

func readFile(t *testing.T, ctx context.Context, s Storage, key string) string {
	t.Helper()
	rc, err := s.Get(ctx, key)
	if err != nil {
		t.Fatalf("get %s: %v", key, err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	return string(data)
}

func TestMigrateDriveMovesObjectsAndRows(t *testing.T) {
	pool := migratePool(t)
	ctx := context.Background()

	src, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	dst, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	source := labelledStorage{Storage: src, name: "local"}
	target := labelledStorage{Storage: dst, name: "s3"}

	key := path.Join("aa", "bb", "deadbeef")
	putFile(t, ctx, source, key, "hello media")
	if _, err := pool.Exec(ctx, `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend)
VALUES ('f1', 'u1', 'hello.txt', 'text/plain', 11, 'deadbeef', $1, 'local')`, key); err != nil {
		t.Fatalf("insert file: %v", err)
	}

	report, err := MigrateDrive(ctx, pool, MigrationOptions{
		Source: source, Target: target, DeleteSource: true,
	})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if report.Scanned != 1 || report.Migrated != 1 || report.Failed != 0 {
		t.Fatalf("report = %+v", report)
	}

	if got := readFile(t, ctx, target, key); got != "hello media" {
		t.Errorf("target content = %q", got)
	}
	if _, err := source.Get(ctx, key); err == nil {
		t.Error("source object still exists after --delete-source")
	}

	var backend string
	if err := pool.QueryRow(ctx, `SELECT storage_backend FROM drive_files WHERE id = 'f1'`).Scan(&backend); err != nil {
		t.Fatal(err)
	}
	if backend != "s3" {
		t.Errorf("storage_backend = %q, want s3", backend)
	}
}

func TestMigrateDriveDryRunChangesNothing(t *testing.T) {
	pool := migratePool(t)
	ctx := context.Background()

	src, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	dst, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatal(err)
	}
	source := labelledStorage{Storage: src, name: "local"}
	target := labelledStorage{Storage: dst, name: "s3"}

	key := path.Join("cc", "dd", "cafebabe")
	putFile(t, ctx, source, key, "dry run")
	if _, err := pool.Exec(ctx, `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend)
VALUES ('f2', 'u1', 'dry.txt', 'text/plain', 7, 'cafebabe', $1, 'local')`, key); err != nil {
		t.Fatalf("insert file: %v", err)
	}

	report, err := MigrateDrive(ctx, pool, MigrationOptions{Source: source, Target: target, DryRun: true})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if report.Scanned != 1 || report.Migrated != 1 {
		t.Fatalf("report = %+v", report)
	}
	if _, err := target.Get(ctx, key); err == nil {
		t.Error("dry run wrote to the target")
	}

	var backend string
	if err := pool.QueryRow(ctx, `SELECT storage_backend FROM drive_files WHERE id = 'f2'`).Scan(&backend); err != nil {
		t.Fatal(err)
	}
	if backend != "local" {
		t.Errorf("storage_backend = %q, want local after dry run", backend)
	}
}

func TestMigrateDriveReportsMissingObject(t *testing.T) {
	pool := migratePool(t)
	ctx := context.Background()

	src, _ := NewLocalStorage(t.TempDir(), "/media")
	dst, _ := NewLocalStorage(t.TempDir(), "/media")
	source := labelledStorage{Storage: src, name: "local"}
	target := labelledStorage{Storage: dst, name: "s3"}

	if _, err := pool.Exec(ctx, `
INSERT INTO drive_files (id, user_id, name, content_type, size, md5, storage_key, storage_backend)
VALUES ('f3', 'u1', 'gone.txt', 'text/plain', 4, 'gone', 'zz/zz/gone', 'local')`); err != nil {
		t.Fatalf("insert file: %v", err)
	}

	report, err := MigrateDrive(ctx, pool, MigrationOptions{Source: source, Target: target})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if report.Failed != 1 || len(report.Failures) != 1 {
		t.Fatalf("report = %+v, want one failure", report)
	}
}
