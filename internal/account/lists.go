package account

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// UserList is one named group of users, owned by a local account. Public
// lists can be read and favourited by anyone; only the owner edits them.
type UserList struct {
	ID        string
	UserID    string
	Name      string
	IsPublic  bool
	CreatedAt time.Time
}

// UserListMembership links one user into one list. WithReplies controls
// whether the list timeline shows the member's replies.
type UserListMembership struct {
	ID          string
	UserListID  string
	UserID      string
	WithReplies bool
	CreatedAt   time.Time
}

// Caps that Misskey keeps in role policies (RoleService.ts L104-105).
// Cinderfish has no roles, so the defaults are fixed: ten lists per user,
// fifty members per list.
const (
	userListsPerUserLimit = 10
	userListMembersLimit  = 50
	userListNameMax       = 100
)

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrUserListNotFound     = errors.New("account: user list not found")
	ErrUserListNotPublic    = errors.New("account: user list is not public")
	ErrTooManyUserLists     = errors.New("account: too many user lists")
	ErrTooManyMembers       = errors.New("account: too many members in the user list")
	ErrAlreadyAdded         = errors.New("account: user already in the list")
	ErrNotAMember           = errors.New("account: user is not a member of the list")
	ErrAlreadyFavorited     = errors.New("account: user list already favorited")
	ErrNotFavorited         = errors.New("account: user list not favorited")
	ErrAccountMoved         = errors.New("account: account has moved")
	ErrRemoteUserNotAllowed = errors.New("account: remote user not allowed")
	ErrMemberBlocksYou      = errors.New("account: member blocks the caller")
)

// ErrListNameInvalid marks a list name outside 1..100 characters; the
// handler answers Misskey's INVALID_PARAM for it.
var ErrListNameInvalid = errors.New("account: list name invalid")

// CreateList makes a list owned by userID. Misskey rejects creation beyond
// the role-policy cap with TOO_MANY_USERLISTS.
func (s *Service) CreateList(ctx context.Context, userID, name string) (*UserList, error) {
	if name == "" || len([]rune(name)) > userListNameMax {
		return nil, ErrListNameInvalid
	}
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM user_lists WHERE user_id = $1`, userID).Scan(&n); err != nil {
		return nil, fmt.Errorf("account: count lists: %w", err)
	}
	if n >= userListsPerUserLimit {
		return nil, ErrTooManyUserLists
	}
	l := &UserList{ID: idgen.New(), UserID: userID, Name: name, CreatedAt: time.Now()}
	_, err := s.pool.Exec(ctx, `
INSERT INTO user_lists (id, user_id, name) VALUES ($1, $2, $3)
`, l.ID, l.UserID, l.Name)
	if err != nil {
		return nil, fmt.Errorf("account: create list: %w", err)
	}
	return l, nil
}

// GetList loads one list. When requireOwnership is true the list must
// belong to ownerID; when false, a list owned by ownerID or a public list
// is accepted — that is Misskey's show/get-memberships lookup rule
// (UserListService.ts): forPublic, or an anonymous caller, reads public
// lists, the owner reads their own.
func (s *Service) GetList(ctx context.Context, listID, ownerID string, requireOwnership bool) (*UserList, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, name, is_public, created_at FROM user_lists WHERE id = $1
`, listID)
	var l UserList
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.IsPublic, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserListNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: get list: %w", err)
	}
	if l.UserID == ownerID {
		return &l, nil
	}
	if requireOwnership {
		// A non-owner must not learn that someone else's private list id
		// exists; Misskey answers NO_SUCH_LIST for those too.
		return nil, ErrUserListNotFound
	}
	if !l.IsPublic {
		return nil, ErrUserListNotFound
	}
	return &l, nil
}

