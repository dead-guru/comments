package domain

import "time"

type AdminRole string

const (
	RoleOwner     AdminRole = "owner"
	RoleAdmin     AdminRole = "admin"
	RoleModerator AdminRole = "moderator"
)

type Admin struct {
	ID          int64
	GitHubID    int64
	GitHubLogin string
	Email       *string
	Name        *string
	AvatarURL   *string
	Role        AdminRole
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

type AdminSession struct {
	ID        int64
	AdminID   int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
