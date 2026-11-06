// Command cinderfish-media-migrate moves stored media between the local
// filesystem and an S3 compatible bucket, updating drive_files rows so the
// rest of the app keeps serving the same keys.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/media"
)

func main() {
	configPath := flag.String("config", "", "path to config.toml (optional)")
	to := flag.String("to", "", "target backend: local or s3")
	toLocalPath := flag.String("to-local-path", "", "target directory for --to local")
	dryRun := flag.Bool("dry-run", false, "report what would move without changing anything")
	deleteSource := flag.Bool("delete-source", false, "delete objects from the source after a fully successful run")
	limit := flag.Int("limit", 0, "process at most N files (0 = all)")
	backfill := flag.Bool("backfill-thumbnails", false, "generate missing thumbnails for stored images instead of migrating")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("cinderfish-media-migrate: %v", err)
	}

	if *backfill {
		if *to != "" {
			log.Fatal("cinderfish-media-migrate: --backfill-thumbnails cannot be combined with --to")
		}
	} else {
		switch *to {
		case "local", "s3":
		case "":
			log.Fatal("cinderfish-media-migrate: --to is required (local or s3) unless --backfill-thumbnails is set")
		default:
			log.Fatalf("cinderfish-media-migrate: unknown target %q", *to)
		}
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("cinderfish-media-migrate: connect: %v", err)
	}
	defer pool.Close()

	if _, err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("cinderfish-media-migrate: migrate: %v", err)
	}

	if *backfill {
		storage, err := openSource(cfg)
		if err != nil {
			log.Fatalf("cinderfish-media-migrate: storage: %v", err)
		}
		drive := media.NewDriveService(pool, storage)
		report, err := drive.BackfillThumbnails(ctx, *limit)
		if err != nil {
			log.Fatalf("cinderfish-media-migrate: backfill thumbnails: %v", err)
		}
		fmt.Printf("scanned=%d generated=%d skipped=%d failed=%d\n",
			report.Scanned, report.Generated, report.Skipped, report.Failed)
		if report.Failed > 0 {
			os.Exit(1)
		}
		return
	}

	source, err := openSource(cfg)
	if err != nil {
		log.Fatalf("cinderfish-media-migrate: source: %v", err)
	}
	target, err := openTarget(cfg, *to, *toLocalPath)
	if err != nil {
		log.Fatalf("cinderfish-media-migrate: target: %v", err)
	}
	log.Printf("cinderfish-media-migrate: %s → %s", source.Backend(), target.Backend())
	report, err := media.MigrateDrive(ctx, pool, media.MigrationOptions{
		Source:       source,
		Target:       target,
		DryRun:       *dryRun,
		DeleteSource: *deleteSource,
		Limit:        *limit,
		Logf: func(format string, args ...any) {
			log.Printf(format, args...)
		},
	})
	if err != nil {
		log.Fatalf("cinderfish-media-migrate: %v", err)
	}

	fmt.Printf("scanned=%d migrated=%d failed=%d\n", report.Scanned, report.Migrated, report.Failed)
	for _, failure := range report.Failures {
		fmt.Fprintf(os.Stderr, "failed: %s\n", failure)
	}
	if report.Failed > 0 {
		os.Exit(1)
	}
}

func openSource(cfg *config.Config) (media.Storage, error) {
	if strings.EqualFold(cfg.Media.Backend, "s3") {
		return newS3(cfg.Media.S3)
	}
	return media.NewLocalStorage(cfg.Media.LocalPath, "/media")
}

func openTarget(cfg *config.Config, to, localPath string) (media.Storage, error) {
	switch to {
	case "local":
		if strings.TrimSpace(localPath) == "" {
			return nil, fmt.Errorf("--to-local-path is required for --to local")
		}
		return media.NewLocalStorage(localPath, "/media")
	default:
		return newS3(cfg.Media.S3)
	}
}

func newS3(s config.S3Settings) (media.Storage, error) {
	if strings.TrimSpace(s.Endpoint) == "" || strings.TrimSpace(s.Bucket) == "" {
		return nil, fmt.Errorf("S3 endpoint and bucket are required")
	}
	return media.NewS3Storage(media.S3Options{
		Endpoint:      s.Endpoint,
		Bucket:        s.Bucket,
		Region:        s.Region,
		AccessKey:     s.AccessKey,
		SecretKey:     s.SecretKey,
		UseSSL:        s.UseSSL,
		PublicBaseURL: s.PublicBaseURL,
		CACertPath:    s.CACert,
	})
}