// OwnList returns the list only when it belongs to ownerID, without the
// public fallback; mutating endpoints use this. A list owned by someone
// else reports not-found, so ids of other people's private lists are not
// enumerable through the error.
func (s *Service) OwnList(ctx context.Context, listID, ownerID string) (*UserList, error) {
	row := s.pool.QueryRow(ctx, `
SELECT id, user_id, name, is_public, created_at FROM user_lists WHERE id = $1
`, listID)
	var l UserList
	err := row.Scan(&l.ID, &l.UserID, &l.Name, &l.IsPublic, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserListNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: own list: %w", err)
	}
	if l.UserID != ownerID {
		return nil, ErrUserListNotFound
	}
	return &l, nil
}

// ListListsOf returns the lists one account owns, newest first. With
// onlyPublic it narrows to public lists — the branch Misskey's
// users/lists/list takes when a userId is given.
func (s *Service) ListListsOf(ctx context.Context, userID string, onlyPublic bool) ([]*UserList, error) {
	sql := `SELECT id, user_id, name, is_public, created_at FROM user_lists WHERE user_id = $1`
	if onlyPublic {
		sql += ` AND is_public`
	}
	sql += ` ORDER BY id DESC`
	rows, err := s.pool.Query(ctx, sql, userID)
	if err != nil {
		return nil, fmt.Errorf("account: list user lists: %w", err)
	}
	defer rows.Close()

	out := []*UserList{}
	for rows.Next() {
		var l UserList
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.IsPublic, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan user list: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// ListIDsOf is ListListsOf narrowed to ids, for the timeline handler.
func (s *Service) ListIDsOf(ctx context.Context, userID string) ([]string, error) {
	lists, err := s.ListListsOf(ctx, userID, false)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(lists))
	for _, l := range lists {
		ids = append(ids, l.ID)
	}
	return ids, nil
}

// UpdateList renames a list or flips its visibility. A nil pointer leaves
// the field unchanged, matching users/lists/update's skip-undefined
// semantics. The returned list carries the post-update row.
func (s *Service) UpdateList(ctx context.Context, listID, ownerID string, name *string, isPublic *bool) (*UserList, error) {
	l, err := s.OwnList(ctx, listID, ownerID)
	if err != nil {
		return nil, err
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" || len([]rune(trimmed)) > userListNameMax {
			return nil, ErrListNameInvalid
		}
		if _, err := s.pool.Exec(ctx, `UPDATE user_lists SET name = $1 WHERE id = $2`, trimmed, l.ID); err != nil {
			return nil, fmt.Errorf("account: update list name: %w", err)
		}
		l.Name = trimmed
	}
	if isPublic != nil {
		if _, err := s.pool.Exec(ctx, `UPDATE user_lists SET is_public = $1 WHERE id = $2`, *isPublic, l.ID); err != nil {
			return nil, fmt.Errorf("account: update list visibility: %w", err)
		}
		l.IsPublic = *isPublic
	}
	return l, nil
}

// DeleteList removes a list and, by cascade, its memberships and favourites.
func (s *Service) DeleteList(ctx context.Context, listID, ownerID string) error {
	ct, err := s.pool.Exec(ctx, `DELETE FROM user_lists WHERE id = $1 AND user_id = $2`, listID, ownerID)
	if err != nil {
		return fmt.Errorf("account: delete list: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrUserListNotFound
	}
	return nil
}

// AddMember inserts a membership. Misskey's push enforces the per-list
// member cap and rejects a duplicate with ALREADY_ADDED; pull of a
// non-member silently succeeds, which is why PullMember returns nil for a
// missing row instead of an error.
func (s *Service) AddMember(ctx context.Context, listID, ownerID, memberID string, withReplies bool) error {
	l, err := s.OwnList(ctx, listID, ownerID)
	if err != nil {
		return err
	}
	var n int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM user_list_memberships WHERE user_list_id = $1`, l.ID).Scan(&n); err != nil {
		return fmt.Errorf("account: count members: %w", err)
	}
	if n >= userListMembersLimit {
		return ErrTooManyMembers
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO user_list_memberships (id, user_list_id, user_id, user_list_user_id, with_replies)
VALUES ($1, $2, $3, $4, $5)
`, idgen.New(), l.ID, memberID, ownerID, withReplies)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyAdded
		}
		return fmt.Errorf("account: add member: %w", err)
	}
	return nil
}

