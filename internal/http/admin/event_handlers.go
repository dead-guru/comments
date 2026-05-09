package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"deadcomments/internal/repository"
)

func (h *Handlers) Events(w http.ResponseWriter, r *http.Request) {
	filter := eventFilterFromRequest(r)
	events, _ := h.events.List(r.Context(), filter)
	h.render(w, r, "admin/events.html", map[string]any{"Events": events, "Filters": r.URL.Query()})
}

func eventFilterFromRequest(r *http.Request) repository.EventFilter {
	q := r.URL.Query()
	filter := repository.EventFilter{
		Type:          strings.TrimSpace(q.Get("type")),
		AggregateType: strings.TrimSpace(q.Get("aggregate_type")),
		AggregateID:   strings.TrimSpace(q.Get("aggregate_id")),
		Limit:         200,
	}
	if raw := strings.TrimSpace(q.Get("actor_admin_id")); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.ActorAdminID = &id
		}
	}
	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		filter.From = normalizeEventDate(raw, false)
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		filter.To = normalizeEventDate(raw, true)
	}
	return filter
}

func normalizeEventDate(raw string, endOfDay bool) string {
	date, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	if endOfDay {
		date = date.Add(24*time.Hour - time.Nanosecond)
	}
	return date.UTC().Format(time.RFC3339Nano)
}
