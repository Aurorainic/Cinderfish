package account

import (
	"context"
	"fmt"
	"strings"
)

// UserSearchParams is the filter set of `users/search`. Origin is Misskey's
// "local" | "remote" | "combined"; an empty value is treated as "combined".
type UserSearchParams struct {
	Query  string
	Origin string
	Host   *string
	Limit  int
	Offset int
	// SinceID / UntilID are not part of Misskey's users/search, but the
	// client paginator here pages by id cursor. Accepting them keeps one
	// pagination component for every list.
	SinceID string
	UntilID string
	// ViewerID, when set, hides users on either side of a block.
	ViewerID string
}

// escapeLike escapes the characters LIKE treats specially. The query below
// relies on PostgreSQL's default backslash escape.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// SearchUsers answers `users/search` by matching the username, display name or
// profile description. Suspended accounts never appear, matching Misskey's
// user search. Results are newest-id first; `offset` is supported because this
// endpoint has no cursor in the Misskey contract.
func (s *Service) SearchUsers(ctx context.Context, p UserSearchParams) ([]*User, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	pattern := "%" + escapeLike(p.Query) + "%"
	args := []any{pattern}
	conds := []string{
		"NOT u.is_suspended",
		`(u.username_lower ILIKE $1 OR COALESCE(p.name, '') ILIKE $1 OR COALESCE(p.description, '') ILIKE $1)`,
	}

	if p.ViewerID != "" {
		args = append(args, p.ViewerID)
		viewer := fmt.Sprintf("$%d", len(args))
		conds = append(conds, fmt.Sprintf(
			"NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id = %s AND b.blockee_id = u.id) OR (b.blocker_id = u.id AND b.blockee_id = %s))",
			viewer, viewer))
	}

	switch p.Origin {
	case "local":
		conds = append(conds, "u.host IS NULL")
	case "remote":
		conds = append(conds, "u.host IS NOT NULL")
	}
	if p.Host != nil {
		if *p.Host == "" {
			conds = append(conds, "u.host IS NULL")
		} else {
			args = append(args, *p.Host)
			conds = append(conds, fmt.Sprintf("u.host = $%d", len(args)))
		}
	}

	order := "DESC"
	if p.SinceID != "" {
		args = append(args, p.SinceID)
		conds = append(conds, fmt.Sprintf("u.id > $%d", len(args)))
	}
	if p.UntilID != "" {
		args = append(args, p.UntilID)
		conds = append(conds, fmt.Sprintf("u.id < $%d", len(args)))
	}
	if p.SinceID != "" && p.UntilID == "" {
		// A "load newer" cursor walks upward, matching the note lists.
		order = "ASC"
	}

	args = append(args, p.Limit, p.Offset)
	query := `
SELECT u.id, u.username, u.username_lower, u.host, COALESCE(u.password_hash, ''),
       COALESCE(u.token, ''), u.is_admin, u.is_suspended, u.created_at,
       COALESCE(u.moved_to_uri, ''), COALESCE(u.also_known_as, '[]'::jsonb)
FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE ` + strings.Join(conds, " AND ") + fmt.Sprintf(`
ORDER BY u.id %s LIMIT $%d OFFSET $%d`, order, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: search users: %w", err)
	}
	defer rows.Close()

	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
