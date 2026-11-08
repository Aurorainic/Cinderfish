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

// GalleryPost is one gallery entry: drive images with a title and an
// optional MFM description. No AiScript, no federation, no notifications,
// matching the upstream model.
type GalleryPost struct {
	ID          string
	UserID      string
	Title       string
	Description *string
	FileIDs     []string
	IsSensitive bool
	LikedCount  int
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}

// galleryMaxFiles is the fileIds cap of gallery/posts/create.
const galleryMaxFiles = 32

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrGalleryPostNotFound = errors.New("account: gallery post not found")
	ErrGalleryNoFiles      = errors.New("account: gallery post needs files")
	ErrGalleryTitleInvalid = errors.New("account: gallery title invalid")
	ErrGalleryAlreadyLiked = errors.New("account: gallery post already liked")
	ErrGalleryNotLiked     = errors.New("account: gallery post not liked")
	ErrGalleryYourPost     = errors.New("account: cannot like your own gallery post")
)

// GalleryInput carries gallery/posts/create's write path.
type GalleryInput struct {
	Title       string
	Description *string
	FileIDs     []string
	IsSensitive bool
}

// CreatePost stores a gallery post. The file list is filtered to files
// the caller owns, order preserved; zero survivors is the bare error
// upstream 500s with, mirrored as ErrGalleryNoFiles (a recorded
// divergence: upstream declares no id for it).
func (s *Service) CreatePost(ctx context.Context, userID string, in GalleryInput) (*GalleryPost, error) {
	fileIDs, err := s.ownedFileIDs(ctx, userID, in.FileIDs)
	if err != nil {
		return nil, err
	}
	if len(fileIDs) == 0 {
		return nil, ErrGalleryNoFiles
	}
	if in.Title == "" {
		return nil, ErrGalleryTitleInvalid
	}
	var description *string
	if in.Description != nil {
		trimmed := strings.TrimSpace(*in.Description)
		if len([]rune(trimmed)) > 2048 {
			return nil, ErrGalleryTitleInvalid
		}
		if trimmed != "" {
			description = &trimmed
		}
	}

	p := &GalleryPost{
		ID:          idgen.New(),
		UserID:      userID,
		Title:       in.Title,
		Description: description,
		FileIDs:     fileIDs,
		IsSensitive: in.IsSensitive,
		CreatedAt:   time.Now(),
	}
	if _, err := s.pool.Exec(ctx, `
INSERT INTO gallery_posts (id, user_id, title, description, file_ids, is_sensitive)
VALUES ($1, $2, $3, $4, $5, $6)
`, p.ID, p.UserID, p.Title, p.Description, galleryFileIDsParam(p.FileIDs), p.IsSensitive); err != nil {
		return nil, fmt.Errorf("account: create gallery post: %w", err)
	}
	return p, nil
}

