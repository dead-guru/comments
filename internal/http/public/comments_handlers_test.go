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
			if got := createMessage("en", tt.status, tt.reason); got != tt.message {
				t.Fatalf("expected message %q, got %q", tt.message, got)
			}
		})
	}
}

func TestCreateMessageLocalizesModerationOutcome(t *testing.T) {
	got := createMessage("uk", domain.CommentPending, "manual moderation")
	want := "Коментар надіслано й очікує модерації."
	if got != want {
		t.Fatalf("expected localized message %q, got %q", want, got)
	}
}

func TestCreateMessageIncludesRateLimitRetry(t *testing.T) {
	got := createMessageWithRetry("en", domain.CommentRejected, "rate limit", 90*time.Second)
	want := "Comment rejected: too many comments were submitted recently. Try again in about 2 minutes."
	if got != want {
		t.Fatalf("expected retry message %q, got %q", want, got)
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

func TestAPIListCommentsReturnsRequestedSort(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sites/test-site/pages/%2Fposts%2Fone/comments?sort=newest", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		Sort          domain.CommentSort `json:"sort"`
		ApprovedCount int                `json:"approved_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Sort != domain.CommentSortNewest {
		t.Fatalf("expected newest sort, got %q", response.Sort)
	}
	if response.ApprovedCount != 0 {
		t.Fatalf("expected approved count 0, got %d", response.ApprovedCount)
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

func TestAPICreateCommentAllowsConfiguredCrossOriginRequest(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"author_name":   "Oleksii",
		"body_markdown": "Direct public API comment",
	}

	optionsReq := httptest.NewRequest(http.MethodOptions, "/api/v1/sites/test-site/pages/%2Fposts%2Fone/comments", nil)
	optionsReq.Header.Set("Origin", "https://allowed.example")
	optionsRec := httptest.NewRecorder()
	router.ServeHTTP(optionsRec, optionsReq)
	if optionsRec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight success, got %d body=%s", optionsRec.Code, optionsRec.Body.String())
	}
	if optionsRec.Header().Get("Access-Control-Allow-Origin") != "https://allowed.example" {
		t.Fatalf("expected CORS allow-origin header on preflight, got %q", optionsRec.Header().Get("Access-Control-Allow-Origin"))
	}

	req := newJSONCommentRequest(t, payload)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected direct cross-origin create success, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://allowed.example" {
		t.Fatalf("expected CORS allow-origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestAPIPreviewCommentRendersSanitizedMarkdown(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"body_markdown": "**ok**\n\n<script>alert(1)</script>",
		"parent_origin": "https://allowed.example",
		"embed_token":   h.signEmbedToken("test-site", "/posts/one", "https://allowed.example"),
	}

	req := newJSONPreviewRequest(t, payload)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected preview response, got %d body=%s", rec.Code, rec.Body.String())
	}
	var response struct {
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(response.BodyHTML), []byte("<strong>ok</strong>")) {
		t.Fatalf("expected markdown rendering, got %s", response.BodyHTML)
	}
	if bytes.Contains(bytes.ToLower([]byte(response.BodyHTML)), []byte("<script")) || bytes.Contains([]byte(response.BodyHTML), []byte("alert(1)")) {
		t.Fatalf("expected sanitized preview, got %s", response.BodyHTML)
	}
}

func TestAPIPreviewCommentDoesNotTrustForgedParentOrigin(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"body_markdown": "Preview",
		"parent_origin": "https://allowed.example",
	}

	req := newJSONPreviewRequest(t, payload)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forged parent_origin to be rejected, got %d body=%s", rec.Code, rec.Body.String())
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

func TestAnnotationAPIRequiresAllowedOrigin(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"author_name":   "Oleksii",
		"body_markdown": "Inline note",
		"selector":      "#article",
		"selected_text": "selected text",
	}
	req := newJSONAnnotationRequest(t, payload)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAnnotationAPICreatesPublicAnnotation(t *testing.T) {
	h := newPublicHandlerTestDeps(t)
	router := newPublicCommentsRouter(h, nil)
	payload := map[string]any{
		"author_name":      "Oleksii##annotation-secret",
		"body_markdown":    "**Inline** note",
		"selector":         "#article",
		"selected_text":    "selected text",
		"selection_prefix": "before",
		"selection_suffix": "after",
		"text_start":       10,
		"text_end":         23,
	}
	req := newJSONAnnotationRequest(t, payload)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected created, got %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://allowed.example" {
		t.Fatalf("expected CORS allow-origin header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	var created struct {
		Annotation domain.PublicAnnotation `json:"annotation"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Annotation.ID == "" || created.Annotation.Comment == nil {
		t.Fatalf("expected annotation with public comment: %#v", created.Annotation)
	}
	if created.Annotation.Comment.TripcodeKind != domain.TripcodeAnonymous {
		t.Fatalf("expected anonymous tripcode, got %s", created.Annotation.Comment.TripcodeKind)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/test-site/pages/%2Fposts%2Finline/annotations", nil)
	listReq.Header.Set("Origin", "https://allowed.example")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list ok, got %d body=%s", listRec.Code, listRec.Body.String())
	}
	var listed struct {
		Annotations []domain.PublicAnnotation `json:"annotations"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Annotations) != 1 || listed.Annotations[0].SelectedText != "selected text" {
		t.Fatalf("expected one listed annotation, got %#v", listed.Annotations)
	}

	commentsReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/test-site/pages/%2Fposts%2Finline/comments", nil)
	commentsRec := httptest.NewRecorder()
	router.ServeHTTP(commentsRec, commentsReq)
	if commentsRec.Code != http.StatusOK {
		t.Fatalf("expected comments list ok, got %d body=%s", commentsRec.Code, commentsRec.Body.String())
	}
	var commentsResp struct {
		Comments []domain.PublicComment `json:"comments"`
	}
	if err := json.Unmarshal(commentsRec.Body.Bytes(), &commentsResp); err != nil {
		t.Fatal(err)
	}
	if len(commentsResp.Comments) != 1 || commentsResp.Comments[0].Annotation == nil {
		t.Fatalf("expected annotation metadata on public comment, got %#v", commentsResp.Comments)
	}
	if commentsResp.Comments[0].Annotation.ID != created.Annotation.ID {
		t.Fatalf("expected annotation id %q on comment, got %#v", created.Annotation.ID, commentsResp.Comments[0].Annotation)
	}

	regularOnlyReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites/test-site/pages/%2Fposts%2Finline/comments?include_annotations=false", nil)
	regularOnlyRec := httptest.NewRecorder()
	router.ServeHTTP(regularOnlyRec, regularOnlyReq)
	if regularOnlyRec.Code != http.StatusOK {
		t.Fatalf("expected regular-only comments ok, got %d body=%s", regularOnlyRec.Code, regularOnlyRec.Body.String())
	}
	var regularOnlyResp struct {
		ApprovedCount int                    `json:"approved_count"`
		Comments      []domain.PublicComment `json:"comments"`
	}
	if err := json.Unmarshal(regularOnlyRec.Body.Bytes(), &regularOnlyResp); err != nil {
		t.Fatal(err)
	}
	if regularOnlyResp.ApprovedCount != 0 || len(regularOnlyResp.Comments) != 0 {
		t.Fatalf("expected annotations excluded from comments widget response, got %#v", regularOnlyResp)
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
	annotations := repository.NewAnnotationRepository(database)
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
	annotationSvc := service.NewAnnotationService(sites, annotations, commentSvc, bus)
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

	h := NewHandlers(siteSvc, pageSvc, commentSvc, annotationSvc, markdownSvc, template.New("test"), "embed-secret")
	return h
}

func newPublicCommentsRouter(h *Handlers, limiter *middleware.RateLimiter) http.Handler {
	router := chi.NewRouter()
	router.Options("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APICommentsOptions)
	router.Options("/api/v1/sites/{site_key}/pages/{page_key:.*}/preview", h.APICommentsOptions)
	router.Get("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APIListComments)
	router.Get("/api/v1/sites/{site_key}/pages/{page_key:.*}/annotations", h.APIListAnnotations)
	router.Options("/api/v1/sites/{site_key}/pages/{page_key:.*}/annotations", h.APIAnnotationsOptions)
	if limiter == nil {
		router.Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APICreateComment)
		router.Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/preview", h.APIPreviewComment)
		router.Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/annotations", h.APICreateAnnotation)
		return router
	}
	router.With(limiter.Middleware).Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/comments", h.APICreateComment)
	router.With(limiter.Middleware).Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/annotations", h.APICreateAnnotation)
	router.Post("/api/v1/sites/{site_key}/pages/{page_key:.*}/preview", h.APIPreviewComment)
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

func newJSONPreviewRequest(t *testing.T, payload map[string]any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/test-site/pages/%2Fposts%2Fone/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://comments.localhost")
	req.RemoteAddr = "203.0.113.10:12345"
	return req
}

func newJSONAnnotationRequest(t *testing.T, payload map[string]any) *http.Request {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sites/test-site/pages/%2Fposts%2Finline/annotations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://comments.localhost")
	req.RemoteAddr = "203.0.113.10:12345"
	return req
}
