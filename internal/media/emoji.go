package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmojiExists is returned when a custom emoji with that name is already
// registered.
var ErrEmojiExists = errors.New("media: emoji already exists")

// ErrEmojiNotFound is returned when an emoji id does not exist.
var ErrEmojiNotFound = errors.New("media: no such emoji")

// Emoji is one row of the custom emoji registry.
type Emoji struct {
	ID          string
	Name        string
	Host        *string
	Category    *string
	Aliases     []string
	URL         string
	IsSensitive bool
	LocalOnly   bool
	CreatedAt   time.Time
}

// EmojiService manages the instance's custom emoji registry. Remote emoji
// rows (host set) are reserved for federation import; only local rows are
// served by the emojis endpoint today.
type EmojiService struct {
	pool *pgxpool.Pool
}

// NewEmojiService returns an EmojiService over the pool.
func NewEmojiService(pool *pgxpool.Pool) *EmojiService {
	return &EmojiService{pool: pool}
}

// Add registers a local custom emoji pointing at an already stored file URL.
func (s *EmojiService) Add(ctx context.Context, name, url string, category *string, aliases []string, sensitive bool) (*Emoji, error) {
	if aliases == nil {
		aliases = []string{}
	}
	raw, err := json.Marshal(aliases)
	if err != nil {
		return nil, fmt.Errorf("media: encode emoji aliases: %w", err)
	}

	e := &Emoji{
		ID:          idgen.New(),
		Name:        name,
		Category:    category,
		Aliases:     aliases,
		URL:         url,
		IsSensitive: sensitive,
		LocalOnly:   true,
		CreatedAt:   time.Now(),
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO emojis (id, name, category, aliases, url, is_sensitive, local_only)
VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, e.ID, e.Name, e.Category, raw, e.URL, e.IsSensitive, e.LocalOnly)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmojiExists
		}
		return nil, fmt.Errorf("media: add emoji: %w", err)
	}
	return e, nil
}

// ListLocal returns the local emojis in Misskey's category/name order.
func (s *EmojiService) ListLocal(ctx context.Context) ([]*Emoji, error) {
	rows, err := s.pool.Query(ctx, emojiSelect+`
WHERE host IS NULL
ORDER BY category ASC NULLS FIRST, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("media: list emojis: %w", err)
	}
	defer rows.Close()
	return scanEmojis(rows)
}

// EmojiQuery is the paging parameter set of admin/emoji/list.
type EmojiQuery struct {
	Query   string
	Limit   int
	SinceID string
	UntilID string
}

// ListAdmin returns every emoji (local and remote) for the admin console.
func (s *EmojiService) ListAdmin(ctx context.Context, q EmojiQuery) ([]*Emoji, error) {
	if q.Limit <= 0 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	query := emojiSelect + ` WHERE TRUE`
	args := []any{}
	if q.Query != "" {
		args = append(args, "%"+q.Query+"%")
		query += fmt.Sprintf(" AND (name ILIKE $%d OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(aliases) a WHERE a ILIKE $%d))", len(args), len(args))
	}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		query += fmt.Sprintf(" AND id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		query += fmt.Sprintf(" AND id < $%d", len(args))
	}
	query += fmt.Sprintf(" ORDER BY id DESC LIMIT %d", q.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("media: list emojis: %w", err)
	}
	defer rows.Close()
	return scanEmojis(rows)
}

// Delete removes an emoji row by id.
func (s *EmojiService) Delete(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM emojis WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("media: delete emoji: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrEmojiNotFound
	}
	return nil
}

const emojiSelect = `
SELECT id, name, host, category, aliases, url, is_sensitive, local_only, created_at
FROM emojis`

func scanEmojis(rows pgx.Rows) ([]*Emoji, error) {
	out := []*Emoji{}
	for rows.Next() {
		e := &Emoji{}
		var aliases []byte
		if err := rows.Scan(&e.ID, &e.Name, &e.Host, &e.Category, &aliases, &e.URL,
			&e.IsSensitive, &e.LocalOnly, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("media: scan emoji: %w", err)
		}
		e.Aliases = []string{}
		if len(aliases) > 0 {
			if err := json.Unmarshal(aliases, &e.Aliases); err != nil {
				return nil, fmt.Errorf("media: decode emoji aliases: %w", err)
			}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}

// LookupLocal maps the given shortcodes to the URLs of local emoji rows.
// Unknown names are simply absent from the result.
func (s *EmojiService) LookupLocal(ctx context.Context, names []string) (map[string]string, error) {
	out := make(map[string]string, len(names))
	if len(names) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT name, url FROM emojis WHERE host IS NULL AND name = ANY($1)
`, names)
	if err != nil {
		return nil, fmt.Errorf("media: lookup emoji: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var name, url string
		if err := rows.Scan(&name, &url); err != nil {
			return nil, fmt.Errorf("media: scan emoji: %w", err)
		}
		out[name] = url
	}
	return out, rows.Err()
}
