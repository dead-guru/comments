package domain

import "time"

type EventType string

const (
	EventSiteCreated         EventType = "site.created"
	EventSiteUpdated         EventType = "site.updated"
	EventPageAutoCreated     EventType = "page.auto_created"
	EventPageStateChanged    EventType = "page.state_changed"
	EventCommentCreated      EventType = "comment.created"
	EventCommentStatusSet    EventType = "comment.status_set"
	EventCommentEdited       EventType = "comment.edited"
	EventCommentIPBanned     EventType = "comment.ip_banned"
	EventAnnotationCreated   EventType = "annotation.created"
	EventIPBanCreated        EventType = "ip_ban.created"
	EventIPBanDeleted        EventType = "ip_ban.deleted"
	EventWordBanCreated      EventType = "word_ban.created"
	EventWordBanDeleted      EventType = "word_ban.deleted"
	EventAdminLoggedIn       EventType = "admin.logged_in"
	EventIdentityCreated     EventType = "identity.created"
	EventIdentityUpdated     EventType = "identity.updated"
	EventIdentityDeleted     EventType = "identity.deleted"
	EventIdentitySecretReset EventType = "identity.secret_reset"
)

type Event struct {
	ID            string
	Type          EventType
	ActorAdminID  *int64
	SiteID        *int64
	PageID        *int64
	CommentID     *string
	AggregateType string
	AggregateID   string
	Payload       map[string]any
	OccurredAt    time.Time
}

type EventDelivery struct {
	ID          int64
	EventID     string
	HandlerKey  string
	Status      string
	Attempts    int
	LastError   *string
	DeliveredAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
