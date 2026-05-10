package repository

import (
	"context"
	"database/sql"
	"errors"

	"deadcomments/internal/domain"
)

type PageRepository struct {
	db *sql.DB
}

func NewPageRepository(db *sql.DB) *PageRepository {
	return &PageRepository{db: db}
}

func (r *PageRepository) FindOrCreate(ctx context.Context, site *domain.Site, pageKey, title, url string) (*domain.Page, bool, error) {
	page, err := r.BySiteAndKey(ctx, site.ID, pageKey)
	if err != nil || page != nil {
		return page, false, err
	}
	now := nowString()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO pages(site_id, page_key, title, url, state, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?)`, site.ID, pageKey, title, url, site.DefaultPageState, now, now)
	if err != nil {
		existing, findErr := r.BySiteAndKey(ctx, site.ID, pageKey)
		if findErr == nil && existing != nil {
			return existing, false, nil
		}
		return nil, false, err
	}
	id, _ := res.LastInsertId()
	page, err = r.ByID(ctx, id)
	return page, true, err
}

func (r *PageRepository) BySiteAndKey(ctx context.Context, siteID int64, pageKey string) (*domain.Page, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, site_id, page_key, title, url, state, comments_count, approved_count, pending_count, created_at, updated_at FROM pages WHERE site_id=? AND page_key=?`, siteID, pageKey)
	return scanPage(row)
}

func (r *PageRepository) ByID(ctx context.Context, id int64) (*domain.Page, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, site_id, page_key, title, url, state, comments_count, approved_count, pending_count, created_at, updated_at FROM pages WHERE id=?`, id)
	return scanPage(row)
}

func (r *PageRepository) List(ctx context.Context, siteID *int64, state, search string) ([]*domain.Page, error) {
	return r.ListPaginated(ctx, siteID, state, search, 200, 0)
}

func (r *PageRepository) ListPaginated(ctx context.Context, siteID *int64, state, search string, limit, offset int) ([]*domain.Page, error) {
	q := `SELECT id, site_id, page_key, title, url, state, comments_count, approved_count, pending_count, created_at, updated_at FROM pages WHERE 1=1`
	args := []any{}
	if siteID != nil {
		q += ` AND site_id=?`
		args = append(args, *siteID)
	}
	if state != "" {
		q += ` AND state=?`
		args = append(args, state)
	}
	if search != "" {
		q += ` AND (page_key LIKE ? OR title LIKE ? OR url LIKE ?)`
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if limit <= 0 {
		limit = 200
	}
	q += ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, nonNegative(offset))
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []*domain.Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *PageRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pages`).Scan(&n)
	return n, err
}

func (r *PageRepository) SetState(ctx context.Context, id int64, state domain.PageState) error {
	_, err := r.db.ExecContext(ctx, `UPDATE pages SET state=?, updated_at=? WHERE id=?`, state, nowString(), id)
	return err
}

func (r *PageRepository) Recount(ctx context.Context, pageID int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE pages SET
			comments_count = (SELECT COUNT(*) FROM comments WHERE page_id=? AND status != 'deleted'),
			approved_count = (SELECT COUNT(*) FROM comments WHERE page_id=? AND status = 'approved'),
			pending_count = (SELECT COUNT(*) FROM comments WHERE page_id=? AND status = 'pending'),
			updated_at = ?
		WHERE id=?`, pageID, pageID, pageID, nowString(), pageID)
	return err
}

func scanPage(scanner interface{ Scan(...any) error }) (*domain.Page, error) {
	var p domain.Page
	var created, updated string
	if err := scanner.Scan(&p.ID, &p.SiteID, &p.PageKey, &p.Title, &p.URL, &p.State, &p.CommentsCount, &p.ApprovedCount, &p.PendingCount, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.CreatedAt = parseTime(created)
	p.UpdatedAt = parseTime(updated)
	return &p, nil
}
