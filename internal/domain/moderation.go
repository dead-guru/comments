package domain

import "time"

type WordBanAction string

const (
	WordBanPending WordBanAction = "pending"
	WordBanReject  WordBanAction = "reject"
	WordBanSpam    WordBanAction = "spam"
)

type IPBan struct {
	ID               int64
	SiteID           *int64
	IPHash           string
	Reason           *string
	CreatedByAdminID *int64
	CreatedAt        time.Time
}

type WordBan struct {
	ID        int64
	SiteID    *int64
	Pattern   string
	Action    WordBanAction
	CreatedAt time.Time
}

type ModerationEvent struct {
	ID        int64
	CommentID string
	AdminID   *int64
	Action    string
	Reason    *string
	CreatedAt time.Time
}

type ModerationDecision struct {
	Status CommentStatus
	Reason string
}
