package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"deadcomments/internal/domain"
)

type SiteRepository struct {
	db *sql.DB
}

func NewSiteRepository(db *sql.DB) *SiteRepository {
	return &SiteRepository{db: db}
}

func (r *SiteRepository) Create(ctx context.Context, s *domain.Site) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO sites(key, name, allowed_origins_json, default_moderation_mode, default_page_state, default_theme, max_comment_length, allow_replies, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		s.Key, s.Name, originsJSON(s.AllowedOrigins), s.DefaultModerationMode, s.DefaultPageState, s.DefaultTheme, s.MaxCommentLength, boolInt(s.AllowReplies), now, now)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	s.CreatedAt = parseTime(now)
	s.UpdatedAt = parseTime(now)
	return nil
}

func (r *SiteRepository) Update(ctx context.Context, s *domain.Site) error {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
		UPDATE sites SET key=?, name=?, allowed_origins_json=?, default_moderation_mode=?, default_page_state=?, default_theme=?, max_comment_length=?, allow_replies=?, updated_at=?
		WHERE id=?`,
		s.Key, s.Name, originsJSON(s.AllowedOrigins), s.DefaultModerationMode, s.DefaultPageState, s.DefaultTheme, s.MaxCommentLength, boolInt(s.AllowReplies), now, s.ID)
	if err == nil {
		s.UpdatedAt = parseTime(now)
	}
	return err
}

func (r *SiteRepository) ByKey(ctx context.Context, key string) (*domain.Site, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, key, name, allowed_origins_json, default_moderation_mode, default_page_state, default_theme, max_comment_length, allow_replies, created_at, updated_at FROM sites WHERE key=?`, key)
	return scanSite(row)
}

func (r *SiteRepository) ByID(ctx context.Context, id int64) (*domain.Site, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, key, name, allowed_origins_json, default_moderation_mode, default_page_state, default_theme, max_comment_length, allow_replies, created_at, updated_at FROM sites WHERE id=?`, id)
	return scanSite(row)
}

func (r *SiteRepository) List(ctx context.Context) ([]*domain.Site, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, key, name, allowed_origins_json, default_moderation_mode, default_page_state, default_theme, max_comment_length, allow_replies, created_at, updated_at FROM sites ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sites []*domain.Site
	for rows.Next() {
		s, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}

func (r *SiteRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sites`).Scan(&n)
	return n, err
}

func scanSite(scanner interface{ Scan(...any) error }) (*domain.Site, error) {
	var s domain.Site
	var origins, created, updated string
	var replies int
	if err := scanner.Scan(&s.ID, &s.Key, &s.Name, &origins, &s.DefaultModerationMode, &s.DefaultPageState, &s.DefaultTheme, &s.MaxCommentLength, &replies, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	s.AllowedOrigins = parseOrigins(origins)
	s.AllowReplies = replies == 1
	s.CreatedAt = parseTime(created)
	s.UpdatedAt = parseTime(updated)
	return &s, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func NormalizeSiteKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, " ", "-")
	return key
}
