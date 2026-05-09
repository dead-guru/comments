package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

var (
	ErrReservedIdentitySecretRequired = errors.New("this name is reserved")
	ErrReservedIdentitySecretInvalid  = errors.New("this name is reserved")
)

type IdentityService struct {
	identities     *repository.IdentityRepository
	tripcodeSecret string
	events         event.Publisher
}

func NewIdentityService(identities *repository.IdentityRepository, tripcodeSecret string, events ...event.Publisher) *IdentityService {
	return &IdentityService{identities: identities, tripcodeSecret: tripcodeSecret, events: optionalPublisher(events)}
}

func (s *IdentityService) ResolveForComment(ctx context.Context, siteID int64, rawAuthorName string) (domain.IdentityResolution, error) {
	displayName, secret := ParseTripcodeName(rawAuthorName)
	if displayName == "" {
		return domain.IdentityResolution{}, errors.New("author name is required")
	}
	normalizedName := NormalizeIdentityName(displayName)
	identity, err := s.identities.ByNormalizedName(ctx, siteID, normalizedName)
	if err != nil {
		return domain.IdentityResolution{}, err
	}
	if identity != nil {
		if secret == "" {
			return domain.IdentityResolution{}, ErrReservedIdentitySecretRequired
		}
		if bcrypt.CompareHashAndPassword([]byte(identity.SecretHash), []byte(secret)) != nil {
			return domain.IdentityResolution{}, ErrReservedIdentitySecretInvalid
		}
		return domain.IdentityResolution{
			DisplayName:    identity.DisplayName,
			IdentityID:     int64Ptr(identity.ID),
			TripcodePublic: stringPtr(identity.PublicTripcode),
			TripcodeKind:   domain.TripcodeReserved,
			BadgeType:      &identity.BadgeType,
			BadgeLabel:     identity.BadgeLabel,
		}, nil
	}
	if secret == "" {
		return domain.IdentityResolution{
			DisplayName:  displayName,
			TripcodeKind: domain.TripcodeNone,
		}, nil
	}
	tripcode := s.GeneratePublicTripcode(secret)
	return domain.IdentityResolution{
		DisplayName:    displayName,
		TripcodePublic: &tripcode,
		TripcodeKind:   domain.TripcodeAnonymous,
	}, nil
}

func (s *IdentityService) Create(ctx context.Context, input domain.IdentityCreateInput) (*domain.Identity, error) {
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, errors.New("display name is required")
	}
	if input.Secret == "" {
		return nil, errors.New("tripcode secret is required")
	}
	secretHash, err := hashIdentitySecret(input.Secret)
	if err != nil {
		return nil, err
	}
	badgeType := normalizeBadgeType(input.BadgeType)
	publicTripcode := strings.TrimSpace(input.PublicTripcode)
	if publicTripcode == "" {
		publicTripcode = s.GeneratePublicTripcode(input.Secret)
	}
	identity := &domain.Identity{
		SiteID:           input.SiteID,
		DisplayName:      displayName,
		NormalizedName:   NormalizeIdentityName(displayName),
		Type:             domain.IdentityReserved,
		SecretHash:       secretHash,
		PublicTripcode:   publicTripcode,
		BadgeType:        badgeType,
		CreatedByAdminID: input.CreatedByAdminID,
	}
	if label := strings.TrimSpace(input.BadgeLabel); label != "" {
		identity.BadgeLabel = &label
	}
	if err := s.identities.Create(ctx, identity); err != nil {
		return nil, err
	}
	return identity, publish(ctx, s.events, domain.Event{
		Type:          domain.EventIdentityCreated,
		SiteID:        identity.SiteID,
		AggregateType: "identity",
		AggregateID:   int64ID(identity.ID),
		Payload: map[string]any{
			"display_name":    identity.DisplayName,
			"normalized_name": identity.NormalizedName,
			"badge_type":      identity.BadgeType,
		},
	})
}