// ownedFileIDs filters a requested id list to the caller's drive files,
// preserving request order and dropping duplicates.
func (s *Service) ownedFileIDs(ctx context.Context, userID string, ids []string) ([]string, error) {
	if len(ids) > galleryMaxFiles {
		ids = ids[:galleryMaxFiles]
	}
	out := []string{}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		var owner string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id FROM drive_files WHERE id = $1`, id).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("account: gallery file lookup: %w", err)
		}
		if owner == userID {
			out = append(out, id)
		}
	}
	return out, nil
}

// galleryFileIDsParam marshals a string slice for a jsonb parameter; nil
// stays the empty array.
func galleryFileIDsParam(ids []string) []byte {
	if ids == nil {
		return []byte("[]")
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return []byte("[]")
	}
	return raw
}

// UpdateGalleryPatch is the partial merge of gallery/posts/update, with
// the two presence flags separating "omitted" from "cleared".
type GalleryUpdatePatch struct {
	Title              *string
	Description        *string
	DescriptionPresent bool
	FileIDs            []string
	FileIDsPresent     bool
	IsSensitive        *bool
}

// UpdatePost applies the patch to an owned post. Upstream scopes the
// UPDATE by (id, userId), so a non-owner's write silently matches no rows
// and still returns the post — this port answers the ownership question
// upfront, the recording note lives in docs/HANDOVER.md. The like counter
// and updatedAt travel together from the write.
func (s *Service) UpdatePost(ctx context.Context, postID, ownerID string, patch *GalleryUpdatePatch) (*GalleryPost, error) {
	if _, err := s.GetGalleryPost(ctx, postID); err != nil {
		return nil, err
	}
	if _, err := s.OwnGalleryPost(ctx, postID, ownerID); err != nil {
		return nil, err
	}
	if patch == nil {
		return s.GetGalleryPost(ctx, postID)
	}
	sets := []string{}
	args := []any{}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if patch.Title != nil {
		if *patch.Title == "" {
			return nil, ErrGalleryTitleInvalid
		}
		add("title", *patch.Title)
	}
	if patch.DescriptionPresent {
		var desc *string
		if patch.Description != nil {
			trimmed := strings.TrimSpace(*patch.Description)
			if len([]rune(trimmed)) > 2048 {
				return nil, ErrGalleryTitleInvalid
			}
			if trimmed != "" {
				desc = &trimmed
			}
		}
		add("description", desc)
	}
	if patch.FileIDsPresent {
		fileIDs, err := s.ownedFileIDs(ctx, ownerID, patch.FileIDs)
		if err != nil {
			return nil, err
		}
		if len(fileIDs) == 0 {
			return nil, ErrGalleryNoFiles
		}
		add("file_ids", galleryFileIDsParam(fileIDs))
	}
	if patch.IsSensitive != nil {
		add("is_sensitive", *patch.IsSensitive)
	}
	if len(sets) > 0 {
		add("updated_at", time.Now())
		args = append(args, postID)
		if _, err := s.pool.Exec(ctx,
			`UPDATE gallery_posts SET `+strings.Join(sets, ", ")+` WHERE id = $`+fmt.Sprint(len(args)),
			args...); err != nil {
			return nil, fmt.Errorf("account: update gallery post: %w", err)
		}
	}
	return s.GetGalleryPost(ctx, postID)
}

// DeletePost removes a gallery post; the owner or an administrator may.
func (s *Service) DeletePost(ctx context.Context, postID, actorID string, isAdmin bool) error {
	if !isAdmin {
		if _, err := s.OwnGalleryPost(ctx, postID, actorID); err != nil {
			return err
		}
	}
	ct, err := s.pool.Exec(ctx, `DELETE FROM gallery_posts WHERE id = $1`, postID)
	if err != nil {
		return fmt.Errorf("account: delete gallery post: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrGalleryPostNotFound
	}
	return nil
}

// GetGalleryPost loads any post — gallery/posts/show is public.
func (s *Service) GetGalleryPost(ctx context.Context, postID string) (*GalleryPost, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
		 FROM gallery_posts WHERE id = $1`, postID)
	return scanGalleryPost(row)
}

