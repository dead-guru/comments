package service

import (
	"context"
	"errors"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type PageService struct {
	pages  *repository.PageRepository
	events event.Publisher
}

func NewPageService(pages *repository.PageRepository, events ...event.Publisher) *PageService {
	return &PageService{pages: pages, events: optionalPublisher(events)}
}

func (s *PageService) FindOrCreate(ctx context.Context, site *domain.Site, pageKey, title, pageURL string) (*domain.Page, error) {
	pageKey = strings.TrimSpace(pageKey)
	if pageKey == "" {
		return nil, errors.New("page key is required")
	}
	page, created, err := s.pages.FindOrCreate(ctx, site, pageKey, title, pageURL)
	if err != nil || !created || page == nil {
		return page, err
	}
	return page, publish(ctx, s.events, domain.Event{
		Type:          domain.EventPageAutoCreated,
		SiteID:        int64Ptr(site.ID),
		PageID:        int64Ptr(page.ID),
		AggregateType: "page",
		AggregateID:   int64ID(page.ID),
		Payload: map[string]any{
			"page_key": page.PageKey,
			"title":    page.Title,
			"url":      page.URL,
			"state":    page.State,
		},
	})
}

func (s *PageService) ByID(ctx context.Context, id int64) (*domain.Page, error) {
	return s.pages.ByID(ctx, id)
}

func (s *PageService) List(ctx context.Context, siteID *int64, state, search string) ([]*domain.Page, error) {
	return s.pages.List(ctx, siteID, state, search)
}

func (s *PageService) Count(ctx context.Context) (int, error) {
	return s.pages.Count(ctx)
}

func (s *PageService) SetState(ctx context.Context, id int64, state domain.PageState) error {
	page, err := s.pages.ByID(ctx, id)
	if err != nil {
		return err
	}
	if page == nil {
		return nil
	}
	oldState := page.State
	if err := s.pages.SetState(ctx, id, state); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventPageStateChanged,
		SiteID:        int64Ptr(page.SiteID),
		PageID:        int64Ptr(page.ID),
		AggregateType: "page",
		AggregateID:   int64ID(page.ID),
		Payload: map[string]any{
			"old_state": oldState,
			"new_state": state,
			"page_key":  page.PageKey,
		},
	})
}
