package service

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type CommentService struct {
	sites      *repository.SiteRepository
	pages      *repository.PageRepository
	comments   *repository.CommentRepository
	identities *IdentityService
	moderation *ModerationService
	markdown   *MarkdownService
	secret     string
	events     event.Publisher
}

func NewCommentService(sites *repository.SiteRepository, pages *repository.PageRepository, comments *repository.CommentRepository, identities *IdentityService, moderation *ModerationService, markdown *MarkdownService, secret string, events ...event.Publisher) *CommentService {
	return &CommentService{sites: sites, pages: pages, comments: comments, identities: identities, moderation: moderation, markdown: markdown, secret: secret, events: optionalPublisher(events)}
}

func (s *CommentService) Create(ctx context.Context, input domain.CommentCreateInput) (*domain.Comment, string, error) {
	site, err := s.sites.ByKey(ctx, input.SiteKey)
	if err != nil {
		return nil, "", err
	}
	if site == nil {
		return nil, "", errors.New("site not found")
	}
	origin := firstNonEmpty(input.Origin, input.Referer)
	if !NewSiteService(s.sites).OriginAllowed(site, origin) {
		return nil, "", errors.New("origin is not allowed for this site")
	}
	page, err := s.findOrCreatePage(ctx, site, input.PageKey, input.PageTitle, input.PageURL)
	if err != nil {
		return nil, "", err
	}
	if page == nil {
		return nil, "", errors.New("page not found")
	}
	if !page.CanPost() {
		return nil, "", errors.New("page does not allow new comments")
	}
	identity, err := s.identities.ResolveForComment(ctx, site.ID, input.AuthorName)
	if err != nil {
		return nil, "", err
	}
	input.AuthorName = identity.DisplayName
	input.BodyMarkdown = strings.TrimSpace(input.BodyMarkdown)
	if input.AuthorName == "" {
		return nil, "", errors.New("author name is required")
	}
	if input.BodyMarkdown == "" {
		return nil, "", errors.New("comment body is required")
	}
	if len([]rune(input.BodyMarkdown)) > site.MaxCommentLength {
		return nil, "", errors.New("comment is too long")
	}
	var parent *domain.Comment
	if input.ParentID != nil && *input.ParentID != "" {
		if !site.AllowReplies {
			return nil, "", errors.New("replies are disabled")
		}
		parent, err = s.comments.ByID(ctx, *input.ParentID)
		if err != nil {
			return nil, "", err
		}
		if parent == nil || parent.PageID != page.ID {
			return nil, "", errors.New("parent comment not found on this page")
		}
	}
	bodyHTML, err := s.markdown.Render(input.BodyMarkdown)
	if err != nil {
		return nil, "", err
	}
	ipHash := ""
	if input.IP != "" {
		ipHash = HashValue(s.secret, input.IP)
	}
	decision, err := s.moderation.Decide(ctx, site, input, ipHash)
	if err != nil {
		return nil, "", err
	}
	c := &domain.Comment{
		ID:                newID(),
		SiteID:            site.ID,
		PageID:            page.ID,
		AuthorName:        identity.DisplayName,
		AuthorDisplayName: identity.DisplayName,
		IdentityID:        identity.IdentityID,
		TripcodePublic:    identity.TripcodePublic,
		TripcodeKind:      identity.TripcodeKind,
		BadgeType:         identity.BadgeType,
		BadgeLabel:        identity.BadgeLabel,
		BodyMarkdown:      input.BodyMarkdown,
		BodyHTML:          bodyHTML,
		Status:            decision.Status,
		Depth:             0,
	}
	if parent != nil {
		c.ParentID = &parent.ID
		c.Depth = parent.Depth + 1
		if parent.RootID != nil {
			c.RootID = parent.RootID
		} else {
			c.RootID = &parent.ID
		}
	}
	if input.AuthorEmail != "" {
		email := strings.ToLower(strings.TrimSpace(input.AuthorEmail))
		v := HashValue(s.secret, email)
		c.AuthorEmailHash = &v
		avatar := publicEmailAvatarHash(email)
		c.AuthorAvatarHash = &avatar
	}
	if input.AuthorWebsite != "" {
		if u, err := url.ParseRequestURI(input.AuthorWebsite); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			w := input.AuthorWebsite
			c.AuthorWebsite = &w
		}
	}
	if ipHash != "" {
		c.IPHash = &ipHash
	}
	if input.UserAgent != "" {
		ua := HashValue(s.secret, input.UserAgent)
		c.UserAgentHash = &ua
	}
	if err := s.comments.Create(ctx, c); err != nil {
		return nil, "", err
	}
	c.Path = repository.CommentPath(parent, c.ID)
	if parent == nil {
		c.RootID = &c.ID
	}
	if err := s.comments.UpdateTreeFields(ctx, c.ID, c.RootID, c.Depth, c.Path); err != nil {
		return nil, "", err
	}
	_ = s.pages.Recount(ctx, page.ID)
	if err := publish(ctx, s.events, domain.Event{
		Type:          domain.EventCommentCreated,
		SiteID:        int64Ptr(site.ID),
		PageID:        int64Ptr(page.ID),
		CommentID:     stringPtr(c.ID),
		AggregateType: "comment",
		AggregateID:   c.ID,
		Payload: map[string]any{
			"status":            c.Status,
			"reason":            decision.Reason,
			"author_name":       c.AuthorDisplayName,
			"page_key":          page.PageKey,
			"parent_id":         c.ParentID,
			"tripcode_kind":     c.TripcodeKind,
			"identity_id":       c.IdentityID,
			"moderation_mode":   site.DefaultModerationMode,
			"has_author_email":  c.AuthorEmailHash != nil,
			"has_author_site":   c.AuthorWebsite != nil,
			"body_markdown_len": len([]rune(c.BodyMarkdown)),
		},
	}); err != nil {
		return nil, "", err
	}
	return c, decision.Reason, nil
}

