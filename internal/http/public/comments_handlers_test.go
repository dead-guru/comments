package public

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/db"
	"deadcomments/internal/domain"
	dcevent "deadcomments/internal/event"
	"deadcomments/internal/http/middleware"
	dcmarkdown "deadcomments/internal/markdown"
	"deadcomments/internal/repository"
	"deadcomments/internal/service"
)

func TestCreateMessageExplainsModerationOutcome(t *testing.T) {
	tests := []struct {
		name    string
		status  domain.CommentStatus
		reason  string
		code    int
		message string
	}{
		{
			name:    "approved",
			status:  domain.CommentApproved,
			reason:  "auto moderation",
			code:    http.StatusCreated,
			message: "Comment posted.",
		},
		{
			name:    "pending",
			status:  domain.CommentPending,
			reason:  "manual moderation",
			code:    http.StatusAccepted,
			message: "Comment submitted and waiting for moderation.",
		},
		{
			name:    "rejected rate limit",
			status:  domain.CommentRejected,
			reason:  "rate limit",
			code:    http.StatusForbidden,
			message: "Comment rejected: too many comments were submitted recently. Please try again later.",
		},
		{
			name:    "spam links",
			status:  domain.CommentSpam,
			reason:  "too many links",
			code:    http.StatusForbidden,
			message: "Comment rejected by spam protection: too many links.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusForCreatedComment(tt.status); got != tt.code {
				t.Fatalf("expected status code %d, got %d", tt.code, got)
			}
			if got := createMessage(tt.status, tt.reason); got != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, got)
			}
		})
	}
}

func TestAPICreateCommentRejectsInvalidJSON(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/test-site/pages/%2Fposts%2Fone/comments", bytes.NewBufferString("{"))
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPICreateCommentDoesNotTrustForgedParentOrigin(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"author_name":    "Oleksii",
		"body_markdown":  "Nice post",
		"parent_origin":  "https://allowed.example",
		"author_email":   "oleksii@example.com",
		"author_website": "https://example.com",
	}

	req := newJSONCommentRequest(t, payload)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forged parent_origin to be rejected, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPICreateCommentAcceptsValidEmbedTokenParentOrigin(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"author_name":   "Oleksii",
		"body_markdown": "Nice post",
		"parent_origin": "https://allowed.example",
		"embed_token":   h.signEmbedToken("test-site", "/posts/one", "https://allowed.example"),
	}

	req := newJSONCommentRequest(t, payload)
	req.Header.Set("Origin", "http://comments.localhost")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected valid embed token to allow parent origin, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Status  domain.CommentStatus  `json:"status"`
		Message string                `json:"message"`
		Comment *domain.PublicComment `json:"comment"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != domain.CommentApproved || response.Comment == nil || response.Comment.AuthorName != "Oleksii" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestAPICreateCommentRateLimitMiddlewareChain(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	limiter := middleware.NewRateLimiter(1, time.Hour)
	router := newPublicCommentsRouter(h, limiter)

	payload := map[string]any{
		"author_name":   "Oleksii",
		"body_markdown": "First",
		"parent_origin": "https://allowed.example",
		"embed_token":   h.signEmbedToken("test-site", "/posts/one", "https://allowed.example"),
	}
	firstReq := newJSONCommentRequest(t, payload)
	firstReq.RemoteAddr = "203.0.113.10:12345"
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("expected first request through, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}

	payload["body_markdown"] = "Second"
	secondReq := newJSONCommentRequest(t, payload)
	secondReq.RemoteAddr = "203.0.113.10:54321"
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request to be rate limited, got %d body=%s", secondRec.Code, secondRec.Body.String())
	}
}

func newPublicHandlerTestDeps(t *testing.T) *Handlers {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(context.Background(), database); err != nil {
		t.Fatal(err)
	}
	sites := repository.NewSiteRepository(database)
	pages := repository.NewPageRepository(database)
	comments := repository.NewCommentRepository(database)
	identities := repository.NewIdentityRepository(database)
	moderation := repository.NewModerationRepository(database)
	events := repository.NewEventRepository(database)
	bus := dcevent.NewBus(events)
	bus.Subscribe(dcevent.NewAuditHandler(moderation))

	siteSvc := service.NewSiteService(sites, bus)
	pageSvc := service.NewPageService(pages, bus)
	identitySvc := service.NewIdentityService(identities, "tripcode-secret", bus)
	moderationSvc := service.NewModerationService(moderation, comments, bus)
	markdownSvc := service.NewMarkdownService(dcmarkdown.NewRenderer())
	commentSvc := service.NewCommentService(sites, pages, comments, identitySvc, moderationSvc, markdownSvc, "server-secret", bus)
	site := &domain.Site{
		Key:                   "test-site",
		Name:                  "Test Site",
		AllowedOrigins:        []string{"https://allowed.example"},
		DefaultModerationMode: domain.ModerationAuto,
		DefaultPageState:      domain.PageOpen,
		DefaultTheme:          domain.ThemeAuto,
		MaxCommentLength:      5000,
		AllowReplies:          true,
	}
	if err := siteSvc.Create(context.Background(), site); err != nil {
		t.Fatal(err)
	}

	h := NewHandlers(siteSvc, pageSvc, commentSvc, template.New("test"), "embed-secret")
	return h
}

func newPublicCommentsRouter(h *Handlers, limiter *middleware.RateLimiter) http.Handler {
	router := chi.NewRouter()
	if limiter == nil {
		router.Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APICreateComment)
		return router
	}
	router.With(limiter.Middleware).Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APICreateComment)
	return router
}

func newJSONCommentRequest(t *testing.T, payload map[string]any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/test-site/pages/%2Fposts%2Fone/comments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://comments.localhost")
	req.RemoteAddr = "203.0.113.10:12345"
	return req
}
