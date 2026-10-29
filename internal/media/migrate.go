package media

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationOptions configures a drive migration between two storage
// backends. The bytes keep their content-hash key, so notes and file URLs
// do not change; only drive_files.storage_backend is updated.
type MigrationOptions struct {
	Source Storage
	Target Storage
	// DryRun counts what would move without copying or updating rows.
	DryRun bool
	// DeleteSource removes each source object after every row that shares
	// it has been migrated.
	DeleteSource bool
	// Limit caps the number of rows; zero means all.
	Limit int
	// Logf receives one line per migrated file; nil disables logging.
	Logf func(format string, args ...any)
}

// MigrationReport summarises a run.
type MigrationReport struct {
	Scanned  int
	Migrated int
	Skipped  int
	Failed   int
	Failures []string
}

// MigrateDrive moves every object recorded against the source backend to
// the target backend. Failures are collected per file so one unreadable
// object does not abort the rest.
func MigrateDrive(ctx context.Context, pool *pgxpool.Pool, opts MigrationOptions) (MigrationReport, error) {
	var report MigrationReport
	if opts.Source == nil || opts.Target == nil {
		return report, errors.New("media: migrate: source and target are required")
	}
	if pool == nil {
		return report, errors.New("media: migrate: database is required")
	}

	query := `
SELECT id, storage_key, content_type, size FROM drive_files
WHERE storage_backend = $1 ORDER BY id`
	args := []any{opts.Source.Backend()}
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return report, fmt.Errorf("media: migrate: list files: %w", err)
	}

	type item struct {
		id, key, contentType string
		size                 int64
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.key, &it.contentType, &it.size); err != nil {
			rows.Close()
			return report, fmt.Errorf("media: migrate: scan file: %w", err)
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return report, fmt.Errorf("media: migrate: list files: %w", err)
	}

	report.Scanned = len(items)
	keys := make(map[string]struct{}, len(items))

	for _, it := range items {
		if opts.DryRun {
			opts.logf("would migrate %s (%s, %d bytes)", it.key, it.contentType, it.size)
			report.Migrated++
			keys[it.key] = struct{}{}
			continue
		}

		rc, err := opts.Source.Get(ctx, it.key)
		if err != nil {
			report.fail(it.key, fmt.Errorf("read: %w", err))
			continue
		}
		_, putErr := opts.Target.Put(ctx, it.key, rc, it.size, it.contentType)
		closeErr := rc.Close()
		if putErr == nil {
			putErr = closeErr
		}
		if putErr != nil {
			report.fail(it.key, fmt.Errorf("write: %w", putErr))
			continue
		}

		if _, err := pool.Exec(ctx,
			`UPDATE drive_files SET storage_backend = $1 WHERE id = $2`,
			opts.Target.Backend(), it.id); err != nil {
			report.fail(it.key, fmt.Errorf("update row: %w", err))
			continue
		}

		opts.logf("migrated %s", it.key)
		report.Migrated++
		keys[it.key] = struct{}{}
	}

	if opts.DeleteSource && !opts.DryRun && report.Failed == 0 {
		for key := range keys {
			if err := opts.Source.Delete(ctx, key); err != nil {
				report.fail(key, fmt.Errorf("delete source: %w", err))
			}
		}
	}

	return report, nil
}

func (o MigrationOptions) logf(format string, args ...any) {
	if o.Logf != nil {
		o.Logf(format, args...)
	}
}

func (r *MigrationReport) fail(key string, err error) {
	r.Failed++
	r.Failures = append(r.Failures, fmt.Sprintf("%s: %v", key, err))
}
