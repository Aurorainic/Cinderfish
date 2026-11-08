package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Aurorainic/cinderfish/internal/idgen"
)

// Page is one authored document, stored with its full content model. The
// upstream viewer at c05e844 renders only the four static block types and
// keeps script/variables opaque, and Cinderfish's viewer follows that, so
// no AiScript runtime is involved.
type Page struct {
	ID                  string
	UserID              string
	Title               string
	Name                string
	Summary             *string
	AlignCenter         bool
	HideTitleWhenPinned bool
	Font                string
	EyeCatchingImageID  *string
	Content             json.RawMessage
	Variables           json.RawMessage
	Script              string
	LikedCount          int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// The page slug pattern of pages/create's paramDef.
var pageNamePattern = regexp.MustCompile(`^[^\s:\/?#\[\]@!$&'()*+,;=\\%\x00-\x20]{1,256}$`)

// pageFonts is the closed set of the font enum.
var pageFonts = map[string]bool{"serif": true, "sans-serif": true}

// Sentinel errors the handlers map onto Misskey's error codes.
var (
	ErrPageNotFound     = errors.New("account: page not found")
	ErrPageNameInvalid  = errors.New("account: page name invalid")
	ErrPageNameTaken    = errors.New("account: page name taken")
	ErrPageFontInvalid  = errors.New("account: page font invalid")
	ErrPageNotOwner     = errors.New("account: page not owned by the caller")
	ErrPageAlreadyLiked = errors.New("account: page already liked")
	ErrPageNotLiked     = errors.New("account: page not liked")
	ErrPageYourPage     = errors.New("account: cannot like your own page")
	ErrPageFileNotFound = errors.New("account: no such eye-catching file")
	ErrPageOtherProfile = errors.New("account: page belongs to another profile")
)

// PageInput carries pages/create's write path. Visibility is always
// public — upstream hardcodes it and exposes no parameter, so none is
// taken here either.
type PageInput struct {
	Title               string
	Name                string
	Summary             *string
	AlignCenter         bool
	HideTitleWhenPinned bool
	Font                string
	EyeCatchingImageID  *string
	Content             json.RawMessage
	Variables           json.RawMessage
	Script              string
}

// pageColumns without the table prefix; queries that join under an alias
// use pageColumnsC.
const pageColumns = `id, user_id, updated_at, title, name, summary, align_center,
       hide_title_when_pinned, font, eye_catching_image_id, content, variables,
       script, liked_count, created_at`

const pageColumnsC = `c.id, c.user_id, c.updated_at, c.title, c.name, c.summary, c.align_center,
       c.hide_title_when_pinned, c.font, c.eye_catching_image_id, c.content, c.variables,
       c.script, c.liked_count, c.created_at`

func scanPage(row pgx.Row) (*Page, error) {
	var p Page
	var eyeCatching, summary *string
	var content, variables []byte
	err := row.Scan(&p.ID, &p.UserID, &p.UpdatedAt, &p.Title, &p.Name, &summary,
		&p.AlignCenter, &p.HideTitleWhenPinned, &p.Font, &eyeCatching,
		&content, &variables, &p.Script, &p.LikedCount, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("account: scan page: %w", err)
	}
	p.Summary = summary
	p.EyeCatchingImageID = eyeCatching
	p.Content = json.RawMessage(content)
	p.Variables = json.RawMessage(variables)
	return &p, nil
}

func scanPages(rows pgx.Rows) ([]*Page, error) {
	defer rows.Close()
	out := []*Page{}
	for rows.Next() {
		var p Page
		var eyeCatching, summary *string
		var content, variables []byte
		if err := rows.Scan(&p.ID, &p.UserID, &p.UpdatedAt, &p.Title, &p.Name, &summary,
			&p.AlignCenter, &p.HideTitleWhenPinned, &p.Font, &eyeCatching,
			&content, &variables, &p.Script, &p.LikedCount, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan page: %w", err)
		}
		p.Summary = summary
		p.EyeCatchingImageID = eyeCatching
		p.Content = json.RawMessage(content)
		p.Variables = json.RawMessage(variables)
		out = append(out, &p)
	}
	return out, rows.Err()
}

// validatePageName applies the paramDef pattern and the font enum.
func validatePageInput(title, name, font string) error {
	if title == "" || len([]rune(title)) > 256 {
		return ErrPageNameInvalid
	}
	if !pageNamePattern.MatchString(name) {
		return ErrPageNameInvalid
	}
	if font != "" && !pageFonts[font] {
		return ErrPageFontInvalid
	}
	return nil
}

// CreatePage stores a page. The (userId, name) pair is unique; a second
// use answers NAME_ALREADY_EXISTS. The eye-catching image must belong to
// the caller, as upstream looks the file up per owner.
func (s *Service) CreatePage(ctx context.Context, userID string, in PageInput) (*Page, error) {
	if err := validatePageInput(in.Title, in.Name, in.Font); err != nil {
		return nil, err
	}
	font := in.Font
	if font == "" {
		font = "sans-serif"
	}
	var eyeCatching *string
	if in.EyeCatchingImageID != nil && *in.EyeCatchingImageID != "" {
		var owner string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id FROM drive_files WHERE id = $1`, *in.EyeCatchingImageID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != userID) {
			return nil, ErrPageFileNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("account: page banner lookup: %w", err)
		}
		eyeCatching = in.EyeCatchingImageID
	}

	var summary *string
	if in.Summary != nil {
		trimmed := strings.TrimSpace(*in.Summary)
		if len([]rune(trimmed)) > 256 {
			return nil, ErrPageNameInvalid
		}
		if trimmed != "" {
			summary = &trimmed
		}
	}
	content := in.Content
	if len(content) == 0 {
		content = json.RawMessage("[]")
	}
	variables := in.Variables
	if len(variables) == 0 {
		variables = json.RawMessage("[]")
	}

	p := &Page{
		ID:                  idgen.New(),
		UserID:              userID,
		Title:               in.Title,
		Name:                in.Name,
		Summary:             summary,
		AlignCenter:         in.AlignCenter,
		HideTitleWhenPinned: in.HideTitleWhenPinned,
		Font:                font,
		EyeCatchingImageID:  eyeCatching,
		Content:             content,
		Variables:           variables,
		Script:              in.Script,
		CreatedAt:           time.Now(),
	}
	p.UpdatedAt = p.CreatedAt
	if _, err := s.pool.Exec(ctx, `
INSERT INTO pages (id, user_id, title, name, summary, align_center,
                   hide_title_when_pinned, font, eye_catching_image_id,
                   content, variables, script, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
`, p.ID, p.UserID, p.Title, p.Name, p.Summary, p.AlignCenter,
		p.HideTitleWhenPinned, p.Font, p.EyeCatchingImageID,
		[]byte(p.Content), []byte(p.Variables), p.Script, p.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPageNameTaken
		}
		return nil, fmt.Errorf("account: create page: %w", err)
	}
	return s.GetPage(ctx, p.ID)
}

// PagePatch is the partial merge shape of pages/update; every optional
// field travels with a presence flag, since the booleans and the nullable
// summary/banners must distinguish omitted from cleared.
type PagePatch struct {
	Title               *string
	Name                *string
	Summary             *string
	SummaryPresent      bool
	AlignCenter         *bool
	HideTitleWhenPinned *bool
	Font                *string
	EyeCatchingImageID  *string
	EyeCatchingPresent  bool
	Content             json.RawMessage
	ContentPresent      bool
	Variables           json.RawMessage
	VariablesPresent    bool
	Script              *string
	ScriptPresent       bool
}

// UpdatePage applies the patch to an owned page (or, for an administrator,
// any page, mirroring pages/delete's moderator branch). The uniqueness
// check mirrors create's.
func (s *Service) UpdatePage(ctx context.Context, pageID, editorID string, isAdmin bool, patch *PagePatch) error {
	if _, err := s.OwnOrModeratedPage(ctx, pageID, editorID, isAdmin); err != nil {
		return err
	}
	sets := []string{}
	args := []any{}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if patch.Title != nil {
		if *patch.Title == "" || len([]rune(*patch.Title)) > 256 {
			return ErrPageNameInvalid
		}
		add("title", *patch.Title)
	}
	if patch.Name != nil {
		if !pageNamePattern.MatchString(*patch.Name) {
			return ErrPageNameInvalid
		}
		add("name", *patch.Name)
	}
	if patch.SummaryPresent && patch.Summary != nil {
		trimmed := strings.TrimSpace(*patch.Summary)
		if len([]rune(trimmed)) > 256 {
			return ErrPageNameInvalid
		}
		var summary *string
		if trimmed != "" {
			summary = &trimmed
		}
		add("summary", summary)
	} else if patch.SummaryPresent {
		add("summary", nil)
	}
	if patch.AlignCenter != nil {
		add("align_center", *patch.AlignCenter)
	}
	if patch.HideTitleWhenPinned != nil {
		add("hide_title_when_pinned", *patch.HideTitleWhenPinned)
	}
	if patch.Font != nil {
		if !pageFonts[*patch.Font] {
			return ErrPageFontInvalid
		}
		add("font", *patch.Font)
	}
	if patch.EyeCatchingPresent {
		var eyeCatching *string
		if patch.EyeCatchingImageID != nil && *patch.EyeCatchingImageID != "" {
			var owner string
			err := s.pool.QueryRow(ctx,
				`SELECT user_id FROM drive_files WHERE id = $1`, *patch.EyeCatchingImageID).Scan(&owner)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != editorID) {
				return ErrPageFileNotFound
			}
			if err != nil {
				return fmt.Errorf("account: page banner lookup: %w", err)
			}
			eyeCatching = patch.EyeCatchingImageID
		}
		add("eye_catching_image_id", eyeCatching)
	}
	if patch.ContentPresent {
		add("content", []byte(patch.Content))
	}
	if patch.VariablesPresent {
		add("variables", []byte(patch.Variables))
	}
	if patch.ScriptPresent {
		if patch.Script == nil || len([]rune(*patch.Script)) > 16384 {
			return ErrPageNameInvalid
		}
		add("script", *patch.Script)
	}
	if len(sets) > 0 {
		add("updated_at", time.Now())
		args = append(args, pageID)
		if _, err := s.pool.Exec(ctx,
			`UPDATE pages SET `+strings.Join(sets, ", ")+` WHERE id = $`+fmt.Sprint(len(args)),
			args...); err != nil {
			if isUniqueViolation(err) {
				return ErrPageNameTaken
			}
			return fmt.Errorf("account: update page: %w", err)
		}
	}
	return nil
}

// OwnOrModeratedPage loads the page when the editor is the owner or an
// administrator; anyone else answers not-found, so foreign page ids stay
// unenumerable through the error.
func (s *Service) OwnOrModeratedPage(ctx context.Context, pageID, editorID string, isAdmin bool) (*Page, error) {
	p, err := s.GetPage(ctx, pageID)
	if err != nil {
		return nil, err
	}
	if p.UserID != editorID && !isAdmin {
		return nil, ErrPageNotFound
	}
	return p, nil
}

// DeletePage removes a page; the page pin of the deleted row clears by
// cascade of the FK.
func (s *Service) DeletePage(ctx context.Context, pageID, editorID string, isAdmin bool) error {
	if _, err := s.OwnOrModeratedPage(ctx, pageID, editorID, isAdmin); err != nil {
		return err
	}
	ct, err := s.pool.Exec(ctx, `DELETE FROM pages WHERE id = $1`, pageID)
	if err != nil {
		return fmt.Errorf("account: delete page: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrPageNotFound
	}
	return nil
}

// GetPage loads any page by id — pages/show has no visibility check
// upstream, and the port reproduces that.
func (s *Service) GetPage(ctx context.Context, pageID string) (*Page, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+pageColumns+` FROM pages WHERE id = $1`, pageID)
	return scanPage(row)
}

// GetPageByName resolves pages/show's {name, username} form: local
// authors only, as upstream's lookup keeps host IS NULL.
func (s *Service) GetPageByName(ctx context.Context, username, name string) (*Page, error) {
	row := s.pool.QueryRow(ctx, `
SELECT `+pageColumnsC+`
FROM pages c
JOIN users u ON u.id = c.user_id
WHERE c.name = $1 AND u.username_lower = lower($2) AND u.host IS NULL
`, name, username)
	return scanPage(row)
}

// ListPages answers i/pages: the caller's pages, newest first.
func (s *Service) ListPages(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Page, error) {
	return s.listPagesWhere(ctx, userID, limit, sinceID, untilID)
}

// ListUserPages answers users/pages: the public pages of one account.
func (s *Service) ListUserPages(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Page, error) {
	return s.listPagesWhereFiltered(ctx, userID, limit, sinceID, untilID)
}

// listPagesWhere pages pages of one owner without the visibility filter —
// the i/pages branch, which carries private pages too.
func (s *Service) listPagesWhere(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Page, error) {
	limit = normalizePageLimit(limit)
	var sql strings.Builder
	sql.WriteString(`SELECT ` + pageColumns + ` FROM pages WHERE user_id = $1`)
	args := []any{userID}
	if sinceID != "" {
		args = append(args, sinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	order := "DESC"
	if sinceID != "" && untilID == "" {
		order = "ASC"
	}
	args = append(args, limit)
	sql.WriteString(" ORDER BY id " + order + " LIMIT $" + fmt.Sprint(len(args)))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: list pages: %w", err)
	}
	return scanPages(rows)
}

// listPagesWhereFiltered is the users/pages branch: only visibility =
// 'public' rows travel.
func (s *Service) listPagesWhereFiltered(ctx context.Context, userID string, limit int, sinceID, untilID string) ([]*Page, error) {
	limit = normalizePageLimit(limit)
	var sql strings.Builder
	sql.WriteString(`SELECT ` + pageColumns + ` FROM pages WHERE user_id = $1 AND visibility = 'public'`)
	args := []any{userID}
	if sinceID != "" {
		args = append(args, sinceID)
		fmt.Fprintf(&sql, " AND id > $%d", len(args))
	}
	if untilID != "" {
		args = append(args, untilID)
		fmt.Fprintf(&sql, " AND id < $%d", len(args))
	}
	order := "DESC"
	if sinceID != "" && untilID == "" {
		order = "ASC"
	}
	args = append(args, limit)
	sql.WriteString(" ORDER BY id " + order + " LIMIT $" + fmt.Sprint(len(args)))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: list user pages: %w", err)
	}
	return scanPages(rows)
}

// normalizePageLimit applies the shared 1..100 clamp with the pages
// default of 10.
func normalizePageLimit(limit int) int {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return limit
}

// FeaturedPages answers pages/featured: the ten most-liked public pages,
// no pagination, exactly upstream's query.
func (s *Service) FeaturedPages(ctx context.Context) ([]*Page, error) {
	rows, err := s.pool.Query(ctx, `
SELECT `+pageColumns+` FROM pages
WHERE visibility = 'public' AND liked_count > 0
ORDER BY liked_count DESC LIMIT 10
`)
	if err != nil {
		return nil, fmt.Errorf("account: featured pages: %w", err)
	}
	return scanPages(rows)
}

// LikePage records a like with its counter in one transaction; liking
// your own page is YOUR_PAGE.
func (s *Service) LikePage(ctx context.Context, userID, pageID string) error {
	p, err := s.GetPage(ctx, pageID)
	if err != nil {
		return err
	}
	if p.UserID == userID {
		return ErrPageYourPage
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: like page: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO page_likes (id, user_id, page_id) VALUES ($1, $2, $3)`,
		idgen.New(), userID, pageID); err != nil {
		if isUniqueViolation(err) {
			return ErrPageAlreadyLiked
		}
		return fmt.Errorf("account: insert page like: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE pages SET liked_count = liked_count + 1 WHERE id = $1`, pageID); err != nil {
		return fmt.Errorf("account: bump page likes: %w", err)
	}
	return tx.Commit(ctx)
}

// UnlikePage removes a like; unliking a page never liked is NOT_LIKED.
func (s *Service) UnlikePage(ctx context.Context, userID, pageID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("account: unlike page: %w", err)
	}
	defer tx.Rollback(ctx)
	ct, err := tx.Exec(ctx,
		`DELETE FROM page_likes WHERE user_id = $1 AND page_id = $2`, userID, pageID)
	if err != nil {
		return fmt.Errorf("account: delete page like: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrPageNotLiked
	}
	if _, err := tx.Exec(ctx,
		`UPDATE pages SET liked_count = GREATEST(liked_count - 1, 0) WHERE id = $1`, pageID); err != nil {
		return fmt.Errorf("account: drop page likes: %w", err)
	}
	return tx.Commit(ctx)
}

// HasPageLike reports whether userID liked pageID, for isLiked.
func (s *Service) HasPageLike(ctx context.Context, userID, pageID string) (bool, error) {
	var f bool
	err := s.pool.QueryRow(ctx, `
SELECT EXISTS (SELECT 1 FROM page_likes WHERE user_id = $1 AND page_id = $2)
`, userID, pageID).Scan(&f)
	if err != nil {
		return false, fmt.Errorf("account: check page like: %w", err)
	}
	return f, nil
}

// PageLikeEntry is one i/page-likes row: the like id plus the packed page.
type PageLikeEntry struct {
	ID   string
	Page *Page
}

// ListPageLikes answers i/page-likes.
func (s *Service) ListPageLikes(ctx context.Context, userID string, limit int, untilID, sinceID string) ([]*PageLikeEntry, error) {
	limit = normalizePageLimit(limit)
	var sql strings.Builder
	sql.WriteString(`
SELECT f.id, ` + pageColumnsC + `
FROM page_likes f
JOIN pages c ON c.id = f.page_id
WHERE f.user_id = $1`)
	args := []any{userID}
	if untilID != "" {
		args = append(args, untilID)
		fmt.Fprintf(&sql, " AND f.id < $%d", len(args))
	}
	if sinceID != "" {
		args = append(args, sinceID)
		fmt.Fprintf(&sql, " AND f.id > $%d", len(args))
	}
	sql.WriteString(" ORDER BY f.id DESC LIMIT " + fmt.Sprint(limit))
	rows, err := s.pool.Query(ctx, sql.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("account: page likes: %w", err)
	}
	defer rows.Close()

	out := []*PageLikeEntry{}
	for rows.Next() {
		var e PageLikeEntry
		var p Page
		var eyeCatching, summary *string
		var content, variables []byte
		if err := rows.Scan(&e.ID, &p.ID, &p.UserID, &p.UpdatedAt, &p.Title, &p.Name, &summary,
			&p.AlignCenter, &p.HideTitleWhenPinned, &p.Font, &eyeCatching,
			&content, &variables, &p.Script, &p.LikedCount, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("account: scan page like: %w", err)
		}
		p.Summary = summary
		p.EyeCatchingImageID = eyeCatching
		p.Content = json.RawMessage(content)
		p.Variables = json.RawMessage(variables)
		e.Page = &p
		out = append(out, &e)
	}
	return out, rows.Err()
}

// PinnedPageID returns the page the profile pins, nil when none.
func (s *Service) PinnedPageID(ctx context.Context, userID string) (*string, error) {
	var id *string
	err := s.pool.QueryRow(ctx,
		`SELECT pinned_page_id FROM user_profiles WHERE user_id = $1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("account: pinned page: %w", err)
	}
	return id, nil
}

// SetPinnedPage applies i/update's pinnedPageId: foreign pages answer
// ErrPageOtherProfile with upstream's id, null clears the pin.
func (s *Service) SetPinnedPage(ctx context.Context, userID string, pageID *string) error {
	if pageID != nil && *pageID != "" {
		var owner string
		err := s.pool.QueryRow(ctx,
			`SELECT user_id FROM pages WHERE id = $1`, *pageID).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != userID) {
			return ErrPageOtherProfile
		}
		if err != nil {
			return fmt.Errorf("account: pin page lookup: %w", err)
		}
	}
	var clear any
	if pageID != nil && *pageID != "" {
		clear = *pageID
	}
	if _, err := s.pool.Exec(ctx,
		`UPDATE user_profiles SET pinned_page_id = $1 WHERE user_id = $2`,
		clear, userID); err != nil {
		return fmt.Errorf("account: set pinned page: %w", err)
	}
	return nil
}