func (s *IdentityService) Update(ctx context.Context, input domain.IdentityUpdateInput) (*domain.Identity, error) {
	identity, err := s.identities.ByID(ctx, input.ID)
	if err != nil || identity == nil {
		return identity, err
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		return nil, errors.New("display name is required")
	}
	identity.SiteID = input.SiteID
	identity.DisplayName = displayName
	identity.NormalizedName = NormalizeIdentityName(displayName)
	if publicTripcode := strings.TrimSpace(input.PublicTripcode); publicTripcode != "" {
		identity.PublicTripcode = publicTripcode
	}
	identity.BadgeType = normalizeBadgeType(input.BadgeType)
	identity.BadgeLabel = nil
	if label := strings.TrimSpace(input.BadgeLabel); label != "" {
		identity.BadgeLabel = &label
	}
	if err := s.identities.Update(ctx, identity); err != nil {
		return nil, err
	}
	return identity, publish(ctx, s.events, domain.Event{
		Type:          domain.EventIdentityUpdated,
		SiteID:        identity.SiteID,
		AggregateType: "identity",
		AggregateID:   int64ID(identity.ID),
		Payload: map[string]any{
			"display_name":    identity.DisplayName,
			"normalized_name": identity.NormalizedName,
			"badge_type":      identity.BadgeType,
		},
	})
}

func (s *IdentityService) ResetSecret(ctx context.Context, id int64, secret string) error {
	if strings.TrimSpace(secret) == "" {
		return errors.New("tripcode secret is required")
	}
	identity, err := s.identities.ByID(ctx, id)
	if err != nil || identity == nil {
		return err
	}
	secretHash, err := hashIdentitySecret(secret)
	if err != nil {
		return err
	}
	publicTripcode := s.GeneratePublicTripcode(secret)
	if err := s.identities.ResetSecret(ctx, id, secretHash, publicTripcode); err != nil {
		return err
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventIdentitySecretReset,
		SiteID:        identity.SiteID,
		AggregateType: "identity",
		AggregateID:   int64ID(id),
	})
}

func (s *IdentityService) Delete(ctx context.Context, id int64) error {
	identity, err := s.identities.ByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.identities.Delete(ctx, id); err != nil {
		return err
	}
	var siteID *int64
	if identity != nil {
		siteID = identity.SiteID
	}
	return publish(ctx, s.events, domain.Event{
		Type:          domain.EventIdentityDeleted,
		SiteID:        siteID,
		AggregateType: "identity",
		AggregateID:   int64ID(id),
	})
}

func (s *IdentityService) ByID(ctx context.Context, id int64) (*domain.Identity, error) {
	return s.identities.ByID(ctx, id)
}

func (s *IdentityService) List(ctx context.Context) ([]*domain.Identity, error) {
	return s.identities.List(ctx)
}

func (s *IdentityService) GeneratePublicTripcode(secret string) string {
	h := hmac.New(sha256.New, []byte(s.tripcodeSecret))
	_, _ = h.Write([]byte(normalizeTripcodeSecret(secret)))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(h.Sum(nil))
	if len(encoded) > 10 {
		return encoded[:10]
	}
	return encoded
}

func ParseTripcodeName(raw string) (displayName string, secret string) {
	parts := strings.SplitN(raw, "##", 2)
	displayName = collapseSpaces(strings.TrimSpace(parts[0]))
	if len(parts) == 2 {
		secret = strings.TrimSpace(parts[1])
	}
	return displayName, secret
}

func NormalizeIdentityName(name string) string {
	return strings.ToLower(collapseSpaces(strings.TrimSpace(name)))
}

func normalizeTripcodeSecret(secret string) string {
	return strings.TrimSpace(secret)
}

func collapseSpaces(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func hashIdentitySecret(secret string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	return string(hash), err
}

func normalizeBadgeType(value domain.BadgeType) domain.BadgeType {
	switch value {
	case domain.BadgeAdmin, domain.BadgeAuthor, domain.BadgeCustom:
		return value
	default:
		return domain.BadgeVerified
	}
}
