package account

import (
	"context"
	"fmt"
	"strings"
)

// AdminUserParams is the filter set of `admin/show-users`.
type AdminUserParams struct {
	Limit    int
	Offset   int
	Sort     string
	State    string
	Origin   string
	Username *string
	Hostname *string
}

// AdminList answers admin/show-users with Misskey's filters. Two degradations
// are deliberate: the users table has no `updated_at`, so ±updatedAt sort by
// created_at, and there is no silenced flag, so state=silenced matches nothing
// instead of silently returning every account.
func (s *Service) AdminList(ctx context.Context, p AdminUserParams) ([]*User, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	conds := []string{}
	args := []any{}

	switch p.State {
	case "", "all":
	case "available", "alive":
		conds = append(conds, "NOT is_suspended")
	case "admin", "moderator", "adminOrModerator":
		conds = append(conds, "is_admin")
	case "suspended":
		conds = append(conds, "is_suspended")
	case "silenced":
		conds = append(conds, "FALSE")
	default:
		return nil, fmt.Errorf("account: unknown state %q", p.State)
	}

	switch p.Origin {
	case "", "combined":
	case "local":
		conds = append(conds, "host IS NULL")
	case "remote":
		conds = append(conds, "host IS NOT NULL")
	default:
		return nil, fmt.Errorf("account: unknown origin %q", p.Origin)
	}

	if p.Username != nil && *p.Username != "" {
		args = append(args, escapeLike(strings.ToLower(*p.Username))+"%")
		conds = append(conds, fmt.Sprintf("username_lower LIKE $%d", len(args)))
	}
	if p.Hostname != nil && *p.Hostname != "" {
		args = append(args, strings.ToLower(*p.Hostname))
		conds = append(conds, fmt.Sprintf("host = $%d", len(args)))
	}

	// Misskey's default is `id ASC`, which in an all-ULID database is creation
	// order. Ordering by created_at keeps that meaning when a row was inserted
	// with a hand-written id (the bootstrap administrator).
	order := "created_at ASC"
	switch p.Sort {
	case "+follower":
		order = "followers_count DESC"
	case "-follower":
		order = "followers_count ASC"
	case "+createdAt":
		order = "created_at DESC"
	case "-createdAt":
		order = "created_at ASC"
	case "+updatedAt":
		order = "created_at DESC"
	case "-updatedAt":
		order = "created_at ASC"
	}

	args = append(args, p.Limit, p.Offset)
	query := `
SELECT ` + userColumns + `
FROM users`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: admin list users: %w", err)
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

// UsersByIDs loads users for batch packing (the admin file list needs the
// uploader of every row). Missing ids are simply absent from the map.
func (s *Service) UsersByIDs(ctx context.Context, ids []string) (map[string]*User, error) {
	out := make(map[string]*User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT `+userColumns+`
FROM users WHERE id = ANY($1)
`, ids)
	if err != nil {
		return nil, fmt.Errorf("account: users by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out[u.ID] = u
	}
	return out, rows.Err()
}
