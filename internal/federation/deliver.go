package federation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Deliverer fans activities out to remote inboxes through the
// deliver_queue table, retrying failures with exponential backoff.
//
// The queue is durable and shared, so a restart resumes pending
// deliveries and a second process would not duplicate them. A personal
// instance usually runs one worker; the design does not depend on that.
type Deliverer struct {
	pool   *pgxpool.Pool
	store  *Store
	client *Client
	policy *Policy

	// MaxAttempts bounds retries before a delivery is abandoned.
	MaxAttempts int
	// PollInterval is how often an idle worker checks the queue.
	PollInterval time.Duration

	// wake nudges a sleeping worker after this process enqueues work.
	wake chan struct{}
	// waker notifies workers in other processes when Redis is enabled.
	waker WakeNotifier
}

// WakeNotifier publishes a wakeup so workers in other processes poll the
// queue now instead of at the next tick. It is optional.
type WakeNotifier interface {
	NotifyDelivery(ctx context.Context) error
}

// Delivery is one queued delivery.
type Delivery struct {
	ID         int64
	Payload    json.RawMessage
	InboxURL   string
	ActivityID string
	KeyID      string
	UserID     string
	Attempts   int
}

// NewDeliverer returns a deliverer over the given pool.
func NewDeliverer(pool *pgxpool.Pool, store *Store, client *Client, policy *Policy) *Deliverer {
	return &Deliverer{
		pool:         pool,
		store:        store,
		client:       client,
		policy:       policy,
		MaxAttempts:  12,
		PollInterval: 2 * time.Second,
		wake:         make(chan struct{}, 1),
	}
}

// Enqueue records an activity for delivery to one inbox. The caller
// supplies every inbox separately because fan-out differs per activity.
func (d *Deliverer) Enqueue(ctx context.Context, inboxURL, activityID, keyID, userID string, activity Object) error {
	// Policy is applied at enqueue time as well as at send time: a
	// delivery to a disallowed host is dropped before it reaches the
	// queue, so switching to none stops new work immediately.
	if err := d.policy.CheckURLAllowed(ctx, inboxURL); err != nil {
		return err
	}
	payload, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("federation: encode activity: %w", err)
	}
	if _, err := d.pool.Exec(ctx, `
INSERT INTO deliver_queue (payload, inbox_url, activity_id, key_id, user_id)
VALUES ($1,$2,$3,$4,$5)
	`, payload, inboxURL, nullIfEmpty(activityID), nullIfEmpty(keyID), nullIfEmpty(userID)); err != nil {
		return fmt.Errorf("federation: enqueue delivery: %w", err)
	}

	d.Wake()
	if d.waker != nil {
		ctx := context.Background()
		go func() {
			if err := d.waker.NotifyDelivery(ctx); err != nil {
				logf("federation: delivery wakeup failed: %v", err)
			}
		}()
	}
	return nil
}

// SetWakeNotifier installs the cross-process wakeup publisher. Without
// one, a worker only wakes for work enqueued in its own process.
func (d *Deliverer) SetWakeNotifier(n WakeNotifier) { d.waker = n }

// Wake asks the worker to check the queue now. It never blocks: one
// pending wakeup is enough.
func (d *Deliverer) Wake() {
	select {
	case d.wake <- struct{}{}:
	default:
	}
}

// Pending returns up to limit deliveries that are due now.
func (d *Deliverer) Pending(ctx context.Context, limit int) ([]Delivery, error) {
	rows, err := d.pool.Query(ctx, `
SELECT id, payload, inbox_url, COALESCE(activity_id, ''), COALESCE(key_id, ''),
       COALESCE(user_id, ''), attempts
FROM deliver_queue
WHERE delivered_at IS NULL AND attempts < $1 AND next_attempt_at <= now()
ORDER BY next_attempt_at
LIMIT $2
	`, d.MaxAttempts, limit)
	if err != nil {
		return nil, fmt.Errorf("federation: read queue: %w", err)
	}
	defer rows.Close()

	var out []Delivery
	for rows.Next() {
		var dl Delivery
		if err := rows.Scan(&dl.ID, &dl.Payload, &dl.InboxURL, &dl.ActivityID, &dl.KeyID, &dl.UserID, &dl.Attempts); err != nil {
			return nil, fmt.Errorf("federation: scan queue: %w", err)
		}
		out = append(out, dl)
	}
	return out, rows.Err()
}

