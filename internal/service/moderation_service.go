package service

import (
	"context"
	"regexp"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type ModerationService struct {
	moderation *repository.ModerationRepository
	comments   *repository.CommentRepository
	events     event.Publisher
}

const recentIPCommentLimit = 30

func NewModerationService(moderation *repository.ModerationRepository, comments *repository.CommentRepository, events ...event.Publisher) *ModerationService {
	return &ModerationService{moderation: moderation, comments: comments, events: optionalPublisher(events)}
}

func (s *ModerationService) Decide(ctx context.Context, site *domain.Site, input domain.CommentCreateInput, ipHash string) (domain.ModerationDecision, error) {
	if strings.TrimSpace(input.Honeypot) != "" {
		return domain.ModerationDecision{Status: domain.CommentSpam, Reason: "honeypot"}, nil
	}
	if ipHash != "" {
		banned, err := s.moderation.IsIPBanned(ctx, site.ID, ipHash)
		if err != nil {
			return domain.ModerationDecision{}, err
		}
		if banned {
			return domain.ModerationDecision{Status: domain.CommentRejected, Reason: "ip banned"}, nil
		}
		count, err := s.comments.RecentIPCount(ctx, ipHash)
		if err != nil {
			return domain.ModerationDecision{}, err
		}
		if count >= recentIPCommentLimit {
			return domain.ModerationDecision{Status: domain.CommentRejected, Reason: "rate limit"}, nil
		}
		dupes, err := s.comments.RecentSameIP(ctx, ipHash, input.BodyMarkdown)
		if err != nil {
			return domain.ModerationDecision{}, err
		}
		if dupes >= 2 {
			return domain.ModerationDecision{Status: domain.CommentSpam, Reason: "duplicate body"}, nil
		}
	}
	wordBans, err := s.moderation.WordBans(ctx, site.ID)
	if err != nil {
		return domain.ModerationDecision{}, err
	}
	body := strings.ToLower(input.BodyMarkdown)
	for _, ban := range wordBans {
		if matched, _ := regexp.MatchString(strings.ToLower(ban.Pattern), body); matched || strings.Contains(body, strings.ToLower(ban.Pattern)) {
			switch ban.Action {
			case domain.WordBanReject:
				return domain.ModerationDecision{Status: domain.CommentRejected, Reason: "word ban"}, nil
			case domain.WordBanSpam:
				return domain.ModerationDecision{Status: domain.CommentSpam, Reason: "word ban"}, nil
			default:
				return domain.ModerationDecision{Status: domain.CommentPending, Reason: "word ban"}, nil
			}
		}
	}
	if linkCount(input.BodyMarkdown) > 5 {
		return domain.ModerationDecision{Status: domain.CommentSpam, Reason: "too many links"}, nil
	}
	if site.DefaultModerationMode == domain.ModerationAuto {
		return domain.ModerationDecision{Status: domain.CommentApproved, Reason: "auto moderation"}, nil
	}
	return domain.ModerationDecision{Status: domain.CommentPending, Reason: "manual moderation"}, nil
}

func (s *ModerationService) AddIPBan(ctx context.Context, ban *domain.IPBan) error {
	if err := s.moderation.CreateIPBan(ctx, ban); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventIPBanCreated,
		SiteID:        ban.SiteID,
		AggregateType: "ip_ban",
		AggregateID:   int64ID(ban.ID),
		Payload: map[string]any{
			"ip_hash": ban.IPHash,
			"reason":  ban.Reason,
		},
	})
}

func (s *ModerationService) ListIPBans(ctx context.Context) ([]*domain.IPBan, error) {
	return s.moderation.ListIPBans(ctx)
}

func (s *ModerationService) ListWordBans(ctx context.Context) ([]*domain.WordBan, error) {
	return s.moderation.ListWordBans(ctx)
}

func (s *ModerationService) AddWordBan(ctx context.Context, ban *domain.WordBan) error {
	if err := s.moderation.CreateWordBan(ctx, ban); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventWordBanCreated,
		SiteID:        ban.SiteID,
		AggregateType: "word_ban",
		AggregateID:   int64ID(ban.ID),
		Payload: map[string]any{
			"pattern": ban.Pattern,
			"action":  ban.Action,
		},
	})
}

func (s *ModerationService) DeleteBan(ctx context.Context, id int64) error {
	if err := s.moderation.DeleteBan(ctx, id); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventIPBanDeleted,
		AggregateType: "ip_ban",
		AggregateID:   int64ID(id),
	})
}

func (s *ModerationService) DeleteWordBan(ctx context.Context, id int64) error {
	if err := s.moderation.DeleteWordBan(ctx, id); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventWordBanDeleted,
		AggregateType: "word_ban",
		AggregateID:   int64ID(id),
	})
}

func (s *ModerationService) EventsForComment(ctx context.Context, commentID string) ([]*domain.ModerationEvent, error) {
	return s.moderation.EventsForComment(ctx, commentID)
}

func linkCount(s string) int {
	return strings.Count(s, "http://") + strings.Count(s, "https://") + strings.Count(s, "](")
}
