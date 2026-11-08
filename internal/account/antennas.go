package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Antenna is a standing filter that collects notes as they are created.
// The surface mirrors Misskey's Antenna schema
// (packages/misskey-js/src/autogen/types.ts L5080); Firefish-only fields
// (instances, userGroupId, expression) are not part of the contract.
type Antenna struct {
	ID                      string
	UserID                  string
	Name                    string
	Src                     string
	UserListID              *string
	Users                   []string
	Keywords                [][]string
	ExcludeKeywords         [][]string
	CaseSensitive           bool
	ExcludeBots             bool
	WithReplies             bool
	WithFile                bool
	LocalOnly               bool
	ExcludeSensitiveChannel bool
	IsActive                bool
	LastUsedAt              time.Time
	CreatedAt               time.Time
}

// The upstream src enum. 'home' is behaviourally identical to 'all'
// (upstream leaves it as a TODO), and this port reproduces that rather
// than inventing a followings-only filter.
const (
	AntennaSrcHome           = "home"
	AntennaSrcAll            = "all"
	AntennaSrcUsers          = "users"
	AntennaSrcList           = "list"
	AntennaSrcUsersBlacklist = "users_blacklist"
)

// antennaSrcs is the closed set create/update accept.
var antennaSrcs = map[string]bool{
	AntennaSrcHome: true, AntennaSrcAll: true, AntennaSrcUsers: true,
	AntennaSrcList: true, AntennaSrcUsersBlacklist: true,
}

// antennaLimit is Misskey's role-policy default (RoleService.ts L99),
// enforced only on create.
const antennaLimit = 5

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrAntennaNotFound    = errors.New("account: antenna not found")
	ErrTooManyAntennas    = errors.New("account: too many antennas")
	ErrEmptyKeyword       = errors.New("account: antenna keywords empty")
	ErrAntennaSrcInvalid  = errors.New("account: antenna src invalid")
	ErrAntennaNameInvalid = errors.New("account: antenna name invalid")
)

// AntennaInput carries the write-path fields of antennas/create.
type AntennaInput struct {
	Name                    string
	Src                     string
	UserListID              *string
	Users                   []string
	Keywords                [][]string
	ExcludeKeywords         [][]string
	CaseSensitive           bool
	ExcludeBots             bool
	WithReplies             bool
	WithFile                bool
	LocalOnly               bool
	ExcludeSensitiveChannel bool
}

// AntennaPatch is the merge shape of antennas/update: pointer presence
// decides whether a field is written, and the two keywords booleans
// separate "omitted" from "explicitly empty", which upstream's EMPTY_KEYWORD
// check distinguishes.
type AntennaPatch struct {
	Name                    *string
	Src                     *string
	UserListID              *string
	Users                   []string
	Keywords                [][]string
	KeywordsPresent         bool
	ExcludeKeywords         [][]string
	ExcludeKeywordsPresent  bool
	CaseSensitive           *bool
	ExcludeBots             *bool
	WithReplies             *bool
	WithFile                *bool
	LocalOnly               *bool
	ExcludeSensitiveChannel *bool
}

// AntennaNote carries what the ingest path knows about a note, so the
// matcher needs no reference to the note package's service. AuthorAcct is
// the author's full acct — "@user" for a local account, "@user@host" for
// a remote one — resolved by the ingest join, in the shape the users
// array stores.
type AntennaNote struct {
	ID                 string
	AuthorID           string
	AuthorAcct         string
	AuthorIsBot        bool
	Visibility         string
	VisibleUsers       []string
	ReplyID            *string
	Text               *string
	CW                 *string
	FileIDs            []string
	InSensitiveChannel bool
}

const antennaColumns = `id, user_id, name, src, user_list_id, users,
       keywords, exclude_keywords, case_sensitive, exclude_bots,
       with_replies, with_file, local_only, exclude_notes_in_sensitive_channel,
       is_active, last_used_at, created_at`

