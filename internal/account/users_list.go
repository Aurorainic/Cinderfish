package account

import (
	"context"
	"fmt"
	"strings"
)

// UsersParams is the filter set of the public `users` endpoint.
type UsersParams struct {
	Limit    int
	Offset   int
	Sort     string
	State    string
	Origin   string
	Hostname *string
}

// ListUsers answers `users`: a public directory of accounts. Suspended accounts
// are never listed. The state values mirror Misskey's; this port has no
// moderator flag, so moderator states map onto administrators.
func (s *Service) ListUsers(ctx context.Context, p UsersParams) ([]*User, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}

	conds := []string{"NOT is_suspended"}
	args := []any{}

	switch p.State {
	case "", "all", "alive":
	case "admin", "moderator", "adminOrModerator":
		conds = append(conds, "is_admin")
	default:
		return nil, fmt.Errorf("account: unknown state %q", p.State)
	}

	// Misskey defaults to local; the frontend asks for what it wants anyway.
	switch p.Origin {
	case "", "local":
		conds = append(conds, "host IS NULL")
	case "combined":
	case "remote":
		conds = append(conds, "host IS NOT NULL")
	default:
		return nil, fmt.Errorf("account: unknown origin %q", p.Origin)
	}

	if p.Hostname != nil && *p.Hostname != "" {
		args = append(args, strings.ToLower(*p.Hostname))
		conds = append(conds, fmt.Sprintf("host = $%d", len(args)))
	}

	order := "created_at DESC"
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
FROM users WHERE ` + strings.Join(conds, " AND ") +
		fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: list users: %w", err)
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