// OwnGalleryPost loads a post that must belong to ownerID.
func (s *Service) OwnGalleryPost(ctx context.Context, postID, ownerID string) (*GalleryPost, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
		 FROM gallery_posts WHERE id = $1`, postID)
	p, err := scanGalleryPost(row)
	if err != nil {
		return nil, err
	}
	if p.UserID != ownerID {
		return nil, ErrGalleryPostNotFound
	}
	return p, nil
}

func scanGalleryPost(row pgx.Row) (*GalleryPost, error) {
	var p GalleryPost
	var fileIDs []byte
	err := row.Scan(&p.ID, &p.UserID, &p.Title, &p.Description, &fileIDs,
		&p.IsSensitive, &p.LikedCount, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGalleryPostNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan gallery post: %w", err)
	}
	p.FileIDs = decodeTextArray(fileIDs)
	return &p, nil
}

func scanGalleryPosts(rows pgx.Rows) ([]*GalleryPost, error) {
	defer rows.Close()
	out := []*GalleryPost{}
	for rows.Next() {
		var p GalleryPost
		var fileIDs []byte
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Description, &fileIDs,
			&p.IsSensitive, &p.LikedCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("account: scan gallery post: %w", err)
		}
		p.FileIDs = decodeTextArray(fileIDs)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// GalleryQuery pages a gallery listing by post id.
type GalleryQuery struct {
	Limit     int
	SinceID   string
	UntilID   string
	SinceDate *time.Time
	UntilDate *time.Time
}

// listGallery runs the public gallery listing: pagination conditions and
// arguments are appended together, the house rule for dynamic SQL.
func (s *Service) listGallery(ctx context.Context, q GalleryQuery) ([]*GalleryPost, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
FROM gallery_posts`)
	args := []any{}
	started := false
	where := func(cond string) {
		if started {
			sql.WriteString(" AND " + cond)
		} else {
			sql.WriteString(" WHERE " + cond)
			started = true
		}
	}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		where(fmt.Sprintf("id > $%d", len(args)))
	}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		where(fmt.Sprintf("id < $%d", len(args)))
	}
	if q.SinceDate != nil {
		args = append(args, *q.SinceDate)
		where(fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if q.UntilDate != nil {
		args = append(args, *q.UntilDate)
		where(fmt.Sprintf("created_at <= $%d", len(args)))
	}
	order := "DESC"
	if q.SinceID != "" && q.UntilID == "" {
		order = "ASC"
	}
	args = append(args, limit)
	sql.WriteString(" ORDER BY id " + order + " LIMIT $" + fmt.Sprint(len(args)))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: gallery listing: %w", err)
	}
	return scanGalleryPosts(rows)
}

// ListGalleryPosts pages every gallery post — the explore feed.
func (s *Service) ListGalleryPosts(ctx context.Context, q GalleryQuery) ([]*GalleryPost, error) {
	return s.listGallery(ctx, q)
}

// ListUserGalleryPosts answers users/gallery/posts: the public listing of
// one account's posts.
func (s *Service) ListUserGalleryPosts(ctx context.Context, userID string, q GalleryQuery) ([]*GalleryPost, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var sql strings.Builder
	sql.WriteString(`
SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
FROM gallery_posts WHERE user_id = $1`)
	args := []any{userID}
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
	args = append(args, limit)
	sql.WriteString(" ORDER BY id " + order + " LIMIT $" + fmt.Sprint(len(args)))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: user gallery posts: %w", err)
	}
	return scanGalleryPosts(rows)
}

// ListMyGalleryPosts answers i/gallery/posts.
func (s *Service) ListMyGalleryPosts(ctx context.Context, userID string, q GalleryQuery) ([]*GalleryPost, error) {
	return s.ListUserGalleryPosts(ctx, userID, q)
}

// LikeGalleryPost records a like with its counter in one transaction.
// Liking your own post is YOUR_POST upstream; a second like is
// ALREADY_LIKED.
func (s *Service) LikeGalleryPost(ctx context.Context, userID, postID string) error {
	p, err := s.GetGalleryPost(ctx, postID)
	if err != nil {
		return err
	}
	if p.UserID == userID {
		return ErrGalleryYourPost
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: like gallery post: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
INSERT INTO gallery_likes (id, user_id, post_id) VALUES ($1, $2, $3)
`, idgen.New(), userID, postID); err != nil {
		if isUniqueViolation(err) {
			return ErrGalleryAlreadyLiked
		}
		return fmt.Errorf("account: insert gallery like: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE gallery_posts SET liked_count = liked_count + 1 WHERE id = $1`, postID); err != nil {
		return fmt.Errorf("account: bump gallery likes: %w", err)
	}
	return tx.Commit(ctx)
}

