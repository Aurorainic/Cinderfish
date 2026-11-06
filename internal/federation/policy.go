package federation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Mode is the federation mode. The values are the project's own
// vocabulary (public/allowlist/none); meta exposes the Misskey mapping
// all/specified/none.
type Mode string

// Federation modes.
const (
	// ModePublic federates with every host.
	ModePublic Mode = "public"
	// ModeAllowlist federates only with hosts on the allowlist.
	ModeAllowlist Mode = "allowlist"
	// ModeNone federates with nobody.
	ModeNone Mode = "none"
)

// ParseMode validates a configured mode string.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModePublic, ModeAllowlist, ModeNone:
		return Mode(s), nil
	}
	return "", fmt.Errorf("federation: unknown mode %q", s)
}

// MisskeyMode maps a mode onto Misskey's own three-state vocabulary.
func (m Mode) MisskeyMode() string {
	switch m {
	case ModeNone:
		return "none"
	case ModeAllowlist:
		return "specified"
	default:
		return "all"
	}
}

// ErrFederationDenied is returned by a policy check that refused a host.
var ErrFederationDenied = errors.New("federation: federation with this host is not allowed")

// Policy decides whether Cinderfish may exchange anything with a host.
//
// The mode may change at runtime: the architecture calls for switching
// between public, allowlist and none without a restart, so the source is
// read on every check rather than captured at construction time.
type Policy struct {
	db       *pgxpool.Pool
	selfHost string

	// mu guards the cached allowlist and blocked-host list.
	mu        sync.RWMutex
	cached    []string
	cachedAt  time.Time
	blocked   []string
	blockedAt time.Time
}

// allowlistTTL bounds how stale the cached allowlist may be. The list
// changes rarely, and a few seconds of staleness after an administrator
// removes a host is acceptable; the inbox re-checks on every request.
const allowlistTTL = 10 * time.Second

// NewPolicy returns a policy for this instance.
func NewPolicy(db *pgxpool.Pool, selfHost string) *Policy {
	return &Policy{db: db, selfHost: normalizeHost(selfHost)}
}

// SelfHost returns the instance hostname the policy was built with.
func (p *Policy) SelfHost() string { return p.selfHost }

// Mode reads the current federation mode from instance_meta. The
// configuration value seeds that row at startup; changing it afterwards
// changes inbound behaviour on the next request.
func (p *Policy) Mode(ctx context.Context) (Mode, error) {
	if p.db == nil {
		return ModeNone, nil
	}
	var raw string
	err := p.db.QueryRow(ctx, "SELECT value::text FROM instance_meta WHERE key = 'federationMode'").Scan(&raw)
	if err != nil {
		return "", fmt.Errorf("federation: read mode: %w", err)
	}
	return ParseMode(strings.Trim(raw, "\""))
}

// IsAllowed reports whether federation with host is permitted.
//
// selfHost is always allowed so the instance can resolve its own actors
// (for example when the frontend fetches /api/ap/show for a local user).
func (p *Policy) IsAllowed(ctx context.Context, host string) bool {
	host = normalizeHost(host)
	if host == "" {
		return false
	}
	if host == p.selfHost {
		return true
	}
	if p.HostBlocked(ctx, host) {
		return false
	}
	mode, err := p.Mode(ctx)
	if err != nil || mode == ModeNone {
		return false
	}
	if mode == ModePublic {
		return true
	}
	return p.hostInAllowlist(ctx, host)
}

// CheckAllowed returns ErrFederationDenied when host may not federate.
func (p *Policy) CheckAllowed(ctx context.Context, host string) error {
	if p.IsAllowed(ctx, host) {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrFederationDenied, host)
}

// CheckURLAllowed resolves rawURL's host and applies the policy to it.
func (p *Policy) CheckURLAllowed(ctx context.Context, rawURL string) error {
	host, err := HostOf(rawURL)
	if err != nil {
		return err
	}
	return p.CheckAllowed(ctx, host)
}

// hostInAllowlist applies the suffix rule Misskey uses for its
// allowlist: an entry of "example.com" matches example.com and
// sub.example.com but not notexample.com.
func (p *Policy) hostInAllowlist(ctx context.Context, host string) bool {
	for _, entry := range p.Allowlist(ctx) {
		if HostMatchesSuffix(host, entry) {
			return true
		}
	}
	return false
}

// HostBlocked reports whether the instance has blocked federation with host.
// Matching is exact after normalisation: an operator who wants a wildcard adds
// each host explicitly, which matches how the list is presented in the admin
// page.
func (p *Policy) HostBlocked(ctx context.Context, host string) bool {
	host = normalizeHost(host)
	if host == "" {
		return false
	}
	for _, entry := range p.BlockedHosts(ctx) {
		if entry == host {
			return true
		}
	}
	return false
}

