package public

import (
	"crypto/rand"
	"encoding/base64"
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
		"CSPNonce":     newCSPNonce(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	setEmbedCSP(w, data["CSPNonce"].(string))
	if err := h.tmpl.ExecuteTemplate(w, "comments.html", data); err != nil {
		http.Error(w, "comments unavailable", http.StatusInternalServerError)
	}
}

func (h *Handlers) renderEmbedError(w http.ResponseWriter, locale string, msg string) {
	nonce := newCSPNonce()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	setEmbedCSP(w, nonce)
	_ = h.tmpl.ExecuteTemplate(w, "comments.html", map[string]any{
		"Error":    msg,
		"Theme":    domain.ThemeAuto,
		"Locale":   locale,
		"T":        i18n.Embed(locale),
		"Sort":     domain.CommentSortOldest,
		"CSPNonce": nonce,
	})
}

func setEmbedCSP(w http.ResponseWriter, nonce string) {
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'nonce-"+nonce+"'; connect-src 'self'; style-src 'self'; img-src 'self' https://www.gravatar.com https://secure.gravatar.com data:; base-uri 'none'; form-action 'none'")
}

func newCSPNonce() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ""
	}
	return base64.RawStdEncoding.EncodeToString(raw[:])
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
