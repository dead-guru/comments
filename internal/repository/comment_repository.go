package repository

import (
	"context"
	"database/sql"
	"fmt"

	"deadcomments/internal/domain"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, c *domain.Comment) error {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO comments(id, site_id, page_id, parent_id, root_id, depth, path, author_name, author_display_name, identity_id, tripcode_public, tripcode_kind, author_email_hash, author_avatar_hash, author_website, body_markdown, body_html, status, ip_hash, user_agent_hash, metadata_json, created_at, updated_at, edited_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.ID, c.SiteID, c.PageID, c.ParentID, c.RootID, c.Depth, c.Path, c.AuthorName, c.AuthorDisplayName, c.IdentityID, c.TripcodePublic, c.TripcodeKind, c.AuthorEmailHash, c.AuthorAvatarHash, c.AuthorWebsite, c.BodyMarkdown, c.BodyHTML, c.Status, c.IPHash, c.UserAgentHash, c.MetadataJSON, now, now, nil)
	if err != nil {
		return err
	}
	c.CreatedAt = parseTime(now)
	c.UpdatedAt = parseTime(now)
	return nil
}

func (r *CommentRepository) UpdateTreeFields(ctx context.Context, id string, rootID *string, depth int, path string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE comments SET root_id=?, depth=?, path=?, updated_at=? WHERE id=?`, rootID, depth, path, nowString(), id)
	return err
}

func (r *CommentRepository) ByID(ctx context.Context, id string) (*domain.Comment, error) {
	row := r.db.QueryRowContext(ctx, commentSelectSQL+` WHERE comments.id=?`, id)
	return scanComment(row)
}

func (r *CommentRepository) ApprovedByPage(ctx context.Context, pageID int64) ([]*domain.Comment, error) {
	return r.list(ctx, `WHERE comments.page_id=? AND comments.status='approved' ORDER BY COALESCE(root_comments.created_at, comments.created_at), comments.created_at, comments.id`, pageID)
}

func (r *CommentRepository) ByPage(ctx context.Context, pageID int64) ([]*domain.Comment, error) {
	return r.list(ctx, `WHERE comments.page_id=? ORDER BY comments.created_at DESC`, pageID)
}

func (r *CommentRepository) List(ctx context.Context, status, search string, siteID, pageID *int64, limit int) ([]*domain.Comment, error) {
	q := `WHERE 1=1`
	args := []any{}
	if status != "" {
		q += ` AND comments.status=?`
		args = append(args, status)
	}
	if siteID != nil {
		q += ` AND comments.site_id=?`
		args = append(args, *siteID)
	}
	if pageID != nil {
		q += ` AND comments.page_id=?`
		args = append(args, *pageID)
	}
	if search != "" {
		q += ` AND (comments.body_markdown LIKE ? OR comments.author_name LIKE ? OR comments.author_display_name LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	q += ` ORDER BY comments.created_at DESC LIMIT ?`
	args = append(args, limit)
	return r.list(ctx, q, args...)
}

func (r *CommentRepository) CountByStatus(ctx context.Context, status domain.CommentStatus) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE status=?`, status).Scan(&n)
	return n, err
}

func (r *CommentRepository) CountTodayByStatus(ctx context.Context, status domain.CommentStatus) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE status=? AND created_at >= date('now')`, status).Scan(&n)
	return n, err
}

func (r *CommentRepository) UpdateStatus(ctx context.Context, id string, status domain.CommentStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE comments SET status=?, updated_at=? WHERE id=?`, status, nowString(), id)
	return err
}

func (r *CommentRepository) UpdateBody(ctx context.Context, id, markdown, html string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE comments SET body_markdown=?, body_html=?, edited_at=?, updated_at=? WHERE id=?`, markdown, html, nowString(), nowString(), id)
	return err
}

func (r *CommentRepository) RecentSameIP(ctx context.Context, ipHash, body string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE ip_hash=? AND body_markdown=? AND created_at >= datetime('now', '-1 day')`, ipHash, body).Scan(&n)
	return n, err
}

func (r *CommentRepository) RecentIPCount(ctx context.Context, ipHash string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE ip_hash=? AND created_at >= datetime('now', '-10 minutes')`, ipHash).Scan(&n)
	return n, err
}

