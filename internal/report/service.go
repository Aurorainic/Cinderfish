// Package report stores abuse reports: a user's complaint about another
// account, and the administrator's resolution flag.
package report

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Report is one abuse report row.
type Report struct {
	ID           string
	ReporterID   string
	TargetUserID string
	Comment      string
	Resolved     bool
	// Forwarded is true when an administrator sent the complaint to the
	// reported account's instance as an ActivityPub Flag.
	Forwarded bool
	CreatedAt time.Time
}

// Service reads and writes reports.
type Service struct {
	pool *pgxpool.Pool
}

// NewService returns a Service backed by the given pool.
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

// Create stores a report. The caller validates the comment length.
func (s *Service) Create(ctx context.Context, reporterID, targetUserID, comment string) (*Report, error) {
	row := s.pool.QueryRow(ctx, `
INSERT INTO abuse_reports (id, reporter_id, target_user_id, comment)
VALUES ($1, $2, $3, $4)
RETURNING id, reporter_id, target_user_id, comment, resolved, forwarded, created_at
`, idgen.New(), reporterID, targetUserID, comment)
	var r Report
	if err := row.Scan(&r.ID, &r.ReporterID, &r.TargetUserID, &r.Comment, &r.Resolved, &r.Forwarded, &r.CreatedAt); err != nil {
		return nil, fmt.Errorf("report: create: %w", err)
	}
	return &r, nil
}

// ListParams is the filter set of admin/abuse-user-reports. State is
// "resolved", "unresolved" or "" (all).
type ListParams struct {
	Limit            int
	SinceID          string
	UntilID          string
	State            string
	ReporterOrigin   string
	TargetUserOrigin string
}

// List returns reports newest first.
func (s *Service) List(ctx context.Context, p ListParams) ([]*Report, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}

	conds := []string{}
	args := []any{}
	switch p.State {
	case "":
	case "resolved":
		conds = append(conds, "r.resolved")
	case "unresolved":
		conds = append(conds, "NOT r.resolved")
	default:
		return nil, fmt.Errorf("report: unknown state %q", p.State)
	}
	originCond := func(alias, origin string) (string, error) {
		switch origin {
		case "", "combined":
			return "", nil
		case "local":
			return alias + ".host IS NULL", nil
		case "remote":
			return alias + ".host IS NOT NULL", nil
		}
		return "", fmt.Errorf("report: unknown origin %q", origin)
	}
	if cond, err := originCond("reporter", p.ReporterOrigin); err != nil {
		return nil, err
	} else if cond != "" {
		conds = append(conds, cond)
	}
	if cond, err := originCond("target", p.TargetUserOrigin); err != nil {
		return nil, err
	} else if cond != "" {
		conds = append(conds, cond)
	}
	if p.SinceID != "" {
		args = append(args, p.SinceID)
		conds = append(conds, fmt.Sprintf("r.id > $%d", len(args)))
	}
	if p.UntilID != "" {
		args = append(args, p.UntilID)
		conds = append(conds, fmt.Sprintf("r.id < $%d", len(args)))
	}

	query := `
SELECT r.id, r.reporter_id, r.target_user_id, r.comment, r.resolved, r.forwarded, r.created_at
FROM abuse_reports r
JOIN users reporter ON reporter.id = r.reporter_id
JOIN users target ON target.id = r.target_user_id`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY r.id DESC LIMIT %d", p.Limit)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("report: list: %w", err)
	}
	defer rows.Close()

	var out []*Report
	for rows.Next() {
		var r Report
		if err := rows.Scan(&r.ID, &r.ReporterID, &r.TargetUserID, &r.Comment, &r.Resolved, &r.Forwarded, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("report: scan: %w", err)
		}
		out = append(out, &r)
	}
	return out, rows.Err()
}

// Resolve marks a report resolved. It reports whether the row existed.
func (s *Service) Resolve(ctx context.Context, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE abuse_reports SET resolved = TRUE WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("report: resolve: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// MarkForwarded flags a report as forwarded and reports whether this call
// performed the transition: a second call is a no-op, so a retried request
// cannot send the same Flag twice.
func (s *Service) MarkForwarded(ctx context.Context, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE abuse_reports SET forwarded = TRUE WHERE id = $1 AND NOT forwarded`, id)
	if err != nil {
		return false, fmt.Errorf("report: mark forwarded: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// UnmarkForwarded clears the flag after a failed delivery, so the
// administrator can retry the same report.
func (s *Service) UnmarkForwarded(ctx context.Context, id string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE abuse_reports SET forwarded = FALSE WHERE id = $1 AND forwarded`, id)
	if err != nil {
		return false, fmt.Errorf("report: unmark forwarded: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ErrNotFound is returned by Get when the id does not exist.
var ErrNotFound = errors.New("report: not found")

// Get loads one report.
func (s *Service) Get(ctx context.Context, id string) (*Report, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, reporter_id, target_user_id, comment, resolved, forwarded, created_at
FROM abuse_reports WHERE id = $1`, id)
	var r Report
	err := row.Scan(&r.ID, &r.ReporterID, &r.TargetUserID, &r.Comment, &r.Resolved, &r.Forwarded, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("report: get: %w", err)
	}
	return &r, nil
}
