package admin

import (
	"html/template"
	"io"
	"path/filepath"
	"testing"
	"time"

	"deadcomments/internal/domain"
)

func TestCommentsQueueTemplateExecutes(t *testing.T) {
	tmpl := mustAdminTemplate(t)
	comment := &domain.Comment{
		ID:                "comment-1",
		AuthorName:        "Oleksii",
		AuthorDisplayName: "Oleksii",
		BodyMarkdown:      "Needs review",
		Status:            domain.CommentPending,
		CreatedAt:         time.Now(),
	}
	data := map[string]any{
		"CSRFToken":   "csrf",
		"CurrentPath": "/admin/comments/pending",
		"Status":      domain.CommentPending,
		"Comments":    []*domain.Comment{comment},
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "comments_queue.html", data); err != nil {
		t.Fatalf("execute comments_queue.html: %v", err)
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "dashboard.html", map[string]any{
		"CSRFToken":     "csrf",
		"CurrentPath":   "/admin",
		"RecentPending": []*domain.Comment{comment},
	}); err != nil {
		t.Fatalf("execute dashboard.html: %v", err)
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "page_detail.html", map[string]any{
		"CSRFToken":   "csrf",
		"CurrentPath": "/admin/pages/1",
		"Page":        &domain.Page{ID: 1, PageKey: "/post", State: domain.PageOpen},
		"Comments":    []*domain.Comment{comment},
	}); err != nil {
		t.Fatalf("execute page_detail.html: %v", err)
	}
}

func mustAdminTemplate(t *testing.T) *template.Template {
	t.Helper()
	funcs := template.FuncMap{
		"safeHTML":     func(s string) template.HTML { return template.HTML(s) },
		"join":         func(v []string) string { return "" },
		"siteSelected": func(selected *int64, id int64) bool { return selected != nil && *selected == id },
		"dict": func(values ...any) map[string]any {
			out := map[string]any{}
			for i := 0; i+1 < len(values); i += 2 {
				key, _ := values[i].(string)
				out[key] = values[i+1]
			}
			return out
		},
	}
	tmpl, err := template.New("").Funcs(funcs).ParseGlob(filepath.Join("..", "..", "templates", "*.html"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.ParseGlob(filepath.Join("..", "..", "templates", "admin", "*.html")); err != nil {
		t.Fatal(err)
	}
	return tmpl
}