// PullMember removes a membership. Not being a member is not an error,
// matching upstream's unconditional DELETE.
func (s *Service) PullMember(ctx context.Context, listID, ownerID, memberID string) error {
	l, err := s.OwnList(ctx, listID, ownerID)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
DELETE FROM user_list_memberships WHERE user_list_id = $1 AND user_id = $2
`, l.ID, memberID); err != nil {
		return fmt.Errorf("account: pull member: %w", err)
	}
	return nil
}

// UpdateMembership flips one member's withReplies switch. A missing
// membership is ErrNotAMember; upstream answers this with a bare 500
// (UserListService.ts L161-163), which is normalised here to a client
// error — decision recorded in docs/HANDOVER.md.
func (s *Service) UpdateMembership(ctx context.Context, listID, ownerID, memberID string, withReplies *bool) error {
	l, err := s.OwnList(ctx, listID, ownerID)
	if err != nil {
		return err
	}
	if withReplies == nil {
		return nil
	}
	ct, err := s.pool.Exec(ctx, `
UPDATE user_list_memberships SET with_replies = $1 WHERE user_list_id = $2 AND user_id = $3
`, *withReplies, l.ID, memberID)
	if err != nil {
		return fmt.Errorf("account: update membership: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotAMember
	}
	return nil
}

// IsMember reports whether userID is in listID.
func (s *Service) IsMember(ctx context.Context, listID, userID string) (bool, error) {
	var m bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM user_list_memberships WHERE user_list_id = $1 AND user_id = $2)
`, listID, userID).Scan(&m)
	if err != nil {
		return false, fmt.Errorf("account: check membership: %w", err)
	}
	return m, nil
}

