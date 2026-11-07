// Package webhook stores per-user outbound webhooks and delivers note events.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// EventTypes is Misskey's webhook event enum. Only note, reply and renote are
// produced by this port; the rest are accepted so a Firefish client's payload
// still validates, but they never fire.
var EventTypes = []string{
	"mention", "unfollow", "follow", "followed", "note", "reply", "renote", "reaction",
}

// ErrNotFound is returned when a webhook id does not belong to the caller.
var ErrNotFound = errors.New("webhook: not found")

// Webhook is one registered endpoint.
type Webhook struct {
	ID           string
	UserID       string
	Name         string
	URL          string
	Secret       string
	On           []string
	Active       bool
	CreatedAt    time.Time
	LatestSentAt *time.Time
	LatestStatus *int
}

// Service reads, writes and delivers webhooks.
type Service struct {
	pool   *pgxpool.Pool
	client *http.Client
	// AllowPrivateTargets disables the public-address validation, for
	// tests that deliver against a loopback HTTP server.
	AllowPrivateTargets bool
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:   pool,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

const webhookColumns = `id, user_id, name, url, secret, events, active, created_at, latest_sent_at, latest_status`

func scanWebhook(row pgx.Row) (*Webhook, error) {
	var w Webhook
	err := row.Scan(&w.ID, &w.UserID, &w.Name, &w.URL, &w.Secret, &w.On, &w.Active,
		&w.CreatedAt, &w.LatestSentAt, &w.LatestStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("webhook: scan: %w", err)
	}
	return &w, nil
}

// ErrInvalidURL rejects a webhook target the instance must not call.
var ErrInvalidURL = errors.New("webhook: URL is not a reachable public http(s) address")

// Create stores a webhook for the user.
func (s *Service) Create(ctx context.Context, userID, name, url, secret string, on []string) (*Webhook, error) {
	if !s.AllowPrivateTargets {
		if err := ValidateURL(ctx, url); err != nil {
			return nil, err
		}
	}
	row := s.pool.QueryRow(ctx, `
INSERT INTO webhooks (id, user_id, name, url, secret, events)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING `+webhookColumns, idgen.New(), userID, name, url, secret, on)
	w, err := scanWebhook(row)
	if err != nil {
		return nil, fmt.Errorf("webhook: create: %w", err)
	}
	return w, nil
}

// List returns the user's webhooks, newest first.
func (s *Service) List(ctx context.Context, userID string) ([]*Webhook, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+webhookColumns+` FROM webhooks WHERE user_id = $1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("webhook: list: %w", err)
	}
	defer rows.Close()
	var out []*Webhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// Show loads one of the user's webhooks.
func (s *Service) Show(ctx context.Context, userID, id string) (*Webhook, error) {
	row := s.pool.QueryRow(ctx, `
SELECT `+webhookColumns+` FROM webhooks WHERE id = $1 AND user_id = $2`, id, userID)
	return scanWebhook(row)
}

// Update rewrites the editable fields of one webhook.
func (s *Service) Update(ctx context.Context, userID, id, name, url, secret string, on []string, active bool) (*Webhook, error) {
	if !s.AllowPrivateTargets {
		if err := ValidateURL(ctx, url); err != nil {
			return nil, err
		}
	}
	row := s.pool.QueryRow(ctx, `
UPDATE webhooks SET name = $3, url = $4, secret = $5, events = $6, active = $7
WHERE id = $1 AND user_id = $2
RETURNING `+webhookColumns, id, userID, name, url, secret, on, active)
	w, err := scanWebhook(row)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return w, err
}

// Delete removes one of the user's webhooks. It reports whether a row existed.
func (s *Service) Delete(ctx context.Context, userID, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM webhooks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return false, fmt.Errorf("webhook: delete: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// Dispatch delivers one event to every active hook of the user that subscribed
// to the type. Delivery is fire-and-forget: the note request must not wait for
// a third-party server, and the latest status is written back when it answers.
func (s *Service) Dispatch(userID, eventType string, body any) {
	rows, err := s.pool.Query(context.Background(), `
SELECT `+webhookColumns+` FROM webhooks
WHERE user_id = $1 AND active AND events @> ARRAY[$2]::text[]`, userID, eventType)
	if err != nil {
		return
	}
	defer rows.Close()
	var hooks []*Webhook
	for rows.Next() {
		w, err := scanWebhook(rows)
		if err != nil {
			continue
		}
		hooks = append(hooks, w)
	}
	for _, hook := range hooks {
		go s.deliver(hook, userID, eventType, body)
	}
}

func (s *Service) deliver(hook *Webhook, userID, eventType string, body any) {
	payload, err := json.Marshal(map[string]any{
		"hookId":    hook.ID,
		"userId":    userID,
		"eventId":   idgen.New(),
		"createdAt": time.Now().UTC().Format(time.RFC3339Nano),
		"type":      eventType,
		"body":      body,
	})
	if err != nil {
		return
	}

	// The URL was validated at registration, but the row may predate that
	// check or the DNS may have changed; re-checking at delivery keeps the
	// endpoint from probing internal addresses.
	if !s.AllowPrivateTargets {
		if err := ValidateURL(context.Background(), hook.URL); err != nil {
			s.record(hook.ID, 0)
			return
		}
	}

	req, err := http.NewRequest(http.MethodPost, hook.URL, bytes.NewReader(payload))
	if err != nil {
		s.record(hook.ID, 0)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Cinderfish-Hooks")
	req.Header.Set("X-Firefish-Hook-Id", hook.ID)
	req.Header.Set("X-Firefish-Hook-Secret", hook.Secret)

	resp, err := s.client.Do(req)
	if err != nil {
		s.record(hook.ID, 1)
		return
	}
	defer resp.Body.Close()
	s.record(hook.ID, resp.StatusCode)
}

// ValidateURL rejects a webhook target that is not an http(s) URL on a
// public host. Loopback, private, link-local and unresolved names are
// refused: a registered webhook POSTs on every note, so a targeted URL
// would turn latest_status into a port scanner against the instance's own
// network. The check runs at registration and again at delivery.
func ValidateURL(ctx context.Context, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return ErrInvalidURL
	}
	host := parsed.Hostname()
	// A literal IP needs no lookup.
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return ErrInvalidURL
		}
		return nil
	}
	if host == "localhost" {
		return ErrInvalidURL
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(lookupCtx, host)
	if err != nil {
		return ErrInvalidURL
	}
	for _, addr := range ips {
		if isPrivateIP(addr.IP) {
			return ErrInvalidURL
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// record writes the delivery result back; a missing row is ignored.
func (s *Service) record(id string, status int) {
	_, _ = s.pool.Exec(context.Background(), `
UPDATE webhooks SET latest_sent_at = now(), latest_status = $2 WHERE id = $1`, id, status)
}
