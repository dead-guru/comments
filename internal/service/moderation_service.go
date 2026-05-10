package service

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type ModerationService struct {
	moderation *repository.ModerationRepository
	comments   *repository.CommentRepository
	events     event.Publisher
}

const (
	recentIPCommentLimit         = 30
	reservedIdentityCommentLimit = 300
	recentIPWindow               = 10 * time.Minute
)

var (
	markdownLinkRe  = regexp.MustCompile(`(?i)(^|[^!])(\[[^\]]+\]\(https?://[^)\s]+\))`)
	markdownImageRe = regexp.MustCompile(`(?i)!\[[^\]]*\]\(https?://[^)\s]+\)`)
	rawURLRe        = regexp.MustCompile(`(?i)https?://[^\s)\]]+`)
)

func NewModerationService(moderation *repository.ModerationRepository, comments *repository.CommentRepository, events ...event.Publisher) *ModerationService {
	return &ModerationService{moderation: moderation, comments: comments, events: optionalPublisher(events)}
}

func (s *ModerationService) Decide(ctx context.Context, site *domain.Site, input domain.CommentCreateInput, ipHash string, identity domain.IdentityResolution) (domain.ModerationDecision, error) {
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
		limit, window := rateLimitForIdentity(identity)
		stats, err := s.comments.RecentIPStats(ctx, ipHash, time.Now().UTC().Add(-window))
		if err != nil {
			return domain.ModerationDecision{}, err
		}
		if stats.Count >= limit {
			return domain.ModerationDecision{
				Status:     domain.CommentRejected,
				Reason:     "rate limit",
				RetryAfter: retryAfter(window, stats.Oldest),
				Limit:      limit,
				Window:     window,
			}, nil
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
		if wordBanMatches(body, ban.Pattern) {
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

func wordBanMatches(normalizedBody, pattern string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	return pattern != "" && strings.Contains(normalizedBody, pattern)
}

func rateLimitForIdentity(identity domain.IdentityResolution) (int, time.Duration) {
	if identity.TripcodeKind == domain.TripcodeReserved && identity.IdentityID != nil {
		return reservedIdentityCommentLimit, recentIPWindow
	}
	return recentIPCommentLimit, recentIPWindow
}

func retryAfter(window time.Duration, oldest time.Time) time.Duration {
	if oldest.IsZero() {
		return window
	}
	until := oldest.Add(window).Sub(time.Now().UTC())
	if until < time.Second {
		return time.Second
	}
	return until.Round(time.Second)
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

func (s *ModerationService) ListIPBansPaginated(ctx context.Context, limit, offset int) ([]*domain.IPBan, error) {
	return s.moderation.ListIPBansPaginated(ctx, limit, offset)
}

func (s *ModerationService) ListWordBans(ctx context.Context) ([]*domain.WordBan, error) {
	return s.moderation.ListWordBans(ctx)
}

func (s *ModerationService) ListWordBansPaginated(ctx context.Context, limit, offset int) ([]*domain.WordBan, error) {
	return s.moderation.ListWordBansPaginated(ctx, limit, offset)
}

func (s *ModerationService) AddWordBan(ctx context.Context, ban *domain.WordBan) error {
	ban.Pattern = strings.TrimSpace(ban.Pattern)
	if ban.Pattern == "" {
		return nil
	}
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
	markdownRanges := markdownLinkRanges(s)
	imageRanges := markdownImageRanges(s)
	withoutMarkdown := removeRanges(s, append(markdownRanges, imageRanges...))
	return len(markdownRanges) + len(rawURLRe.FindAllString(withoutMarkdown, -1))
}

type textRange struct {
	start int
	end   int
}

func markdownLinkRanges(s string) []textRange {
	matches := markdownLinkRe.FindAllStringSubmatchIndex(s, -1)
	ranges := make([]textRange, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 6 && match[4] >= 0 && match[5] >= 0 {
			ranges = append(ranges, textRange{start: match[4], end: match[5]})
		}
	}
	return ranges
}

func markdownImageRanges(s string) []textRange {
	matches := markdownImageRe.FindAllStringIndex(s, -1)
	ranges := make([]textRange, 0, len(matches))
	for _, match := range matches {
		ranges = append(ranges, textRange{start: match[0], end: match[1]})
	}
	return ranges
}

func removeRanges(s string, ranges []textRange) string {
	if len(ranges) == 0 {
		return s
	}
	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].start < ranges[j].start
	})
	var b strings.Builder
	last := 0
	for _, r := range ranges {
		if r.start < last {
			continue
		}
		if r.start > last {
			b.WriteString(s[last:r.start])
		}
		last = r.end
	}
	if last < len(s) {
		b.WriteString(s[last:])
	}
	return b.String()
}