// scanAntenna reads one antenna row; raw array/jsonb columns arrive as
// []byte through pgx.
func scanAntenna(row pgx.Row) (*Antenna, error) {
	var a Antenna
	var users, keywords, excludeKeywords []byte
	err := row.Scan(&a.ID, &a.UserID, &a.Name, &a.Src, &a.UserListID, &users,
		&keywords, &excludeKeywords, &a.CaseSensitive, &a.ExcludeBots,
		&a.WithReplies, &a.WithFile, &a.LocalOnly, &a.ExcludeSensitiveChannel,
		&a.IsActive, &a.LastUsedAt, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAntennaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan antenna: %w", err)
	}
	a.Users = decodeTextArray(users)
	a.Keywords = decodeKeywordGroups(keywords)
	a.ExcludeKeywords = decodeKeywordGroups(excludeKeywords)
	return &a, nil
}

func scanAntennas(rows pgx.Rows) ([]*Antenna, error) {
	defer rows.Close()
	out := []*Antenna{}
	for rows.Next() {
		var a Antenna
		var users, keywords, excludeKeywords []byte
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Src, &a.UserListID, &users,
			&keywords, &excludeKeywords, &a.CaseSensitive, &a.ExcludeBots,
			&a.WithReplies, &a.WithFile, &a.LocalOnly, &a.ExcludeSensitiveChannel,
			&a.IsActive, &a.LastUsedAt, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan antenna: %w", err)
		}
		a.Users = decodeTextArray(users)
		a.Keywords = decodeKeywordGroups(keywords)
		a.ExcludeKeywords = decodeKeywordGroups(excludeKeywords)
		out = append(out, &a)
	}
	return out, rows.Err()
}

// decodeTextArray reads a Postgres text[] value scanned into []byte.
func decodeTextArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

// decodeKeywordGroups reads a jsonb string[][] value scanned into []byte.
func decodeKeywordGroups(raw []byte) [][]string {
	if len(raw) == 0 {
		return [][]string{}
	}
	var out [][]string
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return [][]string{}
	}
	return out
}

// CreateAntenna makes an antenna. Upstream's EMPTY_KEYWORD fires when both
// keyword groups are empty after cleaning — and because [].every() is
// vacuously true, omitted arrays trip it too.
func (s *Service) CreateAntenna(ctx context.Context, userID string, in AntennaInput) (*Antenna, error) {
	if in.Name == "" || len([]rune(in.Name)) > 100 {
		return nil, ErrAntennaNameInvalid
	}
	if !antennaSrcs[in.Src] {
		return nil, ErrAntennaSrcInvalid
	}
	if emptyKeywordGroups(in.Keywords) && emptyKeywordGroups(in.ExcludeKeywords) {
		return nil, ErrEmptyKeyword
	}
	if in.Src == AntennaSrcList && in.UserListID != nil {
		if _, err := s.OwnList(ctx, *in.UserListID, userID); err != nil {
			return nil, ErrUserListNotFound
		}
	}
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM antennas WHERE user_id = $1`, userID).Scan(&n); err != nil {
		return nil, fmt.Errorf("account: count antennas: %w", err)
	}
	if n >= antennaLimit {
		return nil, ErrTooManyAntennas
	}

	id := idgen.New()
	_, err := s.pool.Exec(ctx, `
INSERT INTO antennas (id, user_id, name, src, user_list_id, users,
                      keywords, exclude_keywords, case_sensitive, exclude_bots,
                      with_replies, with_file, local_only, exclude_notes_in_sensitive_channel)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`, id, userID, in.Name, in.Src, in.UserListID, textArrayValue(in.Users),
		jsonbValue(in.Keywords), jsonbValue(in.ExcludeKeywords),
		in.CaseSensitive, in.ExcludeBots, in.WithReplies, in.WithFile,
		in.LocalOnly, in.ExcludeSensitiveChannel)
	if err != nil {
		return nil, fmt.Errorf("account: create antenna: %w", err)
	}
	return s.GetAntenna(ctx, id, userID)
}

// textArrayValue normalises a nil slice for the jsonb users column,
// matching the also_known_as storage shape.
func textArrayValue(users []string) []string {
	if users == nil {
		return []string{}
	}
	return users
}

// jsonbValue normalises a nil group set for a jsonb parameter; pgx encodes
// [][]string as JSON natively.
func jsonbValue(groups [][]string) [][]string {
	if groups == nil {
		return [][]string{}
	}
	return groups
}

// GetAntenna loads an antenna the viewer owns; anyone else's antenna is
// not-found, as upstream's userId-scoped lookups answer.
func (s *Service) GetAntenna(ctx context.Context, antennaID, ownerID string) (*Antenna, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+antennaColumns+` FROM antennas WHERE id = $1`, antennaID)
	a, err := scanAntenna(row)
	if err != nil {
		return nil, err
	}
	if a.UserID != ownerID {
		return nil, ErrAntennaNotFound
	}
	return a, nil
}

// ListAntennas returns every antenna of one account, newest first.
// Upstream's antennas/list takes no limit and Firefish's 250 is ignored
// client-side, so the response is complete; the cap only guards memory.
func (s *Service) ListAntennas(ctx context.Context, userID string, limit int) ([]*Antenna, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+antennaColumns+` FROM antennas WHERE user_id = $1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: list antennas: %w", err)
	}
	return scanAntennas(rows)
}

