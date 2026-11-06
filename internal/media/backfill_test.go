package media

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

// TestBackfillThumbnails generates a preview for a legacy image, counts a
// corrupt image as a failure, and leaves non-images alone.
func TestBackfillThumbnails(t *testing.T) {
	ctx := context.Background()
	pool := migratePool(t)
	storage, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	drive := NewDriveService(pool, storage)

	// Save writes a thumbnail, so strip both the row and the object to make
	// the file look like one uploaded before thumbnails existed.
	legacy, err := drive.Save(ctx, "u1", "legacy.png", bytes.NewReader(encodePNG(t, 800, 400)), "image/png", nil)
	if err != nil {
		t.Fatalf("save legacy: %v", err)
	}
	if legacy.ThumbnailKey == nil {
		t.Fatalf("save produced no thumbnail to strip")
	}
	if err := storage.Delete(ctx, *legacy.ThumbnailKey); err != nil {
		t.Fatalf("delete thumbnail: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE drive_files SET thumbnail_key = NULL WHERE id = $1`, legacy.ID); err != nil {
		t.Fatalf("strip thumbnail key: %v", err)
	}

	// A corrupt image also has no thumbnail and must be counted, not retried
	// silently.
	broken, err := drive.Save(ctx, "u1", "broken.png", bytes.NewBufferString("not an image"), "image/png", nil)
	if err != nil {
		t.Fatalf("save broken: %v", err)
	}
	if broken.ThumbnailKey != nil {
		t.Fatalf("broken image got a thumbnail")
	}

	// Text is not a candidate: the query never selects it.
	if _, err := drive.Save(ctx, "u1", "notes.txt", bytes.NewBufferString("hello"), "text/plain", nil); err != nil {
		t.Fatalf("save text: %v", err)
	}

	report, err := drive.BackfillThumbnails(ctx, 0)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if report.Scanned != 2 || report.Generated != 1 || report.Failed != 1 || report.Skipped != 0 {
		t.Fatalf("report = %+v, want scanned=2 generated=1 failed=1", report)
	}

	var key *string
	if err := pool.QueryRow(ctx, `SELECT thumbnail_key FROM drive_files WHERE id = $1`, legacy.ID).Scan(&key); err != nil {
		t.Fatalf("read legacy row: %v", err)
	}
	if key == nil || *key == "" {
		t.Fatalf("legacy row still has no thumbnail key")
	}
	rc, err := storage.Get(ctx, *key)
	if err != nil {
		t.Fatalf("stored thumbnail: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		t.Fatalf("read thumbnail: %v", err)
	}
	rc.Close()
	if cfg := decodeConfig(t, buf.Bytes()); cfg.Width != 512 || cfg.Height != 256 {
		t.Errorf("backfilled thumbnail = %dx%d, want 512x256", cfg.Width, cfg.Height)
	}

	// The corrupt file stays without a thumbnail so a later run can retry it
	// after the original is replaced.
	if err := pool.QueryRow(ctx, `SELECT thumbnail_key FROM drive_files WHERE id = $1`, broken.ID).Scan(&key); err != nil {
		t.Fatalf("read broken row: %v", err)
	}
	if key != nil {
		t.Errorf("broken image thumbnail key = %q, want NULL", *key)
	}
}

// TestBackfillThumbnailsLimitAndBackend covers the two guards: the limit keeps
// a run bounded, and a row stored on another backend is skipped rather than
// read from the wrong storage.
func TestBackfillThumbnailsLimitAndBackend(t *testing.T) {
	ctx := context.Background()
	pool := migratePool(t)
	storage, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	drive := NewDriveService(pool, storage)

	first, err := drive.Save(ctx, "u1", "first.png", bytes.NewReader(encodePNG(t, 700, 300)), "image/png", nil)
	if err != nil {
		t.Fatalf("save first: %v", err)
	}
	// A different size keeps the content hash, and therefore the storage key,
	// distinct from first.
	second, err := drive.Save(ctx, "u1", "second.png", bytes.NewReader(encodePNG(t, 640, 480)), "image/png", nil)
	if err != nil {
		t.Fatalf("save second: %v", err)
	}
	for _, id := range []string{first.ID, second.ID} {
		if _, err := pool.Exec(ctx, `UPDATE drive_files SET thumbnail_key = NULL WHERE id = $1`, id); err != nil {
			t.Fatalf("strip %s: %v", id, err)
		}
	}
	// The second row claims another backend, so the configured local storage
	// must not serve it.
	if _, err := pool.Exec(ctx, `UPDATE drive_files SET storage_backend = 's3' WHERE id = $1`, second.ID); err != nil {
		t.Fatalf("mark second: %v", err)
	}

	report, err := drive.BackfillThumbnails(ctx, 1)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if report.Scanned != 1 || report.Generated != 1 || report.Skipped != 0 || report.Failed != 0 {
		t.Fatalf("first run = %+v, want one generated file", report)
	}

	report, err = drive.BackfillThumbnails(ctx, 0)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if report.Scanned != 1 || report.Skipped != 1 || report.Generated != 0 || report.Failed != 0 {
		t.Fatalf("second run = %+v, want the other-backend row skipped", report)
	}

	var key *string
	if err := pool.QueryRow(ctx, `SELECT thumbnail_key FROM drive_files WHERE id = $1`, first.ID).Scan(&key); err != nil {
		t.Fatalf("read first: %v", err)
	}
	if key == nil {
		t.Errorf("limited run did not generate the first thumbnail")
	}
	if err := pool.QueryRow(ctx, `SELECT thumbnail_key FROM drive_files WHERE id = $1`, second.ID).Scan(&key); err != nil {
		t.Fatalf("read second: %v", err)
	}
	if key != nil {
		t.Errorf("other-backend row got a thumbnail: %q", *key)
	}
}

// TestBackfillThumbnailsMissingOriginal counts a row whose object is gone as a
// failure instead of aborting the run.
func TestBackfillThumbnailsMissingOriginal(t *testing.T) {
	ctx := context.Background()
	pool := migratePool(t)
	storage, err := NewLocalStorage(t.TempDir(), "/media")
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	drive := NewDriveService(pool, storage)

	file, err := drive.Save(ctx, "u1", "gone.png", bytes.NewReader(encodePNG(t, 600, 300)), "image/png", nil)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE drive_files SET thumbnail_key = NULL WHERE id = $1`, file.ID); err != nil {
		t.Fatalf("strip: %v", err)
	}
	// Remove both objects so a leftover thumbnail cannot make the check pass
	// for the wrong reason.
	if err := storage.Delete(ctx, *file.ThumbnailKey); err != nil {
		t.Fatalf("delete thumbnail: %v", err)
	}
	if err := storage.Delete(ctx, file.StorageKey); err != nil {
		t.Fatalf("delete original: %v", err)
	}

	report, err := drive.BackfillThumbnails(ctx, 0)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if report.Failed != 1 || report.Generated != 0 {
		t.Fatalf("report = %+v, want one failure", report)
	}
	if _, err := storage.Get(ctx, ThumbnailKeyFor(file.StorageKey, ".png")); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing-original row produced an object: %v", err)
	}
}
