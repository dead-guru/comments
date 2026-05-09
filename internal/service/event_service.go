package service

import (
	"context"

	"deadcomments/internal/domain"
	"deadcomments/internal/repository"
)

type EventService struct {
	events *repository.EventRepository
}

func NewEventService(events *repository.EventRepository) *EventService {
	return &EventService{events: events}
}

func (s *EventService) Recent(ctx context.Context, limit int) ([]domain.Event, error) {
	return s.events.List(ctx, limit)
}

func (s *EventService) List(ctx context.Context, filter repository.EventFilter) ([]domain.Event, error) {
	return s.events.ListFiltered(ctx, filter)
}