func publicEmailAvatarHash(email string) string {
	sum := md5.Sum([]byte(email))
	return hex.EncodeToString(sum[:])
}

func (s *CommentService) PublicTree(ctx context.Context, siteKey, pageKey string, sort domain.CommentSort) (*domain.Page, []*domain.Comment, error) {
	site, err := s.sites.ByKey(ctx, siteKey)
	if err != nil || site == nil {
		return nil, nil, err
	}
	page, err := s.findOrCreatePage(ctx, site, pageKey, "", "")
	if err != nil || page == nil {
		return nil, nil, err
	}
	if page.State == domain.PageHidden {
		return page, nil, nil
	}
	comments, err := s.comments.ApprovedByPage(ctx, page.ID, NormalizeCommentSort(string(sort)))
	if err != nil {
		return nil, nil, err
	}
	return page, BuildTree(comments), nil
}

func NormalizeCommentSort(raw string) domain.CommentSort {
	switch domain.CommentSort(strings.ToLower(strings.TrimSpace(raw))) {
	case domain.CommentSortNewest:
		return domain.CommentSortNewest
	case domain.CommentSortBest:
		return domain.CommentSortBest
	default:
		return domain.CommentSortOldest
	}
}

func (s *CommentService) AdminList(ctx context.Context, status, search string, siteID, pageID *int64, limit int) ([]*domain.Comment, error) {
	return s.comments.List(ctx, status, search, siteID, pageID, limit)
}

func (s *CommentService) AdminListFiltered(ctx context.Context, filter repository.CommentListFilter) ([]*domain.Comment, error) {
	return s.comments.ListFiltered(ctx, filter)
}

func (s *CommentService) ByID(ctx context.Context, id string) (*domain.Comment, error) {
	return s.comments.ByID(ctx, id)
}

