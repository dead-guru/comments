package service

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	dcauth "deadcomments/internal/auth"
	"deadcomments/internal/db"
	"deadcomments/internal/domain"
	dcevent "deadcomments/internal/event"
	"deadcomments/internal/repository"
)

type authTestDeps struct {
	db       *sql.DB
	admins   *repository.AdminRepository
	sessions *repository.SessionRepository
	bus      *dcevent.Bus
}

type fakeGitHubOAuth struct {
	user       *dcauth.GitHubUser
	err        error
	configured bool
}

func (f *fakeGitHubOAuth) Configured() bool {
	return f.configured
}

func (f *fakeGitHubOAuth) AuthCodeURL(state string) string {
	return "https://github.example/oauth?state=" + state
}

func (f *fakeGitHubOAuth) ExchangeUser(ctx context.Context, code string) (*dcauth.GitHubUser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func newAuthTestDeps(t *testing.T) authTestDeps {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	admins := repository.NewAdminRepository(database)
	sessions := repository.NewSessionRepository(database)
	events := repository.NewEventRepository(database)
	return authTestDeps{
		db:       database,
		admins:   admins,
		sessions: sessions,
		bus:      dcevent.NewBus(events),
	}
}

func TestAuthServiceRejectsEmptyAllowlist(t *testing.T) {
	deps := newAuthTestDeps(t)
	authSvc := NewAuthService(deps.admins, deps.sessions, &fakeGitHubOAuth{
		user:       githubUser("octo"),
		configured: true,
	}, map[string]struct{}{}, "session-secret", time.Hour, deps.bus)

	admin, token, err := authSvc.CompleteGitHubLogin(context.Background(), "code")
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got admin=%v token=%q err=%v", admin, token, err)
	}
	if countAdmins(t, deps.db) != 0 {
		t.Fatal("expected no admin to be created")
	}
	if countAdminSessions(t, deps.db) != 0 {
		t.Fatal("expected no session to be created")
	}
}

func TestAuthServiceRejectsLoginOutsideAllowlist(t *testing.T) {
	deps := newAuthTestDeps(t)
	authSvc := NewAuthService(deps.admins, deps.sessions, &fakeGitHubOAuth{
		user:       githubUser("octo"),
		configured: true,
	}, map[string]struct{}{"someone-else": {}}, "session-secret", time.Hour, deps.bus)

	admin, token, err := authSvc.CompleteGitHubLogin(context.Background(), "code")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got admin=%v token=%q err=%v", admin, token, err)
	}
	if countAdmins(t, deps.db) != 0 {
		t.Fatal("expected no admin to be created")
	}
}

func TestAuthServiceCreatesAllowedAdminSessionWithoutStoringRawToken(t *testing.T) {
	deps := newAuthTestDeps(t)
	authSvc := NewAuthService(deps.admins, deps.sessions, &fakeGitHubOAuth{
		user:       githubUser("Octo"),
		configured: true,
	}, map[string]struct{}{"octo": {}}, "session-secret", time.Hour, deps.bus)

	admin, token, err := authSvc.CompleteGitHubLogin(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if admin == nil || admin.ID == 0 {
		t.Fatalf("expected persisted admin, got %#v", admin)
	}
	if admin.Role != domain.RoleOwner {
		t.Fatalf("expected new allowed admin to be owner, got %s", admin.Role)
	}
	if token == "" {
		t.Fatal("expected session token")
	}

	var tokenHash string
	if err := deps.db.QueryRow(`SELECT token_hash FROM admin_sessions WHERE admin_id=?`, admin.ID).Scan(&tokenHash); err != nil {
		t.Fatal(err)
	}
	if tokenHash == "" || tokenHash == token {
		t.Fatalf("expected stored token hash, got %q for token %q", tokenHash, token)
	}
	if tokenHash != dcauth.TokenHash("session-secret", token) {
		t.Fatal("expected session token to be HMAC hashed with session secret")
	}

	resolved, err := authSvc.AdminForToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if resolved == nil || resolved.ID != admin.ID {
		t.Fatalf("expected token to resolve admin %d, got %#v", admin.ID, resolved)
	}
}

func TestAuthServicePropagatesGitHubExchangeErrors(t *testing.T) {
	deps := newAuthTestDeps(t)
	exchangeErr := errors.New("exchange failed")
	authSvc := NewAuthService(deps.admins, deps.sessions, &fakeGitHubOAuth{
		err:        exchangeErr,
		configured: true,
	}, map[string]struct{}{"octo": {}}, "session-secret", time.Hour, deps.bus)

	_, _, err := authSvc.CompleteGitHubLogin(context.Background(), "bad-code")
	if !errors.Is(err, exchangeErr) {
		t.Fatalf("expected exchange error, got %v", err)
	}
}

func githubUser(login string) *dcauth.GitHubUser {
	email := "octo@example.com"
	name := "Octo Cat"
	avatarURL := "https://avatars.example/octo.png"
	return &dcauth.GitHubUser{
		ID:        123,
		Login:     login,
		Email:     &email,
		Name:      &name,
		AvatarURL: &avatarURL,
	}
}

func countAdmins(t *testing.T, database *sql.DB) int {
	t.Helper()
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func countAdminSessions(t *testing.T, database *sql.DB) int {
	t.Helper()
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM admin_sessions`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
