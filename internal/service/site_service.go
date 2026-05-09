package service

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type SiteService struct {
	sites  *repository.SiteRepository
	events event.Publisher
}

func NewSiteService(sites *repository.SiteRepository, events ...event.Publisher) *SiteService {
	return &SiteService{sites: sites, events: optionalPublisher(events)}
}

func (s *SiteService) Create(ctx context.Context, site *domain.Site) error {
	site.Key = repository.NormalizeSiteKey(site.Key)
	if err := validateSite(site); err != nil {
		return err
	}
	if err := s.sites.Create(ctx, site); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventSiteCreated,
		SiteID:        int64Ptr(site.ID),
		AggregateType: "site",
		AggregateID:   int64ID(site.ID),
		Payload: map[string]any{
			"key":  site.Key,
			"name": site.Name,
		},
	})
}

func (s *SiteService) Update(ctx context.Context, site *domain.Site) error {
	site.Key = repository.NormalizeSiteKey(site.Key)
	if err := validateSite(site); err != nil {
		return err
	}
	if err := s.sites.Update(ctx, site); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventSiteUpdated,
		SiteID:        int64Ptr(site.ID),
		AggregateType: "site",
		AggregateID:   int64ID(site.ID),
		Payload: map[string]any{
			"key":  site.Key,
			"name": site.Name,
		},
	})
}

func (s *SiteService) ByKey(ctx context.Context, key string) (*domain.Site, error) {
	return s.sites.ByKey(ctx, key)
}

func (s *SiteService) ByID(ctx context.Context, id int64) (*domain.Site, error) {
	return s.sites.ByID(ctx, id)
}

func (s *SiteService) List(ctx context.Context) ([]*domain.Site, error) {
	return s.sites.List(ctx)
}

func (s *SiteService) Count(ctx context.Context) (int, error) {
	return s.sites.Count(ctx)
}

func (s *SiteService) OriginAllowed(site *domain.Site, originOrReferer string) bool {
	if len(site.AllowedOrigins) == 0 {
		return true
	}
	origin := normalizeOrigin(originOrReferer)
	if origin == "" {
		return false
	}
	for _, allowed := range site.AllowedOrigins {
		allowed = strings.TrimRight(strings.TrimSpace(allowed), "/")
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

var siteKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

func validateSite(site *domain.Site) error {
	if !siteKeyRe.MatchString(site.Key) {
		return errors.New("site key must be 3-64 lowercase letters, numbers, or dashes")
	}
	if strings.TrimSpace(site.Name) == "" {
		return errors.New("site name is required")
	}
	if site.MaxCommentLength <= 0 {
		site.MaxCommentLength = 5000
	}
	if site.DefaultModerationMode == "" {
		site.DefaultModerationMode = domain.ModerationManual
	}
	if site.DefaultPageState == "" {
		site.DefaultPageState = domain.PageOpen
	}
	if site.DefaultTheme == "" {
		site.DefaultTheme = domain.ThemeAuto
	}
	return nil
}

func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
