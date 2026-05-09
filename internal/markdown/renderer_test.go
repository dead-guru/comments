package markdown

import (
	"strings"
	"testing"
)

func TestRendererSanitizesUnsafeHTML(t *testing.T) {
	renderer := NewRenderer()
	html, err := renderer.Render("hello **world**\n\n<script>alert(1)</script>\n\n[site](https://example.com)")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToLower(html), "<script") || strings.Contains(html, "alert(1)") {
		t.Fatalf("unsafe script leaked into html: %s", html)
	}
	if !strings.Contains(html, "<strong>world</strong>") {
		t.Fatalf("expected markdown formatting, got: %s", html)
	}
	if !strings.Contains(html, "rel=\"nofollow") {
		t.Fatalf("expected sanitized link rel attributes, got: %s", html)
	}
}