// MemberIDs returns every member id of listID.
func (s *Service) MemberIDs(ctx context.Context, listID string) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT user_id FROM user_list_memberships WHERE user_list_id = $1`, listID)
	if err != nil {
		return nil, fmt.Errorf("account: member ids: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("account: scan member id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// UserListMembershipQuery pages memberships of one list by membership row
// id, as Misskey's get-memberships does.
type UserListMembershipQuery struct {
	Limit   int
	SinceID string
	UntilID string
	// SinceDate/UntilDate bound created_at, the house-style equivalent of
	// upstream's date cursors.
	SinceDate *time.Time
	UntilDate *time.Time
}

// ListMemberships pages a list's memberships, newest first unless a
// sinceId is given without untilId.
func (s *Service) ListMemberships(ctx context.Context, listID string, q UserListMembershipQuery) ([]*UserListMembership, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_list_id, user_id, with_replies, created_at
FROM user_list_memberships WHERE user_list_id = $1`)
	args := []any{listID}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		fmt.Fprintf(&sql, " AND created_at >= $%d", len(args))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		fmt.Fprintf(&sql, " AND created_at <= $%d", len(args))
	}
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	sql.WriteString(" ORDER BY id " + order + " LIMIT " + strconv.Itoa(limit))

	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: list memberships: %w", err)
	}
	defer rows.Close()

	out := []*UserListMembership{}
	for rows.Next() {
		var m UserListMembership
		if err := rows.Scan(&m.ID, &m.UserListID, &m.UserID, &m.WithReplies, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan membership: %w", err)
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

// FavoriteList records a favourite of a public list. Favouriting an
// already-favourited list is an error, as upstream's favorite is.
func (s *Service) FavoriteList(ctx context.Context, userID string, listID string) error {
	l, err := s.GetList(ctx, listID, "", false)
	if err != nil {
		return err
	}
	if !l.IsPublic {
		return ErrUserListNotPublic
	}
	_, err = s.pool.Exec(ctx, `
INSERT INTO user_list_favorites (id, user_id, user_list_id) VALUES ($1, $2, $3)
`, idgen.New(), userID, l.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrAlreadyFavorited
		}
		return fmt.Errorf("account: favorite list: %w", err)
	}
	return nil
}

// UnfavoriteList removes a favourite. Favouriting first is required; a
// missing favourite is an error (upstream's mislabelled ALREADY_FAVORITED).
func (s *Service) UnfavoriteList(ctx context.Context, userID, listID string) error {
	ct, err := s.pool.Exec(ctx, `
DELETE FROM user_list_favorites WHERE user_id = $1 AND user_list_id = $2
`, userID, listID)
	if err != nil {
		return fmt.Errorf("account: unfavorite list: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFavorited
	}
	return nil
}

// ListFavorites returns the public lists userID favourited.
func (s *Service) ListFavorites(ctx context.Context, userID string) ([]*UserList, error) {
	rows, err := s.pool.Query(ctx, `
SELECT l.id, l.user_id, l.name, l.is_public, l.created_at
FROM user_list_favorites f
JOIN user_lists l ON l.id = f.user_list_id
WHERE f.user_id = $1
ORDER BY f.id DESC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("account: list favorites: %w", err)
	}
	defer rows.Close()

	out := []*UserList{}
	for rows.Next() {
		var l UserList
		if err := rows.Scan(&l.ID, &l.UserID, &l.Name, &l.IsPublic, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan favorite: %w", err)
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// CountListFavorites returns how many users favourited a public list, for
// show's likedCount when forPublic is set.
func (s *Service) CountListFavorites(ctx context.Context, listID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM user_list_favorites WHERE user_list_id = $1`, listID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("account: count favorites: %w", err)
	}
	return n, nil
}

// HasListFavorite reports whether userID favourited listID, for show's
// isLiked flag.
func (s *Service) HasListFavorite(ctx context.Context, userID, listID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM user_list_favorites WHERE user_id = $1 AND user_list_id = $2)
`, userID, listID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check favorite: %w", err)
	}
	return f, nil
}

// CopyPublicList clones a public list and its memberships into a new
// private one owned by userID — Misskey's create-from-public. The source
// list must be public; the clone starts private and empty of favourites.
// Upstream checks "the member blocks the caller" for every member except
// the caller themself and enforces the member cap on the copy, so a
// full source list cannot be cloned at all.
func (s *Service) CopyPublicList(ctx context.Context, userID, sourceListID, name string, blockCheck func(memberID string) (bool, error)) (*UserList, error) {
	src, err := s.GetList(ctx, sourceListID, "", false)
	if err != nil {
		return nil, err
	}
	if !src.IsPublic {
		return nil, ErrUserListNotPublic
	}

	members, err := s.ListMemberships(ctx, src.ID, UserListMembershipQuery{Limit: userListMembersLimit})
	if err != nil {
		return nil, err
	}
	if len(members) >= userListMembersLimit {
		return nil, ErrTooManyMembers
	}

	l, err := s.CreateList(ctx, userID, name)
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		if m.UserID != userID && blockCheck != nil {
			blocked, err := blockCheck(m.UserID)
			if err != nil {
				return nil, err
			}
			if blocked {
				return nil, ErrMemberBlocksYou
			}
		}
		if err := s.AddMember(ctx, l.ID, userID, m.UserID, m.WithReplies); err != nil {
			return nil, err
		}
	}
	return l, nil
}

// CheckAccountMoved reports whether an account has migrated away, the
// prohibitMoved guard six of the list endpoints declare. Misskey answers
// YOUR_ACCOUNT_MOVED, kind permission.
func (s *Service) CheckAccountMoved(ctx context.Context, userID string) error {
	u, err := s.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.MovedToURI != "" {
		return ErrAccountMoved
	}
	return nil
}
