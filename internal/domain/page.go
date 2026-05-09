package domain

import "time"

type PageState string

const (
	PageOpen     PageState = "open"
	PageLocked   PageState = "locked"
	PageHidden   PageState = "hidden"
	PageArchived PageState = "archived"
)

type Page struct {
	ID            int64
	SiteID        int64
	PageKey       string
	Title         string
	URL           string
	State         PageState
	CommentsCount int
	ApprovedCount int
	PendingCount  int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (p Page) CanPost() bool {
	return p.State == PageOpen
}
