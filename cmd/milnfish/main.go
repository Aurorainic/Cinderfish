// Command milnfish runs the MilnFish server: a Go single-binary
// personal federated social instance speaking the Misskey API.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"milnfish/internal/config"
	"milnfish/internal/db"
	"milnfish/internal/httpapi"
	"milnfish/internal/media"
)

func main() {
	configPath := flag.String("config", "", "path to config.toml (optional)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("milnfish: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database is mandatory; fail fast with a clear message.
	pool, err := openDB(ctx, cfg)
	if err != nil {
		log.Fatalf("milnfish: %v", err)
	}
	defer pool.Close()

	version, err := db.Migrate(ctx, pool)
	if err != nil {
		log.Fatalf("milnfish: migrate: %v", err)
	}
	log.Printf("milnfish: schema at version %d", version)

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           httpapi.NewServerWithDB(cfg, pool),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Media storage: local filesystem or an S3 compatible object store.
	storage, err := openStorage(ctx, cfg)
	if err != nil {
		log.Fatalf("milnfish: storage: %v", err)
	}
	srv.Handler.(*httpapi.Server).SetStorage(storage)
	log.Printf("milnfish: media backend is %s", storage.Backend())

	errCh := make(chan error, 1)
	go func() {
		log.Printf("milnfish: listening on %s (domain %s, federation %s, registration %s)",
			cfg.Addr(), cfg.Domain, cfg.FederationMode, cfg.RegistrationMode)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("milnfish: server: %v", err)
		}
	case <-ctx.Done():
		log.Print("milnfish: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("milnfish: shutdown: %v", err)
		}
	}
}

func openDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = 10
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// openStorage builds the media backend named by the configuration.
func openStorage(ctx context.Context, cfg *config.Config) (media.Storage, error) {
	switch cfg.Media.Backend {
	case "s3":
		return media.NewS3Storage(media.S3Options{
			Endpoint:      cfg.Media.S3.Endpoint,
			Bucket:        cfg.Media.S3.Bucket,
			Region:        cfg.Media.S3.Region,
			AccessKey:     cfg.Media.S3.AccessKey,
			SecretKey:     cfg.Media.S3.SecretKey,
			UseSSL:        cfg.Media.S3.UseSSL,
			PublicBaseURL: cfg.Media.S3.PublicBaseURL,
		})
	default:
		return media.NewLocalStorage(cfg.Media.LocalPath, "/media")
	}
}
