package observability

import (
	"context"
	"fmt"

	"deadcomments/internal/domain"
)

type EventMetricsHandler struct {
	metrics *Metrics
}

func NewEventMetricsHandler(metrics *Metrics) *EventMetricsHandler {
	return &EventMetricsHandler{metrics: metrics}
}

func (h *EventMetricsHandler) Key() string {
	return "prometheus-metrics"
}

func (h *EventMetricsHandler) Handle(_ context.Context, evt domain.Event) error {
	if h == nil || h.metrics == nil {
		return nil
	}
	h.metrics.recordDomainEvent(evt)
	return nil
}

func (m *Metrics) recordDomainEvent(evt domain.Event) {
	eventType := cleanLabel(string(evt.Type), "unknown")
	aggregateType := cleanLabel(evt.AggregateType, "unknown")
	m.domainEvents.WithLabelValues(eventType, aggregateType).Inc()

	switch evt.Type {
	case domain.EventCommentCreated:
		m.recordCommentCreated(evt)
	case domain.EventCommentStatusSet:
		m.recordCommentStatusSet(evt)
	case domain.EventPageAutoCreated:
		m.pageEvents.WithLabelValues("auto_created", pageStateFromPayload(evt.Payload, "state")).Inc()
	case domain.EventPageStateChanged:
		m.pageEvents.WithLabelValues("state_changed", pageStateFromPayload(evt.Payload, "new_state")).Inc()
	case domain.EventSiteCreated:
		m.siteEvents.WithLabelValues("created").Inc()
	case domain.EventSiteUpdated:
		m.siteEvents.WithLabelValues("updated").Inc()
	case domain.EventIPBanCreated:
		m.banEvents.WithLabelValues("ip", "created").Inc()
	case domain.EventIPBanDeleted:
		m.banEvents.WithLabelValues("ip", "deleted").Inc()
	case domain.EventWordBanCreated:
		m.banEvents.WithLabelValues("word", "created").Inc()
	case domain.EventWordBanDeleted:
		m.banEvents.WithLabelValues("word", "deleted").Inc()
	case domain.EventAdminLoggedIn:
		m.adminLogins.Inc()
	case domain.EventIdentityCreated:
		m.identityEvents.WithLabelValues("created", badgeTypeFromPayload(evt.Payload)).Inc()
	case domain.EventIdentityUpdated:
		m.identityEvents.WithLabelValues("updated", badgeTypeFromPayload(evt.Payload)).Inc()
	case domain.EventIdentityDeleted:
		m.identityEvents.WithLabelValues("deleted", "unknown").Inc()
	case domain.EventIdentitySecretReset:
		m.identityEvents.WithLabelValues("secret_reset", "unknown").Inc()
	}
}

func (m *Metrics) recordCommentCreated(evt domain.Event) {
	status := commentStatusFromPayload(evt.Payload, "status")
	reason := moderationReasonFromPayload(evt.Payload)
	tripcodeKind := tripcodeKindFromPayload(evt.Payload)
	hasParent := "false"
	if evt.Payload["parent_id"] != nil {
		hasParent = "true"
	}
	m.commentsCreated.WithLabelValues(status, reason, tripcodeKind, hasParent).Inc()
}

func (m *Metrics) recordCommentStatusSet(evt domain.Event) {
	fromStatus := commentStatusFromPayload(evt.Payload, "old_status")
	toStatus := commentStatusFromPayload(evt.Payload, "status")
	action := toStatus
	if action == "unknown" {
		action = "status_set"
	}
	m.commentModeration.WithLabelValues(action, fromStatus, toStatus).Inc()
}

func commentStatusFromPayload(payload map[string]any, key string) string {
	return allowLabel(payloadString(payload, key), map[string]struct{}{
		string(domain.CommentPending):  {},
		string(domain.CommentApproved): {},
		string(domain.CommentRejected): {},
		string(domain.CommentSpam):     {},
		string(domain.CommentDeleted):  {},
	}, "unknown")
}

func moderationReasonFromPayload(payload map[string]any) string {
	return allowLabel(payloadString(payload, "reason"), map[string]struct{}{
		"auto moderation":   {},
		"duplicate body":    {},
		"honeypot":          {},
		"ip banned":         {},
		"manual moderation": {},
		"rate limit":        {},
		"too many links":    {},
		"word ban":          {},
	}, "none")
}

func tripcodeKindFromPayload(payload map[string]any) string {
	return allowLabel(payloadString(payload, "tripcode_kind"), map[string]struct{}{
		string(domain.TripcodeNone):      {},
		string(domain.TripcodeAnonymous): {},
		string(domain.TripcodeReserved):  {},
	}, "unknown")
}

func pageStateFromPayload(payload map[string]any, key string) string {
	return allowLabel(payloadString(payload, key), map[string]struct{}{
		string(domain.PageOpen):     {},
		string(domain.PageLocked):   {},
		string(domain.PageHidden):   {},
		string(domain.PageArchived): {},
	}, "unknown")
}

func badgeTypeFromPayload(payload map[string]any) string {
	return allowLabel(payloadString(payload, "badge_type"), map[string]struct{}{
		string(domain.BadgeVerified): {},
		string(domain.BadgeAdmin):    {},
		string(domain.BadgeAuthor):   {},
		string(domain.BadgeCustom):   {},
	}, "unknown")
}

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}

func allowLabel(value string, allowed map[string]struct{}, fallback string) string {
	value = cleanLabel(value, fallback)
	if _, ok := allowed[value]; ok {
		return value
	}
	return fallback
}
