package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"deadcomments/internal/domain"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Store(ctx context.Context, event domain.Event) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO events(id, type, actor_admin_id, site_id, page_id, comment_id, aggregate_type, aggregate_id, payload_json, occurred_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`,
		event.ID,
		event.Type,
		event.ActorAdminID,
		event.SiteID,
		event.PageID,
		event.CommentID,
		event.AggregateType,
		event.AggregateID,
		string(payload),
		event.OccurredAt.UTC().Format(timeFormat),
	)
	return err
}

func (r *EventRepository) MarkDelivery(ctx context.Context, eventID, handlerKey string, err error) error {
	now := nowString()
	status := "delivered"
	var lastError any
	var deliveredAt any = now
	if err != nil {
		status = "failed"
		lastError = err.Error()
		deliveredAt = nil
	}
	_, execErr := r.db.ExecContext(ctx, `
		INSERT INTO event_deliveries(event_id, handler_key, status, attempts, last_error, delivered_at, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(event_id, handler_key) DO UPDATE SET
			status=excluded.status,
			attempts=event_deliveries.attempts + 1,
			last_error=excluded.last_error,
			delivered_at=excluded.delivered_at,
			updated_at=excluded.updated_at`,
		eventID, handlerKey, status, 1, lastError, deliveredAt, now, now)
	return execErr
}

func (r *EventRepository) List(ctx context.Context, limit int) ([]domain.Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, type, actor_admin_id, site_id, page_id, comment_id, aggregate_type, aggregate_id, payload_json, occurred_at
		FROM events
		ORDER BY occurred_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func scanEvent(scanner interface{ Scan(...any) error }) (domain.Event, error) {
	var event domain.Event
	var actorID, siteID, pageID sql.NullInt64
	var commentID sql.NullString
	var payloadJSON, occurredAt string
	if err := scanner.Scan(&event.ID, &event.Type, &actorID, &siteID, &pageID, &commentID, &event.AggregateType, &event.AggregateID, &payloadJSON, &occurredAt); err != nil {
		return domain.Event{}, err
	}
	event.ActorAdminID = nullableInt64(actorID)
	event.SiteID = nullableInt64(siteID)
	event.PageID = nullableInt64(pageID)
	event.CommentID = nullableString(commentID)
	event.OccurredAt = parseTime(occurredAt)
	if err := json.Unmarshal([]byte(payloadJSON), &event.Payload); err != nil {
		event.Payload = map[string]any{}
	}
	return event, nil
}
