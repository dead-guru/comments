package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmbedTokenBindsSitePageAndOrigin(t *testing.T) {
	h := &Handlers{embedSecret: "test-secret"}
	token := h.signEmbedToken("site", "/post", "https://blog.example")

	if token == "" {
		t.Fatal("expected token")
	}
	if !h.validEmbedToken("site", "/post", "https://blog.example", token) {
		t.Fatal("expected token to validate for normalized origin")
	}
	if h.validEmbedToken("site", "/other", "https://blog.example", token) {
		t.Fatal("expected token to reject a different page")
	}
	if h.validEmbedToken("site", "/post", "https://evil.example", token) {
		t.Fatal("expected token to reject a different origin")
	}
}

func TestEmbedTokenNormalizesOriginURL(t *testing.T) {
	h := &Handlers{embedSecret: "test-secret"}
	token := h.signEmbedToken("site", "/post", "https://blog.example/articles/one")

	if !h.validEmbedToken("site", "/post", "https://blog.example", token) {
		t.Fatal("expected token signed with a full URL to validate against its origin")
	}
}

func TestEmbedCSPAllowsOnlyNonceScript(t *testing.T) {
	rec := httptest.NewRecorder()
	nonce := "nonce-value"
	setEmbedCSP(rec, nonce)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'nonce-"+nonce+"'") {
		t.Fatalf("expected nonce script-src, got %q", csp)
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Fatalf("embed CSP must not allow unsafe-inline, got %q", csp)
	}
}

func TestEmbedCSPAllowsHTTPSMarkdownImages(t *testing.T) {
	rec := httptest.NewRecorder()
	setEmbedCSP(rec, "nonce-value")

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "img-src 'self' https: data:") {
		t.Fatalf("expected HTTPS images to be allowed, got %q", csp)
	}
}

func TestOriginFromRequestUsesTrustedHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/embed/comments", nil)
	req.Header.Set("Origin", "https://blog.example")
	req.Header.Set("Referer", "https://evil.example/post")

	if got := originFromRequest(req); got != "https://blog.example" {
		t.Fatalf("expected origin header to win, got %q", got)
	}

	req.Header.Del("Origin")
	if got := originFromRequest(req); got != "https://evil.example" {
		t.Fatalf("expected referer fallback origin, got %q", got)
	}
}

func TestTrustedCommentOriginRequiresValidEmbedTokenForParentOrigin(t *testing.T) {
	h := &Handlers{embedSecret: "test-secret"}
	req, _ := http.NewRequest(http.MethodPost, "/api/comments", nil)
	req.Header.Set("Origin", "https://comments.example")

	if got := h.trustedCommentOrigin(req, "site", "/post", "https://blog.example", "bad-token"); got != "https://comments.example" {
		t.Fatalf("expected invalid body origin to be ignored, got %q", got)
	}

	token := h.signEmbedToken("site", "/post", "https://blog.example")
	if got := h.trustedCommentOrigin(req, "site", "/post", "https://blog.example", token); got != "https://blog.example" {
		t.Fatalf("expected valid token parent origin, got %q", got)
	}
}

func TestNormalizeThemeAllowsInheritedHostTheme(t *testing.T) {
	if got := normalizeTheme("inherit"); got != "inherit" {
		t.Fatalf("expected inherit theme, got %q", got)
	}
	if got := normalizeTheme("unknown"); got != "auto" {
		t.Fatalf("expected unknown theme to fall back to auto, got %q", got)
	}
}