// UnlikeGalleryPost removes a like; unliking a post never liked is
// NOT_LIKED.
func (s *Service) UnlikeGalleryPost(ctx context.Context, userID, postID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: unlike gallery post: %w", err)
	}
	defer tx.Rollback(ctx)
	ct, err := tx.Exec(ctx,
		`DELETE FROM gallery_likes WHERE user_id = $1 AND post_id = $2`, userID, postID)
	if err != nil {
		return fmt.Errorf("account: delete gallery like: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrGalleryNotLiked
	}
	if _, err := tx.Exec(ctx,
		`UPDATE gallery_posts SET liked_count = GREATEST(liked_count - 1, 0) WHERE id = $1`, postID); err != nil {
		return fmt.Errorf("account: drop gallery likes: %w", err)
	}
	return tx.Commit(ctx)
}

// HasGalleryLike reports whether userID liked postID, for isLiked.
func (s *Service) HasGalleryLike(ctx context.Context, userID, postID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM gallery_likes WHERE user_id = $1 AND post_id = $2)
`, userID, postID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check gallery like: %w", err)
	}
	return f, nil
}

// ListGalleryLikes answers i/gallery/likes: {id, post} pairs, newest
// like first.
type GalleryLikeEntry struct {
	ID   string
	Post *GalleryPost
}

func (s *Service) ListGalleryLikes(ctx context.Context, userID string, q GalleryQuery) ([]*GalleryLikeEntry, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	var sql strings.Builder
	sql.WriteString(`
SELECT f.id, p.id, p.user_id, p.title, p.description, p.file_ids, p.is_sensitive, p.liked_count, p.created_at, p.updated_at
FROM gallery_likes f
JOIN gallery_posts p ON p.id = f.post_id
WHERE f.user_id = $1`)
	args := []any{userID}
	if q.UntilID != "" {
		args = append(args, q.UntilID)
		fmt.Fprintf(&sql, " AND f.id < $%d", len(args))
	}
	if q.SinceID != "" {
		args = append(args, q.SinceID)
		fmt.Fprintf(&sql, " AND f.id > $%d", len(args))
	}
	sql.WriteString(" ORDER BY f.id DESC LIMIT " + fmt.Sprint(limit))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: gallery likes: %w", err)
	}
	defer rows.Close()

	out := []*GalleryLikeEntry{}
	for rows.Next() {
		var e GalleryLikeEntry
		var p GalleryPost
		var fileIDs []byte
		if err := rows.Scan(&e.ID, &p.ID, &p.UserID, &p.Title, &p.Description, &fileIDs,
			&p.IsSensitive, &p.LikedCount, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("account: scan gallery like: %w", err)
		}
		p.FileIDs = decodeTextArray(fileIDs)
		e.Post = &p
		out = append(out, &e)
	}
	return out, rows.Err()
}

// FeaturedGallery posts answers gallery/featured with a SQL ranking: the
// three-day window ordered by likes, the documented stand-in for
// upstream's Redis sorted-set ranking.
func (s *Service) FeaturedGallery(ctx context.Context, limit int, untilID string) ([]*GalleryPost, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	window := time.Now().Add(-3 * 24 * time.Hour)
	args := []any{window}
	query := `
SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
FROM gallery_posts
WHERE created_at >= $1`
	if untilID != "" {
		args = append(args, untilID)
		query += " AND id < $" + fmt.Sprint(len(args))
	}
	args = append(args, limit)
	query += " ORDER BY liked_count DESC, id DESC LIMIT $" + fmt.Sprint(len(args))
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("account: featured gallery: %w", err)
	}
	return scanGalleryPosts(rows)
}

// PopularGallery answers gallery/popular, upstream's hard-set of ten:
// the most-liked posts regardless of age.
func (s *Service) PopularGallery(ctx context.Context) ([]*GalleryPost, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, user_id, title, description, file_ids, is_sensitive, liked_count, created_at, updated_at
FROM gallery_posts
WHERE liked_count > 0
ORDER BY liked_count DESC, id DESC
LIMIT 10
`)
	if err != nil {
		return nil, fmt.Errorf("account: popular gallery: %w", err)
	}
	return scanGalleryPosts(rows)
}
