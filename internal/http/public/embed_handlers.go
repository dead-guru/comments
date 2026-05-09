package public

import (
	"html/template"
	"net/http"
	"os"
	"strings"

	"deadcomments/internal/domain"
	"deadcomments/internal/i18n"
	"deadcomments/internal/service"
)

type Handlers struct {
	sites       *service.SiteService
	pages       *service.PageService
	comments    *service.CommentService
	markdown    *service.MarkdownService
	tmpl        *template.Template
	embedSecret string
}

func NewHandlers(sites *service.SiteService, pages *service.PageService, comments *service.CommentService, markdown *service.MarkdownService, tmpl *template.Template, embedSecret string) *Handlers {
	return &Handlers{sites: sites, pages: pages, comments: comments, markdown: markdown, tmpl: tmpl, embedSecret: embedSecret}
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
	sort := service.NormalizeCommentSort(r.URL.Query().Get("sort"))
	locale := i18n.Normalize(r.URL.Query().Get("locale"), r.Header.Get("Accept-Language"))
	if siteKey == "" || pageKey == "" {
		h.renderEmbedError(w, locale, i18n.Text(locale, "comments_not_configured"))
		return
	}
	site, err := h.sites.ByKey(r.Context(), siteKey)
	if err != nil || site == nil {
		h.renderEmbedError(w, locale, i18n.Text(locale, "comments_unavailable"))
		return
	}
	parentOrigin := originFromRequest(r)
	if !h.sites.OriginAllowed(site, parentOrigin) {
		h.renderEmbedError(w, locale, i18n.Text(locale, "comments_unavailable_origin"))
		return
	}
	page, comments, err := h.comments.PublicTree(r.Context(), siteKey, pageKey, sort)
	if err != nil || page == nil {
		h.renderEmbedError(w, locale, i18n.Text(locale, "comments_unavailable"))
		return
	}
	data := map[string]any{
		"SiteKey":      siteKey,
		"PageKey":      pageKey,
		"Page":         page,
		"Comments":     comments,
		"Theme":        theme,
		"Locale":       locale,
		"T":            i18n.Embed(locale),
		"Sort":         sort,
		"CanReply":     page.State == domain.PageOpen,
		"ParentOrigin": parentOrigin,
		"EmbedToken":   h.signEmbedToken(siteKey, pageKey, parentOrigin),
		"MaxLength":    site.MaxCommentLength,
		"PageTitle":    r.URL.Query().Get("title"),
		"PageURL":      r.URL.Query().Get("url"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "comments.html", data); err != nil {
		http.Error(w, "comments unavailable", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderEmbedError(w http.ResponseWriter, locale string, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = h.tmpl.ExecuteTemplate(w, "comments.html", map[string]any{
		"Error":  msg,
		"Theme":  domain.ThemeAuto,
		"Locale": locale,
		"T":      i18n.Embed(locale),
		"Sort":   domain.CommentSortOldest,
	})
}

func normalizeTheme(raw string) string {
	switch domain.Theme(raw) {
	case domain.ThemeLight, domain.ThemeDark, domain.ThemeMinimal:
		return raw
	case "inherit":
		return raw
	default:
		return string(domain.ThemeAuto)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
