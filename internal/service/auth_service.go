package service

import (
	"context"
	"errors"
	"strings"
	"time"

	dcauth "deadcomments/internal/auth"
	"deadcomments/internal/domain"
	"deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type AuthService struct {
	admins        *repository.AdminRepository
	sessions      *repository.SessionRepository
	github        *dcauth.GitHubOAuth
	allowed       map[string]struct{}
	sessionSecret string
	sessionTTL    time.Duration
	events        event.Publisher
}

func NewAuthService(admins *repository.AdminRepository, sessions *repository.SessionRepository, github *dcauth.GitHubOAuth, allowed map[string]struct{}, sessionSecret string, sessionTTL time.Duration, events ...event.Publisher) *AuthService {
	return &AuthService{admins: admins, sessions: sessions, github: github, allowed: allowed, sessionSecret: sessionSecret, sessionTTL: sessionTTL, events: optionalPublisher(events)}
}

func (s *AuthService) GitHubConfigured() bool {
	return s.github.Configured()
}

func (s *AuthService) GitHubAuthURL(state string) string {
	return s.github.AuthCodeURL(state)
}

func (s *AuthService) CompleteGitHubLogin(ctx context.Context, code string) (*domain.Admin, string, error) {
	user, err := s.github.ExchangeUser(ctx, code)
	if err != nil {
		return nil, "", err
	}
	login := strings.ToLower(user.Login)
	if len(s.allowed) > 0 {
		if _, ok := s.allowed[login]; !ok {
			return nil, "", errors.New("github login is not allowed")
		}
	}
	admin := &domain.Admin{
		GitHubID:    user.ID,
		GitHubLogin: user.Login,
		Email:       user.Email,
		Name:        user.Name,
		AvatarURL:   user.AvatarURL,
		Role:        domain.RoleOwner,
	}
	if err := s.admins.UpsertGitHub(ctx, admin); err != nil {
		return nil, "", err
	}
	token, err := dcauth.NewToken()
	if err != nil {
		return nil, "", err
	}
	session := &domain.AdminSession{
		AdminID:   admin.ID,
		TokenHash: dcauth.TokenHash(s.sessionSecret, token),
		ExpiresAt: time.Now().UTC().Add(s.sessionTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, "", err
	}
	if err := publish(ctx, s.events, domain.Event{
		Type:          domain.EventAdminLoggedIn,
		ActorAdminID:  int64Ptr(admin.ID),
		AggregateType: "admin",
		AggregateID:   int64ID(admin.ID),
		Payload: map[string]any{
			"github_login": admin.GitHubLogin,
		},
	}); err != nil {
		return nil, "", err
	}
	return admin, token, nil
}

func (s *AuthService) AdminForToken(ctx context.Context, token string) (*domain.Admin, error) {
	if token == "" {
		return nil, nil
	}
	session, err := s.sessions.ByTokenHash(ctx, dcauth.TokenHash(s.sessionSecret, token))
	if err != nil || session == nil {
		return nil, err
	}
	return s.admins.ByID(ctx, session.AdminID)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, dcauth.TokenHash(s.sessionSecret, token))
}