func (s *CommentService) SetStatus(ctx context.Context, id string, status domain.CommentStatus) error {
	c, err := s.comments.ByID(ctx, id)
	if err != nil || c == nil {
		return err
	}
	oldStatus := c.Status
	if err := s.comments.UpdateStatus(ctx, id, status); err != nil {
		return err
	}
	if err := s.pages.Recount(ctx, c.PageID); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventCommentStatusSet,
		SiteID:        int64Ptr(c.SiteID),
		PageID:        int64Ptr(c.PageID),
		CommentID:     stringPtr(c.ID),
		AggregateType: "comment",
		AggregateID:   c.ID,
		Payload: map[string]any{
			"old_status": oldStatus,
			"status":     status,
		},
	})
}

func (s *CommentService) Edit(ctx context.Context, id, markdownBody string) error {
	c, err := s.comments.ByID(ctx, id)
	if err != nil || c == nil {
		return err
	}
	html, err := s.markdown.Render(markdownBody)
	if err != nil {
		return err
	}
	if err := s.comments.UpdateBody(ctx, id, markdownBody, html); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventCommentEdited,
		SiteID:        int64Ptr(c.SiteID),
		PageID:        int64Ptr(c.PageID),
		CommentID:     stringPtr(c.ID),
		AggregateType: "comment",
		AggregateID:   c.ID,
		Payload: map[string]any{
			"body_markdown_len": len([]rune(markdownBody)),
		},
	})
}

func (s *CommentService) BanIPAndSpam(ctx context.Context, commentID string, adminID *int64, reason string) error {
	c, err := s.comments.ByID(ctx, commentID)
	if err != nil || c == nil || c.IPHash == nil {
		return err
	}
	ban := &domain.IPBan{SiteID: &c.SiteID, IPHash: *c.IPHash, CreatedByAdminID: adminID}
	if reason != "" {
		ban.Reason = &reason
	}
	if err := s.moderation.AddIPBan(ctx, ban); err != nil {
		return err
	}
	if err := s.comments.MarkIPSpam(ctx, c.SiteID, *c.IPHash); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventCommentIPBanned,
		SiteID:        int64Ptr(c.SiteID),
		PageID:        int64Ptr(c.PageID),
		CommentID:     stringPtr(c.ID),
		AggregateType: "comment",
		AggregateID:   c.ID,
		Payload: map[string]any{
			"reason":  reason,
			"ip_hash": *c.IPHash,
		},
	})
}

func (s *CommentService) CountByStatus(ctx context.Context, status domain.CommentStatus) (int, error) {
	return s.comments.CountByStatus(ctx, status)
}

func (s *CommentService) CountTodayByStatus(ctx context.Context, status domain.CommentStatus) (int, error) {
	return s.comments.CountTodayByStatus(ctx, status)
}

func (s *CommentService) CountByIdentity(ctx context.Context, identityID int64) (int, error) {
	return s.comments.CountByIdentity(ctx, identityID)
}

func (s *CommentService) findOrCreatePage(ctx context.Context, site *domain.Site, pageKey, title, pageURL string) (*domain.Page, error) {
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

func BuildTree(comments []*domain.Comment) []*domain.Comment {
	byID := map[string]*domain.Comment{}
	var roots []*domain.Comment
	for _, c := range comments {
		c.Children = nil
		byID[c.ID] = c
	}
	for _, c := range comments {
		if c.ParentID == nil || byID[*c.ParentID] == nil {
			roots = append(roots, c)
			continue
		}
		parent := byID[*c.ParentID]
		name := parent.AuthorDisplayName
		if name == "" {
			name = parent.AuthorName
		}
		c.ReplyingToAuthor = &name
		if parent.ParentID == nil {
			parent.Children = append(parent.Children, c)
		} else if c.RootID != nil && byID[*c.RootID] != nil {
			root := byID[*c.RootID]
			root.Children = append(root.Children, c)
		} else {
			parent.Children = append(parent.Children, c)
		}
	}
	return roots
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "dc-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
