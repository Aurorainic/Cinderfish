package media

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
)

// BackfillReport summarises one thumbnail backfill run.
type BackfillReport struct {
	// Scanned counts image rows that have no thumbnail.
	Scanned int
	// Generated counts rows that now have a stored thumbnail.
	Generated int
	// Skipped counts rows this run could not or should not touch: a file
	// stored on another backend, or an image small enough that the upload
	// path would have stored none.
	Skipped int
	// Failed counts rows whose original could not be read or decoded.
	Failed int
}

// BackfillThumbnails generates previews for stored image files that have none.
// Uploads made before thumbnails existed, and images that failed to decode at
// upload time, keep thumbnail_key NULL; this walks them oldest first so a run
// with a limit makes progress across runs.
//
// limit 0 means every candidate row.
func (s *DriveService) BackfillThumbnails(ctx context.Context, limit int) (BackfillReport, error) {
	var report BackfillReport
	if limit < 0 {
		limit = 0
	}
	rows, err := s.pool.Query(ctx, `
SELECT id, content_type, storage_key, storage_backend
FROM drive_files
WHERE thumbnail_key IS NULL
  AND content_type IN ('image/jpeg','image/pjpeg','image/png','image/gif')
ORDER BY created_at
LIMIT NULLIF($1, 0)
`, limit)
	if err != nil {
		return report, fmt.Errorf("media: list files without thumbnails: %w", err)
	}
	type candidate struct {
		id, contentType, storageKey, backend string
	}
	candidates := make([]candidate, 0, 16)
	for rows.Next() {
		var c candidate
		if err := rows.Scan(&c.id, &c.contentType, &c.storageKey, &c.backend); err != nil {
			rows.Close()
			return report, fmt.Errorf("media: scan file without thumbnail: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return report, fmt.Errorf("media: read files without thumbnails: %w", err)
	}
	rows.Close()

	for _, c := range candidates {
		report.Scanned++
		if c.backend != s.storage.Backend() {
			report.Skipped++
			log.Printf("media: backfill %s: stored on %s, configured backend is %s",
				c.id, c.backend, s.storage.Backend())
			continue
		}
		rc, err := s.storage.Get(ctx, c.storageKey)
		if err != nil {
			report.Failed++
			log.Printf("media: backfill %s: read original: %v", c.id, err)
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rc, s.maxSize+1))
		rc.Close()
		if err != nil {
			report.Failed++
			log.Printf("media: backfill %s: read original: %v", c.id, err)
			continue
		}
		if int64(len(data)) > s.maxSize {
			report.Failed++
			log.Printf("media: backfill %s: original exceeds the %d byte limit", c.id, s.maxSize)
			continue
		}
		thumb, ext, err := GenerateThumbnail(data, c.contentType)
		if err != nil {
			report.Failed++
			log.Printf("media: backfill %s: %v", c.id, err)
			continue
		}
		if thumb == nil {
			// Unreachable for the query's four types; kept so a future type
			// that decodes but does not shrink is skipped rather than retried
			// forever.
			report.Skipped++
			continue
		}
		thumbKey := ThumbnailKeyFor(c.storageKey, ext)
		if _, err := s.storage.Put(ctx, thumbKey, bytes.NewReader(thumb), int64(len(thumb)),
			thumbnailContentType(ext)); err != nil {
			report.Failed++
			log.Printf("media: backfill %s: store thumbnail: %v", c.id, err)
			continue
		}
		if _, err := s.pool.Exec(ctx, `
UPDATE drive_files SET thumbnail_key = $2 WHERE id = $1
`, c.id, thumbKey); err != nil {
			report.Failed++
			log.Printf("media: backfill %s: record thumbnail: %v", c.id, err)
			continue
		}
		report.Generated++
	}
	return report, nil
}
