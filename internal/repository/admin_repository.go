package repository

import (
	"context"
	"database/sql"

	"deadcomments/internal/domain"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) UpsertGitHub(ctx context.Context, a *domain.Admin) error {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO admins(github_id, github_login, email, name, avatar_url, role, created_at, updated_at, last_login_at)
		VALUES(?,?,?,?,?,?,?,?,?)
		ON CONFLICT(github_id) DO UPDATE SET github_login=excluded.github_login, email=excluded.email, name=excluded.name, avatar_url=excluded.avatar_url, updated_at=excluded.updated_at, last_login_at=excluded.last_login_at`,
		a.GitHubID, a.GitHubLogin, a.Email, a.Name, a.AvatarURL, a.Role, now, now, now)
	if err != nil {
		return err
	}
	stored, err := r.ByGitHubID(ctx, a.GitHubID)
	if err != nil {
		return err
	}
	*a = *stored
	return nil
}

func (r *AdminRepository) ByGitHubID(ctx context.Context, id int64) (*domain.Admin, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, github_id, github_login, email, name, avatar_url, role, created_at, updated_at, last_login_at FROM admins WHERE github_id=?`, id)
	return scanAdmin(row)
}

func (r *AdminRepository) ByID(ctx context.Context, id int64) (*domain.Admin, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, github_id, github_login, email, name, avatar_url, role, created_at, updated_at, last_login_at FROM admins WHERE id=?`, id)
	return scanAdmin(row)
}

func scanAdmin(scanner interface{ Scan(...any) error }) (*domain.Admin, error) {
	var a domain.Admin
	var email, name, avatar, last sql.NullString
	var created, updated string
	if err := scanner.Scan(&a.ID, &a.GitHubID, &a.GitHubLogin, &email, &name, &avatar, &a.Role, &created, &updated, &last); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	a.Email = nullableString(email)
	a.Name = nullableString(name)
	a.AvatarURL = nullableString(avatar)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	a.LastLoginAt = nullableTime(last)
	return &a, nil
}
