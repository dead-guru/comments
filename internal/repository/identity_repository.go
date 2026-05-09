package repository

import (
	"context"
	"database/sql"

	"deadcomments/internal/domain"
)

type IdentityRepository struct {
	db *sql.DB
}

func NewIdentityRepository(db *sql.DB) *IdentityRepository {
	return &IdentityRepository{db: db}
}

func (r *IdentityRepository) Create(ctx context.Context, identity *domain.Identity) error {
	now := nowString()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO identities(site_id, display_name, normalized_name, type, secret_hash, public_tripcode, badge_type, badge_label, created_by_admin_id, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		identity.SiteID,
		identity.DisplayName,
		identity.NormalizedName,
		identity.Type,
		identity.SecretHash,
		identity.PublicTripcode,
		identity.BadgeType,
		identity.BadgeLabel,
		identity.CreatedByAdminID,
		now,
		now)
	if err != nil {
		return err
	}
	identity.ID, _ = res.LastInsertId()
	identity.CreatedAt = parseTime(now)
	identity.UpdatedAt = parseTime(now)
	return nil
}

func (r *IdentityRepository) Update(ctx context.Context, identity *domain.Identity) error {
	now := nowString()
	_, err := r.db.ExecContext(ctx, `
		UPDATE identities
		SET site_id=?, display_name=?, normalized_name=?, public_tripcode=?, badge_type=?, badge_label=?, updated_at=?
		WHERE id=?`,
		identity.SiteID,
		identity.DisplayName,
		identity.NormalizedName,
		identity.PublicTripcode,
		identity.BadgeType,
		identity.BadgeLabel,
		now,
		identity.ID)
	if err == nil {
		identity.UpdatedAt = parseTime(now)
	}
	return err
}

func (r *IdentityRepository) ResetSecret(ctx context.Context, id int64, secretHash, publicTripcode string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE identities SET secret_hash=?, public_tripcode=?, updated_at=? WHERE id=?`, secretHash, publicTripcode, nowString(), id)
	return err
}

func (r *IdentityRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM identities WHERE id=?`, id)
	return err
}

func (r *IdentityRepository) ByID(ctx context.Context, id int64) (*domain.Identity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, site_id, display_name, normalized_name, type, secret_hash, public_tripcode, badge_type, badge_label, created_by_admin_id, created_at, updated_at
		FROM identities
		WHERE id=?`, id)
	return scanIdentity(row)
}

func (r *IdentityRepository) ByNormalizedName(ctx context.Context, siteID int64, normalizedName string) (*domain.Identity, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, site_id, display_name, normalized_name, type, secret_hash, public_tripcode, badge_type, badge_label, created_by_admin_id, created_at, updated_at
		FROM identities
		WHERE normalized_name=? AND (site_id=? OR site_id IS NULL)
		ORDER BY CASE WHEN site_id=? THEN 0 ELSE 1 END
		LIMIT 1`, normalizedName, siteID, siteID)
	return scanIdentity(row)
}

func (r *IdentityRepository) List(ctx context.Context) ([]*domain.Identity, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, site_id, display_name, normalized_name, type, secret_hash, public_tripcode, badge_type, badge_label, created_by_admin_id, created_at, updated_at
		FROM identities
		ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var identities []*domain.Identity
	for rows.Next() {
		identity, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		identities = append(identities, identity)
	}
	return identities, rows.Err()
}

func scanIdentity(scanner interface{ Scan(...any) error }) (*domain.Identity, error) {
	var identity domain.Identity
	var siteID, adminID sql.NullInt64
	var badgeLabel sql.NullString
	var created, updated string
	if err := scanner.Scan(
		&identity.ID,
		&siteID,
		&identity.DisplayName,
		&identity.NormalizedName,
		&identity.Type,
		&identity.SecretHash,
		&identity.PublicTripcode,
		&identity.BadgeType,
		&badgeLabel,
		&adminID,
		&created,
		&updated,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	identity.SiteID = nullableInt64(siteID)
	identity.BadgeLabel = nullableString(badgeLabel)
	identity.CreatedByAdminID = nullableInt64(adminID)
	identity.CreatedAt = parseTime(created)
	identity.UpdatedAt = parseTime(updated)
	return &identity, nil
}
