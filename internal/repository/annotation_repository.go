package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"

	"deadcomments/internal/domain"
)

type AnnotationRepository struct {
	db *sql.DB
}

func NewAnnotationRepository(db *sql.DB) *AnnotationRepository {
	return &AnnotationRepository{db: db}
}

func (r *AnnotationRepository) Create(ctx context.Context, a *domain.Annotation) error {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO annotations(id, site_id, page_id, comment_id, selector, selected_text, selection_prefix, selection_suffix, text_start, text_end, text_hash, metadata_json, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.SiteID, a.PageID, a.CommentID, a.Selector, a.SelectedText, a.SelectionPrefix, a.SelectionSuffix, a.TextStart, a.TextEnd, a.TextHash, a.MetadataJSON, now, now)
	if err != nil {
		return err
	}
	a.CreatedAt = parseTime(now)
	a.UpdatedAt = parseTime(now)
	return nil
}

func (r *AnnotationRepository) ApprovedByPage(ctx context.Context, pageID int64) ([]*domain.Annotation, error) {
	rows, err := r.db.QueryContext(ctx, annotationSelectSQL+` WHERE annotations.page_id=? AND comments.status='approved' ORDER BY annotations.created_at`, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Annotation
	for rows.Next() {
		a, err := scanAnnotation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func AnnotationTextHash(value string) string {
	normalized := strings.Join(strings.Fields(value), " ")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

const annotationSelectSQL = `
SELECT
	annotations.id,
	annotations.site_id,
	sites.key,
	annotations.page_id,
	pages.page_key,
	annotations.comment_id,
	annotations.selector,
	annotations.selected_text,
	annotations.selection_prefix,
	annotations.selection_suffix,
	annotations.text_start,
	annotations.text_end,
	annotations.text_hash,
	annotations.metadata_json,
	annotations.created_at,
	annotations.updated_at,
	comments.id,
	comments.site_id,
	comment_sites.key,
	comments.page_id,
	comment_pages.page_key,
	COALESCE(comment_pages.title, ''),
	COALESCE(comment_pages.url, ''),
	comments.parent_id,
	comments.root_id,
	comments.depth,
	comments.path,
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
FROM annotations
JOIN sites ON sites.id=annotations.site_id
JOIN pages ON pages.id=annotations.page_id
JOIN comments ON comments.id=annotations.comment_id
JOIN sites comment_sites ON comment_sites.id=comments.site_id
JOIN pages comment_pages ON comment_pages.id=comments.page_id
LEFT JOIN identities ON identities.id=comments.identity_id`

func scanAnnotation(scanner interface{ Scan(...any) error }) (*domain.Annotation, error) {
	var a domain.Annotation
	var textStart, textEnd sql.NullInt64
	var metadata sql.NullString
	var created, updated string
	var c domain.Comment
	var parent, root, siteKey, pageKey, pageTitle, pageURL, displayName, tripcodePublic, email, avatar, website, ipHash, uaHash, commentMetadata, moderationReason, edited, badgeType, badgeLabel sql.NullString
	var identityID sql.NullInt64
	var commentCreated, commentUpdated string
	args := []any{
		&a.ID,
		&a.SiteID,
		&a.SiteKey,
		&a.PageID,
		&a.PageKey,
		&a.CommentID,
		&a.Selector,
		&a.SelectedText,
		&a.SelectionPrefix,
		&a.SelectionSuffix,
		&textStart,
		&textEnd,
		&a.TextHash,
		&metadata,
		&created,
		&updated,
		&c.ID, &c.SiteID, &siteKey, &c.PageID, &pageKey, &pageTitle, &pageURL, &parent, &root, &c.Depth, &c.Path, &c.AuthorName, &displayName, &identityID, &tripcodePublic, &c.TripcodeKind, &badgeType, &badgeLabel, &email, &avatar, &website, &c.BodyMarkdown, &c.BodyHTML, &c.Status, &ipHash, &uaHash, &commentMetadata, &moderationReason, &commentCreated, &commentUpdated, &edited,
	}
	if err := scanner.Scan(args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	a.TextStart = nullableInt64(textStart)
	a.TextEnd = nullableInt64(textEnd)
	a.MetadataJSON = nullableString(metadata)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	c.SiteKey = siteKey.String
	c.PageKey = pageKey.String
	c.PageTitle = pageTitle.String
	c.PageURL = pageURL.String
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
	c.MetadataJSON = nullableString(commentMetadata)
	c.ModerationReason = nullableString(moderationReason)
	c.CreatedAt = parseTime(commentCreated)
	c.UpdatedAt = parseTime(commentUpdated)
	c.EditedAt = nullableTime(edited)
	a.Comment = &c
	return &a, nil
}