// BlockedHosts returns the current list, reading through the cache.
func (p *Policy) BlockedHosts(ctx context.Context) []string {
	p.mu.RLock()
	if time.Since(p.blockedAt) < allowlistTTL {
		out := p.blocked
		p.mu.RUnlock()
		return out
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.blockedAt) < allowlistTTL {
		return p.blocked
	}
	entries := p.readBlockedHosts(ctx)
	p.blocked = entries
	p.blockedAt = time.Now()
	return entries
}

func (p *Policy) readBlockedHosts(ctx context.Context) []string {
	if p.db == nil {
		return nil
	}
	var raw []byte
	err := p.db.QueryRow(ctx, "SELECT value FROM instance_meta WHERE key = 'blockedHosts'").Scan(&raw)
	if err != nil {
		return nil
	}
	var hosts []string
	if err := json.Unmarshal(raw, &hosts); err != nil {
		return nil
	}
	out := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = normalizeHost(host)
		if host != "" {
			out = append(out, host)
		}
	}
	return out
}

// SetBlockedHosts replaces the list and invalidates the cache immediately, so
// the next inbox or delivery does not wait for the TTL.
func (p *Policy) SetBlockedHosts(ctx context.Context, hosts []string) error {
	if p.db == nil {
		return fmt.Errorf("federation: no database")
	}
	seen := make(map[string]bool, len(hosts))
	clean := make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = normalizeHost(host)
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		clean = append(clean, host)
	}
	sort.Strings(clean)

	payload, err := json.Marshal(clean)
	if err != nil {
		return fmt.Errorf("federation: encode blocked hosts: %w", err)
	}
	if _, err := p.db.Exec(ctx, `
INSERT INTO instance_meta (key, value) VALUES ('blockedHosts', $1::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
`, payload); err != nil {
		return fmt.Errorf("federation: save blocked hosts: %w", err)
	}
	p.mu.Lock()
	p.blocked = clean
	p.blockedAt = time.Now()
	p.mu.Unlock()
	return nil
}

// Allowlist returns the current entries, reading through the cache.
func (p *Policy) Allowlist(ctx context.Context) []string {
	p.mu.RLock()
	if time.Since(p.cachedAt) < allowlistTTL {
		out := p.cached
		p.mu.RUnlock()
		return out
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	// Another goroutine may have refreshed it while we waited.
	if time.Since(p.cachedAt) < allowlistTTL {
		return p.cached
	}
	entries := p.readAllowlist(ctx)
	p.cached = entries
	p.cachedAt = time.Now()
	return entries
}

func (p *Policy) readAllowlist(ctx context.Context) []string {
	if p.db == nil {
		return nil
	}
	rows, err := p.db.Query(ctx, "SELECT host FROM federation_allowlist ORDER BY host")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return out
		}
		out = append(out, normalizeHost(h))
	}
	return out
}

// InvalidateAllowlist drops the cached allowlist so the next check
// re-reads it.
func (p *Policy) InvalidateAllowlist() {
	p.mu.Lock()
	p.cached = nil
	p.cachedAt = time.Time{}
	p.mu.Unlock()
}

// AddAllowlistHost adds a host to the allowlist.
func (p *Policy) AddAllowlistHost(ctx context.Context, host string) error {
	host = normalizeHost(host)
	if host == "" {
		return errors.New("federation: empty host")
	}
	if p.db == nil {
		return errors.New("federation: no database")
	}
	_, err := p.db.Exec(ctx, "INSERT INTO federation_allowlist (host) VALUES ($1) ON CONFLICT (host) DO NOTHING", host)
	if err != nil {
		return fmt.Errorf("federation: add allowlist host: %w", err)
	}
	p.InvalidateAllowlist()
	return nil
}

// RemoveAllowlistHost removes a host from the allowlist.
func (p *Policy) RemoveAllowlistHost(ctx context.Context, host string) error {
	if p.db == nil {
		return errors.New("federation: no database")
	}
	if _, err := p.db.Exec(ctx, "DELETE FROM federation_allowlist WHERE host = $1", normalizeHost(host)); err != nil {
		return fmt.Errorf("federation: remove allowlist host: %w", err)
	}
	p.InvalidateAllowlist()
	return nil
}

// HostMatchesSuffix reports whether host is entry or a subdomain of it.
// This is Misskey's rule: .host endsWith .entry, which matches
// sub.example.com for example.com but not notexample.com.
func HostMatchesSuffix(host, entry string) bool {
	host, entry = normalizeHost(host), normalizeHost(entry)
	if host == "" || entry == "" {
		return false
	}
	if host == entry {
		return true
	}
	return strings.HasSuffix(host, "."+entry)
}

// normalizeHost lowercases a host and strips a port if one is present.
func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(h); err == nil {
		return host
	}
	return h
}

// HostOf extracts the lowercase hostname from an absolute URL.
func HostOf(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("federation: parse url %q: %w", rawURL, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("federation: url %q has no host", rawURL)
	}
	return normalizeHost(u.Host), nil
}
