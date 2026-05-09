package public

import (
	"net/http"
	"testing"
)

func TestEmbedTokenBindsSitePageAndOrigin(t *testing.T) {
	h := &Handlers{embedSecret: "test-secret"}
	token := h.signEmbedToken("site", "/post", "https://blog.example/articles/one")

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