func (r *CommentRepository) MarkIPSpam(ctx context.Context, siteID int64, ipHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE comments SET status='spam', updated_at=? WHERE site_id=? AND ip_hash=? AND status IN ('pending','approved')`, nowString(), siteID, ipHash)
	return err
}

func (r *CommentRepository) list(ctx context.Context, where string, args ...any) ([]*domain.Comment, error) {
	query := commentSelectSQL + ` ` + where
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var comments []*domain.Comment
	for rows.Next() {
		c, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func scanComment(scanner interface{ Scan(...any) error }) (*domain.Comment, error) {
	var c domain.Comment
	var parent, root, displayName, tripcodePublic, email, avatar, website, ipHash, uaHash, metadata, moderationReason, edited, badgeType, badgeLabel sql.NullString
	var identityID sql.NullInt64
	var created, updated string
	if err := scanner.Scan(&c.ID, &c.SiteID, &c.PageID, &parent, &root, &c.Depth, &c.Path, &c.AuthorName, &displayName, &identityID, &tripcodePublic, &c.TripcodeKind, &badgeType, &badgeLabel, &email, &avatar, &website, &c.BodyMarkdown, &c.BodyHTML, &c.Status, &ipHash, &uaHash, &metadata, &moderationReason, &created, &updated, &edited); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.ParentID = nullableString(parent)
	c.RootID = nullableString(root)
	c.AuthorDisplayName = c.AuthorName
	if displayName.Valid && displayName.String != "" {
		c.AuthorDisplayName = displayName.String
	}
	c.IdentityID = nullableInt64(identityID)
	c.TripcodePublic = nullableString(tripcodePublic)
	if c.TripcodeKind == "" {
		c.TripcodeKind = domain.TripcodeNone
	}
	if badgeType.Valid {
		bt := domain.BadgeType(badgeType.String)
		c.BadgeType = &bt
	}
	c.BadgeLabel = nullableString(badgeLabel)
	c.AuthorEmailHash = nullableString(email)
	c.AuthorAvatarHash = nullableString(avatar)
	c.AuthorWebsite = nullableString(website)
	c.IPHash = nullableString(ipHash)
	c.UserAgentHash = nullableString(uaHash)
	c.MetadataJSON = nullableString(metadata)
	c.ModerationReason = nullableString(moderationReason)
	c.CreatedAt = parseTime(created)
	c.UpdatedAt = parseTime(updated)
	c.EditedAt = nullableTime(edited)
	return &c, nil
}

const commentSelectSQL = `
	SELECT comments.id, comments.site_id, comments.page_id, comments.parent_id, comments.root_id, comments.depth, comments.path,
		comments.author_name,
		COALESCE(comments.author_display_name, comments.author_name) AS author_display_name,
		comments.identity_id,
		comments.tripcode_public,
		comments.tripcode_kind,
		identities.badge_type,
		identities.badge_label,
		comments.author_email_hash,
		comments.author_avatar_hash,
		comments.author_website,
		comments.body_markdown,
		comments.body_html,
		comments.status,
		comments.ip_hash,
		comments.user_agent_hash,
		comments.metadata_json,
		(
			SELECT moderation_events.reason
			FROM moderation_events
			WHERE moderation_events.comment_id = comments.id
				AND moderation_events.reason IS NOT NULL
				AND moderation_events.reason != ''
			ORDER BY moderation_events.created_at DESC, moderation_events.id DESC
			LIMIT 1
		) AS moderation_reason,
		comments.created_at,
		comments.updated_at,
		comments.edited_at
	FROM comments
	LEFT JOIN identities ON identities.id = comments.identity_id
	LEFT JOIN comments root_comments ON root_comments.id = COALESCE(comments.root_id, comments.id)`

func CommentPath(parent *domain.Comment, id string) string {
	if parent == nil || parent.Path == "" {
		return fmt.Sprintf("%s", id)
	}
	return parent.Path + "/" + id
}
