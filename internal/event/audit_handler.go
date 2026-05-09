package event

import (
	"context"
	"fmt"

	"deadcomments/internal/domain"
	"deadcomments/internal/repository"
)

type AuditHandler struct {
	moderation *repository.ModerationRepository
}

func NewAuditHandler(moderation *repository.ModerationRepository) *AuditHandler {
	return &AuditHandler{moderation: moderation}
}

func (h *AuditHandler) Key() string {
	return "audit.moderation_events"
}

func (h *AuditHandler) Handle(ctx context.Context, event domain.Event) error {
	if event.CommentID == nil {
		return nil
	}
	action, ok := auditAction(event)
	if !ok {
		return nil
	}
	reason := payloadString(event.Payload, "reason")
	moderationEvent := &domain.ModerationEvent{
		CommentID: *event.CommentID,
		AdminID:   event.ActorAdminID,
		Action:    action,
	}
	if reason != "" {
		moderationEvent.Reason = &reason
	}
	return h.moderation.CreateEvent(ctx, moderationEvent)
}

func auditAction(event domain.Event) (string, bool) {
	switch event.Type {
	case domain.EventCommentCreated:
		return "created", true
	case domain.EventCommentStatusSet:
		return payloadString(event.Payload, "status"), true
	case domain.EventCommentEdited:
		return "edit", true
	case domain.EventCommentIPBanned:
		return "ban_ip", true
	default:
		return "", false
	}
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
