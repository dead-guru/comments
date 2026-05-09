package admin

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	dcauth "deadcomments/internal/auth"
	"deadcomments/internal/domain"
	"deadcomments/internal/http/middleware"
	"deadcomments/internal/service"
)

type Handlers struct {
	sites      *service.SiteService
	pages      *service.PageService
	comments   *service.CommentService
	moderation *service.ModerationService
	identities *service.IdentityService
	events     *service.EventService
	auth       *service.AuthService
	tmpl       *template.Template
	csrf       *middleware.CSRF
	secure     bool
}

func NewHandlers(sites *service.SiteService, pages *service.PageService, comments *service.CommentService, moderation *service.ModerationService, identities *service.IdentityService, events *service.EventService, auth *service.AuthService, tmpl *template.Template, csrf *middleware.CSRF, secure bool) *Handlers {
	return &Handlers{sites: sites, pages: pages, comments: comments, moderation: moderation, identities: identities, events: events, auth: auth, tmpl: tmpl, csrf: csrf, secure: secure}
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "admin/login.html", map[string]any{})
}

func (h *Handlers) AdminCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	http.ServeFile(w, r, "internal/static/admin.css")
}

func (h *Handlers) AdminJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	http.ServeFile(w, r, "internal/static/admin.js")
}

func (h *Handlers) GitHubStart(w http.ResponseWriter, r *http.Request) {
	state, _ := dcauth.NewToken()
	http.SetCookie(w, &http.Cookie{Name: "dc_oauth_state", Value: state, Path: "/auth/github", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.Redirect(w, r, h.auth.GitHubAuthURL(state), http.StatusFound)
}

func (h *Handlers) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("dc_oauth_state")
	if err != nil || !dcauth.ConstantTimeEqual(stateCookie.Value, r.URL.Query().Get("state")) {
		http.Error(w, "invalid oauth state", http.StatusForbidden)
		return
	}
	_, token, err := h.auth.CompleteGitHubLogin(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: middleware.SessionCookieName, Value: token, Path: "/admin", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(30 * 24 * time.Hour)})
	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(middleware.SessionCookieName); err == nil {
		_ = h.auth.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: middleware.SessionCookieName, Value: "", Path: "/admin", MaxAge: -1})
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	totalSites, _ := h.sites.Count(r.Context())
	totalPages, _ := h.pages.Count(r.Context())
	pending, _ := h.comments.CountByStatus(r.Context(), domain.CommentPending)
	approvedToday, _ := h.comments.CountTodayByStatus(r.Context(), domain.CommentApproved)
	spamToday, _ := h.comments.CountTodayByStatus(r.Context(), domain.CommentSpam)
	recentPending, _ := h.comments.AdminList(r.Context(), string(domain.CommentPending), "", nil, nil, 10)
	recentPages, _ := h.pages.List(r.Context(), nil, "", "")
	recentEvents, _ := h.events.Recent(r.Context(), 10)
	h.render(w, r, "admin/dashboard.html", map[string]any{
		"TotalSites": totalSites, "TotalPages": totalPages, "Pending": pending, "ApprovedToday": approvedToday, "SpamToday": spamToday,
		"RecentPending": recentPending, "RecentPages": limitPages(recentPages, 10), "RecentEvents": recentEvents,
	})
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["CSRFToken"] = h.csrf.Token(w, r)
	data["Admin"] = middleware.AdminFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	name = strings.TrimPrefix(name, "admin/")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseIDParam(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	return id
}

func splitLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, strings.TrimRight(line, "/"))
		}
	}
	return out
}

func limitPages(pages []*domain.Page, n int) []*domain.Page {
	if len(pages) <= n {
		return pages
	}
	return pages[:n]
}
