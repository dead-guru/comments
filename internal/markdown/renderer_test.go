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

func TestRendererRejectsCommonXSSVectors(t *testing.T) {
	renderer := NewRenderer()
	tests := []struct {
		name      string
		input     string
		forbidden []string
	}{
		{
			name:      "script tag",
			input:     `<script>alert(1)</script>`,
			forbidden: []string{"<script", "alert(1)"},
		},
		{
			name:      "mixed case script tag",
			input:     `<ScRiPt>alert(1)</ScRiPt>`,
			forbidden: []string{"<script", "alert(1)"},
		},
		{
			name:      "image onerror",
			input:     `<img src=x onerror=alert(1)>`,
			forbidden: []string{"onerror", "alert(1)"},
		},
		{
			name:      "svg onload",
			input:     `<svg onload=alert(1)><circle /></svg>`,
			forbidden: []string{"<svg", "onload", "alert(1)"},
		},
		{
			name:      "javascript link",
			input:     `[click](javascript:alert(1))`,
			forbidden: []string{"href=\"javascript:", "javascript:alert", "alert(1)"},
		},
		{
			name:      "mixed case javascript link",
			input:     `[click](JaVaScRiPt:alert(1))`,
			forbidden: []string{"href=\"javascript:", "javascript:alert", "alert(1)"},
		},
		{
			name:      "data html link",
			input:     `[click](data:text/html,<script>alert(1)</script>)`,
			forbidden: []string{"href=\"data:", "data:text/html", "<script", "alert(1)"},
		},
		{
			name:      "raw html javascript anchor",
			input:     `<a href="javascript:alert(1)">click</a>`,
			forbidden: []string{"href=\"javascript:", "javascript:alert", "alert(1)"},
		},
		{
			name:      "html entity javascript link",
			input:     `[click](&#106;&#97;vascript:alert(1))`,
			forbidden: []string{"href=\"javascript:", "javascript:alert", "alert(1)"},
		},
		{
			name:      "javascript autolink",
			input:     `<javascript:alert(1)>`,
			forbidden: []string{"href=\"javascript:"},
		},
		{
			name:      "vbscript link",
			input:     `[click](vbscript:msgbox(1))`,
			forbidden: []string{"href=\"vbscript:", "vbscript:"},
		},
		{
			name:      "javascript image",
			input:     `![alt](javascript:alert(1))`,
			forbidden: []string{"src=\"javascript:", "javascript:alert", "alert(1)"},
		},
		{
			name:      "math href",
			input:     `<math href="javascript:alert(1)"></math>`,
			forbidden: []string{"<math", "href=\"javascript:", "alert(1)"},
		},
		{
			name:      "iframe srcdoc",
			input:     `<iframe srcdoc="<script>alert(1)</script>"></iframe>`,
			forbidden: []string{"<iframe", "srcdoc", "<script", "alert(1)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := renderer.Render(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			lower := strings.ToLower(html)
			for _, forbidden := range tt.forbidden {
				if strings.Contains(lower, strings.ToLower(forbidden)) {
					t.Fatalf("rendered HTML contains forbidden fragment %q: %s", forbidden, html)
				}
			}
		})
	}
}