// UpdateAntenna merges optional fields. A list-ownership lookup runs only
// when the resulting src is 'list' and a userListId is supplied, matching
// upstream. Update never re-checks antennaLimit and always marks the row
// active/just-used.
func (s *Service) UpdateAntenna(ctx context.Context, antennaID, ownerID string, patch *AntennaPatch) (*Antenna, error) {
	a, err := s.GetAntenna(ctx, antennaID, ownerID)
	if err != nil {
		return nil, err
	}
	if patch == nil {
		return a, nil
	}
	if patch.KeywordsPresent && patch.ExcludeKeywordsPresent &&
		emptyKeywordGroups(patch.Keywords) && emptyKeywordGroups(patch.ExcludeKeywords) {
		return nil, ErrEmptyKeyword
	}
	sets := []string{}
	args := []any{}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if patch.Name != nil {
		if *patch.Name == "" || len([]rune(*patch.Name)) > 100 {
			return nil, ErrAntennaNameInvalid
		}
		add("name", *patch.Name)
	}
	if patch.Src != nil {
		if !antennaSrcs[*patch.Src] {
			return nil, ErrAntennaSrcInvalid
		}
		add("src", *patch.Src)
		a.Src = *patch.Src
	}
	if patch.UserListID != nil {
		if a.Src == AntennaSrcList && *patch.UserListID != "" {
			if _, err := s.OwnList(ctx, *patch.UserListID, ownerID); err != nil {
				return nil, ErrUserListNotFound
			}
		}
		var listID *string
		if *patch.UserListID != "" {
			listID = patch.UserListID
		}
		add("user_list_id", listID)
	}
	if patch.Users != nil {
		add("users", textArrayValue(patch.Users))
	}
	if patch.KeywordsPresent {
		add("keywords", jsonbValue(patch.Keywords))
	}
	if patch.ExcludeKeywordsPresent {
		add("exclude_keywords", jsonbValue(patch.ExcludeKeywords))
	}
	if patch.CaseSensitive != nil {
		add("case_sensitive", *patch.CaseSensitive)
	}
	if patch.ExcludeBots != nil {
		add("exclude_bots", *patch.ExcludeBots)
	}
	if patch.WithReplies != nil {
		add("with_replies", *patch.WithReplies)
	}
	if patch.WithFile != nil {
		add("with_file", *patch.WithFile)
	}
	if patch.LocalOnly != nil {
		add("local_only", *patch.LocalOnly)
	}
	if patch.ExcludeSensitiveChannel != nil {
		add("exclude_notes_in_sensitive_channel", *patch.ExcludeSensitiveChannel)
	}
	if len(sets) > 0 {
		args = append(args, a.ID)
		if _, err := s.pool.Exec(ctx,
			`UPDATE antennas SET `+strings.Join(sets, ", ")+` WHERE id = $`+fmt.Sprint(len(args)),
			args...); err != nil {
			return nil, fmt.Errorf("account: update antenna: %w", err)
		}
	}
	_, _ = s.pool.Exec(ctx,
		`UPDATE antennas SET is_active = true, last_used_at = now() WHERE id = $1`, a.ID)
	return s.GetAntenna(ctx, a.ID, ownerID)
}

