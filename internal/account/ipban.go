package account

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// IP bans are a Cinderfish extension. Misskey and Firefish configure their
// admin console around registration policy and captchas instead, so these
// endpoints have no upstream contract; the shapes are kept simple and
// documented in docs/HANDOVER.md (§12GV).

// IPBan is one ban rule: a single address or a CIDR network.
type IPBan struct {
	ID          string
	IP          string
	Reason      string
	CreatedByID string
	CreatedBy   string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

// ErrIPBanNotFound is returned when a ban id does not exist.
var ErrIPBanNotFound = errors.New("account: ip ban not found")

// NormalizeIPRule canonicalises a ban rule: a bare address becomes the same
// address in canonical form, a network becomes its masked CIDR. Anything
// else is refused.
func NormalizeIPRule(rule string) (string, error) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return "", errors.New("an IP address or CIDR network is required")
	}
	if strings.Contains(rule, "/") {
		_, network, err := net.ParseCIDR(rule)
		if err != nil {
			return "", fmt.Errorf("invalid CIDR network: %v", err)
		}
		return network.String(), nil
	}
	addr := net.ParseIP(rule)
	if addr == nil {
		return "", errors.New("invalid IP address")
	}
	return addr.String(), nil
}

// ipRuleMatches reports whether an address falls under a stored rule.
func ipRuleMatches(rule, ip string) bool {
	addr := net.ParseIP(strings.TrimSpace(ip))
	if addr == nil {
		return false
	}
	if strings.Contains(rule, "/") {
		_, network, err := net.ParseCIDR(rule)
		if err != nil {
			return false
		}
		return network.Contains(addr)
	}
	return rule == addr.String()
}

// CreateIPBan stores a ban rule. The rule is normalised first, so callers
// may send "192.0.2.5/24" and the row keeps the masked network.
func (s *Service) CreateIPBan(ctx context.Context, rule, reason string, expiresAt *time.Time, createdByID string) (*IPBan, error) {
	normalized, err := NormalizeIPRule(rule)
	if err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	ban := &IPBan{
		ID:          idgen.New(),
		IP:          normalized,
		Reason:      reason,
		CreatedByID: createdByID,
		CreatedAt:   time.Now(),
		ExpiresAt:   expiresAt,
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO ip_bans (id, ip, reason, created_by_id, created_at, expires_at)
VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
`, ban.ID, ban.IP, ban.Reason, createdByID, ban.CreatedAt, expiresAt); err != nil {
		return nil, fmt.Errorf("account: create ip ban: %w", err)
	}
	return ban, nil
}

// ListIPBans returns the ban rules, newest first.
func (s *Service) ListIPBans(ctx context.Context, limit, offset int) ([]IPBan, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
SELECT b.id, b.ip, b.reason, COALESCE(b.created_by_id, ''), COALESCE(u.username, ''),
       b.created_at, b.expires_at
FROM ip_bans b
LEFT JOIN users u ON u.id = b.created_by_id
ORDER BY b.id DESC
LIMIT $1 OFFSET $2
`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("account: list ip bans: %w", err)
	}
	defer rows.Close()

	out := make([]IPBan, 0, limit)
	for rows.Next() {
		var ban IPBan
		if err := rows.Scan(&ban.ID, &ban.IP, &ban.Reason, &ban.CreatedByID,
			&ban.CreatedBy, &ban.CreatedAt, &ban.ExpiresAt); err != nil {
			return nil, fmt.Errorf("account: scan ip ban: %w", err)
		}
		out = append(out, ban)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account: list ip bans: %w", err)
	}
	return out, nil
}

// DeleteIPBan removes one rule.
func (s *Service) DeleteIPBan(ctx context.Context, id string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM ip_bans WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("account: delete ip ban: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrIPBanNotFound
	}
	return nil
}

// IsIPBanned reports whether an address is covered by a live rule. The
// active rules are matched in Go because the table stores CIDR text; the
// list is small and bounded by what an operator enters.
func (s *Service) IsIPBanned(ctx context.Context, ip string) (bool, error) {
	if strings.TrimSpace(ip) == "" {
		return false, nil
	}
	rows, err := s.pool.Query(ctx, `
SELECT ip FROM ip_bans WHERE expires_at IS NULL OR expires_at > now()
`)
	if err != nil {
		return false, fmt.Errorf("account: read ip bans: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var rule string
		if err := rows.Scan(&rule); err != nil {
			return false, fmt.Errorf("account: scan ip ban: %w", err)
		}
		if ipRuleMatches(rule, ip) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("account: read ip bans: %w", err)
	}
	return false, nil
}

// RecordSigninFailure appends one failed signin attempt for the source
// address. Unknown usernames cannot be stored in signins (the row needs an
// account), so this table is the counter the automatic-ban rule reads.
func (s *Service) RecordSigninFailure(ctx context.Context, ip string) error {
	if strings.TrimSpace(ip) == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
INSERT INTO signin_failures (id, ip) VALUES ($1, $2)
`, idgen.New(), ip)
	if err != nil {
		return fmt.Errorf("account: record signin failure: %w", err)
	}
	return nil
}

// CountRecentSigninFailures counts failed signin attempts from an address
// since the given moment.
func (s *Service) CountRecentSigninFailures(ctx context.Context, ip string, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
SELECT count(*) FROM signin_failures WHERE ip = $1 AND created_at > $2
`, ip, since).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("account: count signin failures: %w", err)
	}
	return n, nil
}

// AdminSigninRecord is one signin row with the account it belongs to.
type AdminSigninRecord struct {
	SigninRecord
	UserID   string
	Username string
}

// AdminSigninHistory lists signins across all accounts for the admin audit
// view, newest first. userID restricts the listing to one account.
func (s *Service) AdminSigninHistory(ctx context.Context, userID string, limit, offset int) ([]AdminSigninRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	query := `
SELECT s.id, s.created_at, COALESCE(s.ip, ''), s.success, u.id, u.username
FROM signins s
JOIN users u ON u.id = s.user_id`
	args := []any{}
	if userID != "" {
		args = append(args, userID)
		query += fmt.Sprintf(" WHERE s.user_id = $%d", len(args))
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY s.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: admin signin history: %w", err)
	}
	defer rows.Close()

	out := make([]AdminSigninRecord, 0, limit)
	for rows.Next() {
		var rec AdminSigninRecord
		if err := rows.Scan(&rec.ID, &rec.CreatedAt, &rec.IP, &rec.Success,
			&rec.UserID, &rec.Username); err != nil {
			return nil, fmt.Errorf("account: scan signin: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account: admin signin history: %w", err)
	}
	return out, nil
}
