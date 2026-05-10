package service

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

const (
	maxAnnotationSelectorLength = 500
	maxAnnotationQuoteLength    = 2000
	maxAnnotationContextLength  = 240
	maxAnnotationMetadataBytes  = 4 << 10
)

var safeAnnotationSelectorRe = regexp.MustCompile(`^(#[A-Za-z0-9_-]+|\[data-dc-(anchor|annotation-root)="[A-Za-z0-9_./-]+"\])$`)

type AnnotationService struct {
	sites       *repository.SiteRepository
	annotations *repository.AnnotationRepository
	comments    *CommentService
	events      event.Publisher
}

type AnnotationCreateResult struct {
	CommentResult CommentCreateResult
	Annotation    *domain.Annotation
	Reused        bool
}

func NewAnnotationService(sites *repository.SiteRepository, annotations *repository.AnnotationRepository, comments *CommentService, events ...event.Publisher) *AnnotationService {
	return &AnnotationService{
		sites:       sites,
		annotations: annotations,
		comments:    comments,
		events:      optionalPublisher(events),
	}
}

func (s *AnnotationService) PublicByPage(ctx context.Context, siteKey, pageKey string) (*domain.Page, []*domain.Annotation, error) {
	site, err := s.sites.ByKey(ctx, siteKey)
	if err != nil || site == nil {
		return nil, nil, err
	}
	page, err := s.comments.findOrCreatePage(ctx, site, pageKey, "", "")
	if err != nil || page == nil {
		return nil, nil, err
	}
	if page.State == domain.PageHidden {
		return page, nil, nil
	}
	annotations, err := s.annotations.ApprovedByPage(ctx, page.ID)
	if err != nil {
		return nil, nil, err
	}
	return page, annotations, nil
}

func (s *AnnotationService) CreateDetailed(ctx context.Context, input domain.AnnotationCreateInput) (AnnotationCreateResult, error) {
	selector := strings.TrimSpace(input.Selector)
	selectedText := strings.TrimSpace(input.SelectedText)
	prefix := trimAnnotationPrefix(input.SelectionPrefix)
	suffix := trimAnnotationSuffix(input.SelectionSuffix)
	if selector == "" {
		return AnnotationCreateResult{}, errors.New("annotation selector is required")
	}
	if selectedText == "" {
		return AnnotationCreateResult{}, errors.New("selected text is required")
	}
	if len([]rune(selector)) > maxAnnotationSelectorLength {
		return AnnotationCreateResult{}, errors.New("annotation selector is too long")
	}
	if !safeAnnotationSelectorRe.MatchString(selector) {
		return AnnotationCreateResult{}, errors.New("annotation selector is invalid")
	}
	if len([]rune(selectedText)) > maxAnnotationQuoteLength {
		return AnnotationCreateResult{}, errors.New("selected text is too long")
	}
	metadataJSON, err := validateAnnotationMetadataJSON(input.MetadataJSON)
	if err != nil {
		return AnnotationCreateResult{}, err
	}
	textHash := repository.AnnotationTextHash(selectedText)
	citationKey := repository.AnnotationCitationKey(selector, textHash)
	_, page, err := s.comments.resolveCreateTarget(ctx, input.CommentCreateInput)
	if err != nil {
		return AnnotationCreateResult{}, err
	}
	existing, err := s.annotations.ActiveByPageCitationKey(ctx, page.ID, citationKey)
	if err != nil {
		return AnnotationCreateResult{}, err
	}
	if existing != nil && existing.Comment != nil {
		parentID := existing.Comment.ID
		input.CommentCreateInput.ParentID = &parentID
		commentResult, err := s.comments.CreateDetailed(ctx, input.CommentCreateInput)
		if err != nil {
			return AnnotationCreateResult{}, err
		}
		comment := commentResult.Comment
		if comment == nil {
			return AnnotationCreateResult{CommentResult: commentResult, Annotation: existing, Reused: true}, nil
		}
		name := existing.Comment.AuthorDisplayName
		if name == "" {
			name = existing.Comment.AuthorName
		}
		comment.ReplyingToAuthor = &name
		existing.Comment.Children = append(existing.Comment.Children, comment)
		return AnnotationCreateResult{CommentResult: commentResult, Annotation: existing, Reused: true}, nil
	}
	commentResult, err := s.comments.CreateDetailed(ctx, input.CommentCreateInput)
	if err != nil {
		return AnnotationCreateResult{}, err
	}
	comment := commentResult.Comment
	if comment == nil {
		return AnnotationCreateResult{CommentResult: commentResult}, nil
	}
	annotation := &domain.Annotation{
		ID:              newID(),
		SiteID:          comment.SiteID,
		SiteKey:         comment.SiteKey,
		PageID:          comment.PageID,
		PageKey:         comment.PageKey,
		CommentID:       comment.ID,
		Selector:        selector,
		CitationKey:     citationKey,
		SelectedText:    selectedText,
		SelectionPrefix: prefix,
		SelectionSuffix: suffix,
		TextStart:       input.TextStart,
		TextEnd:         input.TextEnd,
		TextHash:        textHash,
		MetadataJSON:    metadataJSON,
		Comment:         comment,
	}
	if err := s.annotations.Create(ctx, annotation); err != nil {
		return AnnotationCreateResult{}, err
	}
	if err := publish(ctx, s.events, domain.Event{
		Type:          domain.EventAnnotationCreated,
		SiteID:        int64Ptr(comment.SiteID),
		PageID:        int64Ptr(comment.PageID),
		CommentID:     stringPtr(comment.ID),
		AggregateType: "annotation",
		AggregateID:   annotation.ID,
		Payload: map[string]any{
			"selector":          selector,
			"selected_text_len": len([]rune(selectedText)),
			"text_hash":         annotation.TextHash,
			"citation_key":      annotation.CitationKey,
			"comment_status":    comment.Status,
		},
	}); err != nil {
		return AnnotationCreateResult{}, err
	}
	return AnnotationCreateResult{CommentResult: commentResult, Annotation: annotation}, nil
}

func validateAnnotationMetadataJSON(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}
	if len([]byte(raw)) > maxAnnotationMetadataBytes {
		return nil, errors.New("annotation metadata is too large")
	}
	if !json.Valid([]byte(raw)) {
		return nil, errors.New("annotation metadata must be valid JSON")
	}
	return &raw, nil
}

func trimAnnotationPrefix(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxAnnotationContextLength {
		return value
	}
	return string(runes[len(runes)-maxAnnotationContextLength:])
}

func trimAnnotationSuffix(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxAnnotationContextLength {
		return value
	}
	return string(runes[:maxAnnotationContextLength])
}