// DeleteAntenna removes an antenna; the join table cascades.
func (s *Service) DeleteAntenna(ctx context.Context, antennaID, ownerID string) error {
	ct, err := s.pool.Exec(ctx,
		`DELETE FROM antennas WHERE id = $1 AND user_id = $2`, antennaID, ownerID)
	if err != nil {
		return fmt.Errorf("account: delete antenna: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrAntennaNotFound
	}
	return nil
}

// cleanKeywordGroups drops empty words and fully-empty groups, what the
// upstream matcher does before testing (AntennaService.ts).
func cleanKeywordGroups(groups [][]string) [][]string {
	cleaned := [][]string{}
	for _, group := range groups {
		words := []string{}
		for _, word := range group {
			if word != "" {
				words = append(words, word)
			}
		}
		if len(words) > 0 {
			cleaned = append(cleaned, words)
		}
	}
	return cleaned
}

// emptyKeywordGroups reports whether every word of every group is empty —
// the upstream flat().every(x => x === ”) test.
func emptyKeywordGroups(groups [][]string) bool {
	for _, group := range groups {
		for _, word := range group {
			if word != "" {
				return false
			}
		}
	}
	return true
}

// groupsMatchText reports whether any group's every keyword appears in the
// haystack (OR of ANDs).
func groupsMatchText(groups [][]string, haystack string) bool {
	for _, group := range groups {
		all := true
		for _, word := range group {
			if !strings.Contains(haystack, word) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// NoteMatchesAntenna is the port of AntennaService.checkHitAntenna at
// c05e844: the checks run in upstream's order and the first failure wins.
// followsOwner reports whether the antenna owner follows or is the author
// (the followers-visibility rule), and ownerInVisibleUsers whether the
// owner stands in a specified note's audience.
func (s *Service) NoteMatchesAntenna(a *Antenna, n *AntennaNote, followsOwner, ownerInVisibleUsers bool) bool {
	// The sensitive-channel exclusion upstream checks first
	// (AntennaService.ts L118).
	if a.ExcludeSensitiveChannel && n.InSensitiveChannel {
		return false
	}
	if a.ExcludeBots && n.AuthorIsBot {
		return false
	}
	// localOnly matches on the author's host: a local-only note renoted
	// by a remote account still carries the remote author here. A local
	// acct has one '@', a remote one carries "@user@host".
	if a.LocalOnly && strings.Count(n.AuthorAcct, "@") > 1 {
		return false
	}
	if !a.WithReplies && n.ReplyID != nil {
		return false
	}
	switch n.Visibility {
	case "specified":
		if n.AuthorID != a.UserID && !ownerInVisibleUsers {
			return false
		}
	case "followers":
		if n.AuthorID != a.UserID && !followsOwner {
			return false
		}
	}

	// src filter.
	switch a.Src {
	case AntennaSrcList:
		if a.UserListID == nil {
			return false
		}
		member, err := s.IsMember(context.Background(), *a.UserListID, n.AuthorID)
		if err != nil || !member {
			return false
		}
	case AntennaSrcUsers, AntennaSrcUsersBlacklist:
		lower := strings.ToLower(n.AuthorAcct)
		matched := false
		for _, entry := range a.Users {
			if strings.ToLower(entry) == lower {
				matched = true
				break
			}
		}
		if (a.Src == AntennaSrcUsers && !matched) || (a.Src == AntennaSrcUsersBlacklist && matched) {
			return false
		}
	}

	// Keywords: every non-empty group's words must appear in text+cw.
	text := ""
	if n.Text != nil {
		text += *n.Text
	}
	if n.CW != nil {
		text += "\n" + *n.CW
	}
	groups := cleanKeywordGroups(a.Keywords)
	exclude := cleanKeywordGroups(a.ExcludeKeywords)
	if len(groups) > 0 || len(exclude) > 0 {
		if n.Text == nil && n.CW == nil {
			return false
		}
		if !a.CaseSensitive {
			text = strings.ToLower(text)
		}
	}

	if len(groups) > 0 {
		needle := normalizeGroups(groups, a.CaseSensitive)
		if !groupsMatchText(needle, text) {
			return false
		}
	}
	if len(exclude) > 0 {
		needle := normalizeGroups(exclude, a.CaseSensitive)
		if groupsMatchText(needle, text) {
			return false
		}
	}

	// withFile: a note without files is dropped; FileIDs is always
	// non-nil after packing for Cinderfish notes.
	if a.WithFile && len(n.FileIDs) == 0 {
		return false
	}
	return true
}

// normalizeGroups lowercases every keyword when the antenna is
// case-insensitive, mirroring the upstream lowercase-once comparison.
func normalizeGroups(groups [][]string, caseSensitive bool) [][]string {
	if caseSensitive {
		return groups
	}
	lowered := make([][]string, len(groups))
	for i, group := range groups {
		words := make([]string, len(group))
		for j, w := range group {
			words[j] = strings.ToLower(w)
		}
		lowered[i] = words
	}
	return lowered
}

// LoadOwnedAntennaIDsForUser is a helper for tests that need the raw ids.
func (s *Service) LoadOwnedAntennaIDsForUser(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM antennas WHERE user_id = $1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: antenna ids: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// IngestNote runs a freshly created note past every antenna and records
// the hits. It returns the matched antenna ids so the caller can publish
// the streaming event; with no antennas configured it is a cheap no-op.
//
// Upstream keeps the whole instance's antenna list cached in memory; this
// port re-reads it per note, which one query buys at personal-instance
// scale. The followers/specified visibility rules are evaluated against
// the note as the ingest caller saw it, so deleted notes never match.
func (s *Service) IngestNote(ctx context.Context, n AntennaNote) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+antennaColumns+` FROM antennas`)
	if err != nil {
		return nil, fmt.Errorf("account: ingest antennas: %w", err)
	}
	antennas, err := scanAntennas(rows)
	if err != nil {
		return nil, err
	}
	if len(antennas) == 0 {
		return nil, nil
	}

	owners := make([]string, 0, len(antennas))
	for _, a := range antennas {
		owners = append(owners, a.UserID)
	}

	// The followers rule: the antenna owner follows (or is) the author.
	following := map[string]bool{}
	if n.Visibility == "followers" {
		rows, err := s.pool.Query(ctx, `
SELECT follower_id FROM followings
WHERE followee_id = $1 AND follower_id = ANY($2)
`, n.AuthorID, owners)
		if err != nil {
			return nil, fmt.Errorf("account: ingest follows: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			following[id] = true
		}
		rows.Close()
	}

	// The specified rule: the owner stands in the note's audience.
	visible := map[string]bool{}
	if n.Visibility == "specified" {
		rows, err := s.pool.Query(ctx, `
SELECT visible_user_id FROM note_visibilities
WHERE note_id = $1 AND visible_user_id = ANY($2)
`, n.ID, owners)
		if err != nil {
			return nil, fmt.Errorf("account: ingest visibility: %w", err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, err
			}
			visible[id] = true
		}
		rows.Close()
	}

	matched := []string{}
	for _, a := range antennas {
		if !s.NoteMatchesAntenna(a, &n, following[a.UserID], visible[a.UserID]) {
			continue
		}
		matched = append(matched, a.ID)
	}
	if len(matched) == 0 {
		return matched, nil
	}

	for _, antennaID := range matched {
		if _, err := s.pool.Exec(ctx, `
INSERT INTO antenna_notes (antenna_id, note_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING
`, antennaID, n.ID); err != nil {
			return matched, fmt.Errorf("account: record antenna note: %w", err)
		}
	}
	return matched, nil
}

// RemoveAntennaNote drops one note from an antenna's collection — the
// antennas/remove-note write. Removing a note that is not in the antenna
// is not an error, matching the unconditional DELETE of upstream's list
// removal.
func (s *Service) RemoveAntennaNote(ctx context.Context, antennaID, noteID string) error {
	_, err := s.pool.Exec(ctx, `
DELETE FROM antenna_notes WHERE antenna_id = $1 AND note_id = $2
`, antennaID, noteID)
	if err != nil {
		return fmt.Errorf("account: remove antenna note: %w", err)
	}
	return nil
}
