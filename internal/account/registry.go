package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoSuchRegistryKey is returned when a registry key does not exist.
var ErrNoSuchRegistryKey = errors.New("account: no such registry key")

// RegistryEntry is one row of user_registry.
type RegistryEntry struct {
	Key       string
	Scope     []string
	Value     json.RawMessage
	UpdatedAt time.Time
}

// RegistryService stores the per-user client settings Misskey keeps under
// i/registry/*. Entries are keyed by (key, scope) and are private to their
// owner.
type RegistryService struct {
	pool *pgxpool.Pool
}

// NewRegistryService returns a RegistryService over the pool.
func NewRegistryService(pool *pgxpool.Pool) *RegistryService {
	return &RegistryService{pool: pool}
}

// normalizeScope sorts and dedupes a scope so the same logical scope maps to
// one row regardless of the order the client sent.
func normalizeScope(scope []string) []string {
	seen := make(map[string]bool, len(scope))
	out := make([]string, 0, len(scope))
	for _, s := range scope {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// Set stores a value under key within scope.
func (s *RegistryService) Set(ctx context.Context, userID, key string, scope []string, value json.RawMessage) error {
	if len(value) == 0 {
		value = json.RawMessage("null")
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO user_registry (user_id, domain, scope, key, value)
VALUES ($1, '', $2, $3, $4)
ON CONFLICT (user_id, domain, scope, key)
DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, userID, normalizeScope(scope), key, value); err != nil {
		return fmt.Errorf("account: registry set: %w", err)
	}
	return nil
}

// Get returns the value stored under key within scope.
func (s *RegistryService) Get(ctx context.Context, userID, key string, scope []string) (json.RawMessage, error) {
	var value []byte
	err := s.pool.QueryRow(ctx, `
SELECT value FROM user_registry WHERE user_id = $1 AND domain = '' AND scope = $2 AND key = $3
	`, userID, normalizeScope(scope), key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoSuchRegistryKey
	}
	if err != nil {
		return nil, fmt.Errorf("account: registry get: %w", err)
	}
	return json.RawMessage(value), nil
}

// GetDetail returns the value and the time it was last written.
func (s *RegistryService) GetDetail(ctx context.Context, userID, key string, scope []string) (*RegistryEntry, error) {
	entry := &RegistryEntry{Key: key, Scope: normalizeScope(scope)}
	var value []byte
	err := s.pool.QueryRow(ctx, `
SELECT value, updated_at FROM user_registry
WHERE user_id = $1 AND domain = '' AND scope = $2 AND key = $3
	`, userID, entry.Scope, key).Scan(&value, &entry.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoSuchRegistryKey
	}
	if err != nil {
		return nil, fmt.Errorf("account: registry detail: %w", err)
	}
	entry.Value = json.RawMessage(value)
	return entry, nil
}

// Keys lists the keys stored in scope.
func (s *RegistryService) Keys(ctx context.Context, userID string, scope []string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
SELECT key FROM user_registry WHERE user_id = $1 AND domain = '' AND scope = $2 ORDER BY key
	`, userID, normalizeScope(scope))
	if err != nil {
		return nil, fmt.Errorf("account: registry keys: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("account: scan registry key: %w", err)
		}
		out = append(out, key)
	}
	return out, rows.Err()
}

// All returns every key/value pair in scope, the shape i/registry/get-all
// answers with.
func (s *RegistryService) All(ctx context.Context, userID string, scope []string) (map[string]json.RawMessage, error) {
	rows, err := s.pool.Query(ctx, `
SELECT key, value FROM user_registry WHERE user_id = $1 AND domain = '' AND scope = $2 ORDER BY key
	`, userID, normalizeScope(scope))
	if err != nil {
		return nil, fmt.Errorf("account: registry all: %w", err)
	}
	defer rows.Close()

	out := map[string]json.RawMessage{}
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("account: scan registry entry: %w", err)
		}
		out[key] = json.RawMessage(value)
	}
	return out, rows.Err()
}

// Remove deletes one key. Removing a missing key is not an error.
func (s *RegistryService) Remove(ctx context.Context, userID, key string, scope []string) error {
	if _, err := s.pool.Exec(ctx, `
DELETE FROM user_registry WHERE user_id = $1 AND domain = '' AND scope = $2 AND key = $3
	`, userID, normalizeScope(scope), key); err != nil {
		return fmt.Errorf("account: registry remove: %w", err)
	}
	return nil
}

// Clear drops every key in scope.
func (s *RegistryService) Clear(ctx context.Context, userID string, scope []string) error {
	if _, err := s.pool.Exec(ctx, `
DELETE FROM user_registry WHERE user_id = $1 AND domain = '' AND scope = $2
	`, userID, normalizeScope(scope)); err != nil {
		return fmt.Errorf("account: registry clear: %w", err)
	}
	return nil
}
