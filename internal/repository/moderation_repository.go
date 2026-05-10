package repository

import (
	"context"
	"database/sql"

	"deadcomments/internal/domain"
)

type ModerationRepository struct {
	db *sql.DB
}

func NewModerationRepository(db *sql.DB) *ModerationRepository {
	return &ModerationRepository{db: db}
}

func (r *ModerationRepository) IsIPBanned(ctx context.Context, siteID int64, ipHash string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ip_bans WHERE ip_hash=? AND (site_id IS NULL OR site_id=?)`, ipHash, siteID).Scan(&n)
	return n > 0, err
}

func (r *ModerationRepository) CreateIPBan(ctx context.Context, ban *domain.IPBan) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `INSERT INTO ip_bans(site_id, ip_hash, reason, created_by_admin_id, created_at) VALUES(?,?,?,?,?)`, ban.SiteID, ban.IPHash, ban.Reason, ban.CreatedByAdminID, now)
	if err != nil {
		return err
	}
	ban.ID, _ = res.LastInsertId()
	ban.CreatedAt = parseTime(now)
	return nil
}

func (r *ModerationRepository) ListIPBans(ctx context.Context) ([]*domain.IPBan, error) {
	return r.ListIPBansPaginated(ctx, 0, 0)
}

func (r *ModerationRepository) ListIPBansPaginated(ctx context.Context, limit, offset int) ([]*domain.IPBan, error) {
	query := `SELECT id, site_id, ip_hash, reason, created_by_admin_id, created_at FROM ip_bans ORDER BY created_at DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, nonNegative(offset))
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.IPBan
	for rows.Next() {
		var b domain.IPBan
		var siteID, adminID sql.NullInt64
		var reason sql.NullString
		var created string
		if err := rows.Scan(&b.ID, &siteID, &b.IPHash, &reason, &adminID, &created); err != nil {
			return nil, err
		}
		b.SiteID = nullableInt64(siteID)
		b.Reason = nullableString(reason)
		b.CreatedByAdminID = nullableInt64(adminID)
		b.CreatedAt = parseTime(created)
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *ModerationRepository) DeleteBan(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM ip_bans WHERE id=?`, id)
	if err == nil {
		return nil
	}
	return err
}

func (r *ModerationRepository) CreateWordBan(ctx context.Context, ban *domain.WordBan) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `INSERT INTO word_bans(site_id, pattern, action, created_at) VALUES(?,?,?,?)`, ban.SiteID, ban.Pattern, ban.Action, now)
	if err != nil {
		return err
	}
	ban.ID, _ = res.LastInsertId()
	ban.CreatedAt = parseTime(now)
	return nil
}

func (r *ModerationRepository) WordBans(ctx context.Context, siteID int64) ([]*domain.WordBan, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, site_id, pattern, action, created_at FROM word_bans WHERE site_id IS NULL OR site_id=? ORDER BY id`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WordBan
	for rows.Next() {
		var b domain.WordBan
		var site sql.NullInt64
		var created string
		if err := rows.Scan(&b.ID, &site, &b.Pattern, &b.Action, &created); err != nil {
			return nil, err
		}
		b.SiteID = nullableInt64(site)
		b.CreatedAt = parseTime(created)
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *ModerationRepository) ListWordBans(ctx context.Context) ([]*domain.WordBan, error) {
	return r.ListWordBansPaginated(ctx, 0, 0)
}

func (r *ModerationRepository) ListWordBansPaginated(ctx context.Context, limit, offset int) ([]*domain.WordBan, error) {
	query := `SELECT id, site_id, pattern, action, created_at FROM word_bans ORDER BY created_at DESC`
	args := []any{}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, nonNegative(offset))
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WordBan
	for rows.Next() {
		var b domain.WordBan
		var site sql.NullInt64
		var created string
		if err := rows.Scan(&b.ID, &site, &b.Pattern, &b.Action, &created); err != nil {
			return nil, err
		}
		b.SiteID = nullableInt64(site)
		b.CreatedAt = parseTime(created)
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *ModerationRepository) DeleteWordBan(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM word_bans WHERE id=?`, id)
	return err
}

func (r *ModerationRepository) CreateEvent(ctx context.Context, event *domain.ModerationEvent) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `INSERT INTO moderation_events(comment_id, admin_id, action, reason, created_at) VALUES(?,?,?,?,?)`, event.CommentID, event.AdminID, event.Action, event.Reason, now)
	if err != nil {
		return err
	}
	event.ID, _ = res.LastInsertId()
	event.CreatedAt = parseTime(now)
	return nil
}

func (r *ModerationRepository) EventsForComment(ctx context.Context, commentID string) ([]*domain.ModerationEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, comment_id, admin_id, action, reason, created_at FROM moderation_events WHERE comment_id=? ORDER BY created_at DESC`, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ModerationEvent
	for rows.Next() {
		var e domain.ModerationEvent
		var adminID sql.NullInt64
		var reason sql.NullString
		var created string
		if err := rows.Scan(&e.ID, &e.CommentID, &adminID, &e.Action, &reason, &created); err != nil {
			return nil, err
		}
		e.AdminID = nullableInt64(adminID)
		e.Reason = nullableString(reason)
		e.CreatedAt = parseTime(created)
		out = append(out, &e)
	}
	return out, rows.Err()
}
