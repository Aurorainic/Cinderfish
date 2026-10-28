// Command cinderfish runs the CinderFish server: a Go single-binary
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

	"github.com/Aurorainic/cinderfish/internal/config"
	"github.com/Aurorainic/cinderfish/internal/db"
	"github.com/Aurorainic/cinderfish/internal/httpapi"
	"github.com/Aurorainic/cinderfish/internal/media"
	"github.com/Aurorainic/cinderfish/internal/streaming"
)

func main() {
	configPath := flag.String("config", "", "path to config.toml (optional)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("cinderfish: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database is mandatory; fail fast with a clear message.
	pool, err := openDB(ctx, cfg)
	if err != nil {
		log.Fatalf("cinderfish: %v", err)
	}
	defer pool.Close()

	version, err := db.Migrate(ctx, pool)
	if err != nil {
		log.Fatalf("cinderfish: migrate: %v", err)
	}
	log.Printf("cinderfish: schema at version %d", version)

	api := httpapi.NewServerWithDB(cfg, pool)

	// The Streaming API shares the HTTP server: gorilla/websocket
	// upgrades the /streaming route in place, so one port serves both
	// the RPC endpoints and the WebSocket.
	api.AttachStreaming(log.Printf)

	// Relay events to, and receive them from, other processes of the
	// same instance through PostgreSQL LISTEN/NOTIFY. This costs one
	// NOTIFY per event; it is what makes a future multi-process
	// deployment work without a Redis dependency.
	notifyCtx, stopNotify := context.WithCancel(ctx)
	defer stopNotify()

	bus := api.StreamBus()
	bus.SetPublisher(streaming.NewPublisher(pool))
	go func() {
		if err := streaming.Listen(notifyCtx, pool, bus, log.Printf); err != nil {
			log.Printf("cinderfish: streaming notify listener: %v", err)
		}
	}()

	// ActivityPub: policy, renderer, remote store, inbox and the delivery
	// worker. The worker runs in this process; the queue is a table, so a
	// crash resumes pending deliveries on the next start.
	api.AttachFederation()
	if err := api.ApplyFederationConfig(ctx); err != nil {
		log.Fatalf("cinderfish: %v", err)
	}
	api.StartFederationWorkers(notifyCtx)
	if mode := cfg.FederationMode; mode == "none" {
		log.Printf("cinderfish: federation is off; AP routes answer 403")
	}

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           api,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Media storage: local filesystem or an S3 compatible object store.
	storage, err := openStorage(ctx, cfg)
	if err != nil {
		log.Fatalf("cinderfish: storage: %v", err)
	}
	srv.Handler.(*httpapi.Server).SetStorage(storage)
	log.Printf("cinderfish: media backend is %s", storage.Backend())

	errCh := make(chan error, 1)
	go func() {
		log.Printf("cinderfish: listening on %s (domain %s, federation %s, registration %s)",
			cfg.Addr(), cfg.Domain, cfg.FederationMode, cfg.RegistrationMode)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("cinderfish: server: %v", err)
		}
	case <-ctx.Done():
		log.Print("cinderfish: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("cinderfish: shutdown: %v", err)
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