// Run processes the queue until ctx is cancelled. It wakes every
// PollInterval, and immediately when Wake is called.
func (d *Deliverer) Run(ctx context.Context) {
	ticker := time.NewTicker(d.PollInterval)
	defer ticker.Stop()
	for {
		if err := d.ProcessOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logf("federation: deliver: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-d.wake:
		case <-ticker.C:
		}
	}
}

// ProcessOnce delivers everything due right now and returns after the
// batch, so tests can drive it deterministically.
func (d *Deliverer) ProcessOnce(ctx context.Context) error {
	batch, err := d.Pending(ctx, 32)
	if err != nil {
		return err
	}
	var firstErr error
	for _, dl := range batch {
		if err := d.deliverOne(ctx, dl); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// deliverOne sends one queued activity and records the outcome.
func (d *Deliverer) deliverOne(ctx context.Context, dl Delivery) error {
	var activity Object
	if err := json.Unmarshal(dl.Payload, &activity); err != nil {
		return d.fail(ctx, dl, fmt.Errorf("decode payload: %w", err))
	}

	// Re-check the policy at send time: the mode may have changed
	// while the item sat in the queue.
	if err := d.policy.CheckURLAllowed(ctx, dl.InboxURL); err != nil {
		// A policy refusal is final, not retryable: the host may never
		// become allowed, and retrying would hammer it.
		return d.abandon(ctx, dl, err)
	}

	if err := d.client.PostActivity(ctx, dl.InboxURL, activity, dl.UserID); err != nil {
		return d.fail(ctx, dl, err)
	}

	if _, err := d.pool.Exec(ctx, `
UPDATE deliver_queue SET delivered_at = now(), last_error = NULL WHERE id = $1
	`, dl.ID); err != nil {
		return fmt.Errorf("federation: mark delivered: %w", err)
	}
	return nil
}

// fail schedules a retry with exponential backoff.
func (d *Deliverer) fail(ctx context.Context, dl Delivery, cause error) error {
	attempts := dl.Attempts + 1
	// 1s, 2s, 4s ... capped at five minutes.
	backoff := time.Second << uint(min(attempts, 8))
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	if _, err := d.pool.Exec(ctx, `
UPDATE deliver_queue SET attempts = $2, next_attempt_at = now() + $3::interval, last_error = $4
WHERE id = $1
	`, dl.ID, attempts, fmt.Sprintf("%d seconds", int(backoff.Seconds())), cause.Error()); err != nil {
		return fmt.Errorf("federation: reschedule delivery: %w", err)
	}
	if attempts >= d.MaxAttempts {
		logf("federation: gave up delivering %s after %d attempts: %v", dl.InboxURL, attempts, cause)
	}
	return nil
}

// abandon marks a delivery permanently failed.
func (d *Deliverer) abandon(ctx context.Context, dl Delivery, cause error) error {
	if _, err := d.pool.Exec(ctx, `
UPDATE deliver_queue SET attempts = $2, last_error = $3, delivered_at = now() WHERE id = $1
	`, dl.ID, d.MaxAttempts, cause.Error()); err != nil {
		return fmt.Errorf("federation: abandon delivery: %w", err)
	}
	return nil
}

// InboxesForUser returns the inbox URLs of every remote follower of a
// local user, deduplicated by shared inbox.
func (s *Store) InboxesForUser(ctx context.Context, userID string, includeLocal bool) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT COALESCE(NULLIF(u.shared_inbox, ''), u.inbox), u.host
FROM followings f JOIN users u ON u.id = f.follower_id
WHERE f.followee_id = $1 AND u.host IS NOT NULL AND u.inbox IS NOT NULL AND u.inbox <> ''
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("federation: follower inboxes: %w", err)
	}
	defer rows.Close()

	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var inbox, host string
		if err := rows.Scan(&inbox, &host); err != nil {
			return nil, err
		}
		if !includeLocal && host == "" {
			continue
		}
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		out = append(out, inbox)
	}
	return out, rows.Err()
}

// MentionedInboxes returns the inboxes of users mentioned in a note.
func (s *Store) MentionedInboxes(ctx context.Context, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT COALESCE(NULLIF(shared_inbox, ''), inbox) FROM users
WHERE id = ANY($1) AND host IS NOT NULL AND inbox IS NOT NULL AND inbox <> ''
	`, userIDs)
	if err != nil {
		return nil, fmt.Errorf("federation: mention inboxes: %w", err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err != nil {
			return nil, err
		}
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		out = append(out, inbox)
	}
	return out, rows.Err()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
