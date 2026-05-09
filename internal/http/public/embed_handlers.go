package public

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/service"
)

type Handlers struct {
	sites    *service.SiteService
	pages    *service.PageService
	comments *service.CommentService
	tmpl     *template.Template
}

func NewHandlers(sites *service.SiteService, pages *service.PageService, comments *service.CommentService, tmpl *template.Template) *Handlers {
	return &Handlers{sites: sites, pages: pages, comments: comments, tmpl: tmpl}
}

func (h *Handlers) WidgetJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	http.ServeFile(w, r, "internal/widget/widget.js")
}

func (h *Handlers) EmbedCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	http.ServeFile(w, r, "internal/static/embed.css")
}

func (h *Handlers) EmbedComments(w http.ResponseWriter, r *http.Request) {
	siteKey := strings.TrimSpace(r.URL.Query().Get("site"))
	pageKey := strings.TrimSpace(r.URL.Query().Get("page"))
	theme := normalizeTheme(r.URL.Query().Get("theme"))
	if siteKey == "" || pageKey == "" {
		h.renderEmbedError(w, "Comments are not configured.")
		return
	}
	page, comments, err := h.comments.PublicTree(r.Context(), siteKey, pageKey)
	if err != nil || page == nil {
		h.renderEmbedError(w, "Comments are unavailable.")
		return
	}
	data := map[string]any{
		"SiteKey":      siteKey,
		"PageKey":      pageKey,
		"Page":         page,
		"Comments":     comments,
		"Theme":        theme,
		"ParentOrigin": r.URL.Query().Get("parent_origin"),
		"PageTitle":    r.URL.Query().Get("title"),
		"PageURL":      r.URL.Query().Get("url"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "comments.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handlers) renderEmbedError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "comments.html", map[string]any{
		"Error": msg,
		"Theme": domain.ThemeAuto,
	})
}

func normalizeTheme(raw string) string {
	switch domain.Theme(raw) {
	case domain.ThemeLight, domain.ThemeDark, domain.ThemeMinimal:
		return raw
	default:
		return string(domain.ThemeAuto)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
