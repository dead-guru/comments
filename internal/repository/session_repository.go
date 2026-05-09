package repository

import (
	"context"
	"database/sql"

	"deadcomments/internal/domain"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.AdminSession) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `INSERT INTO admin_sessions(admin_id, token_hash, expires_at, created_at) VALUES(?,?,?,?)`, s.AdminID, s.TokenHash, s.ExpiresAt.UTC().Format(timeFormat), now)
	if err != nil {
		return err
	}
	s.ID, _ = res.LastInsertId()
	s.CreatedAt = parseTime(now)
	return nil
}

func (r *SessionRepository) ByTokenHash(ctx context.Context, hash string) (*domain.AdminSession, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, admin_id, token_hash, expires_at, created_at FROM admin_sessions WHERE token_hash=? AND expires_at > ?`, hash, nowString())
	var s domain.AdminSession
	var expires, created string
	if err := row.Scan(&s.ID, &s.AdminID, &s.TokenHash, &expires, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.ExpiresAt = parseTime(expires)
	s.CreatedAt = parseTime(created)
	return &s, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE token_hash=?`, hash)
	return err
}

func (r *SessionRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE expires_at <= ?`, nowString())
	return err
}

const timeFormat = "2006-01-02T15:04:05.999999999Z07:00"
