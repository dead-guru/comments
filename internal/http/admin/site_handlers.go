package admin

import (
	"net/http"
	"strconv"

	"deadcomments/internal/domain"
)

func (h *Handlers) Sites(w http.ResponseWriter, r *http.Request) {
	sites, _ := h.sites.List(r.Context())
	h.render(w, r, "admin/sites_list.html", map[string]any{"Sites": sites})
}

func (h *Handlers) NewSite(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "admin/site_form.html", map[string]any{"Site": defaultSite(), "Action": "/admin/sites"})
}

func (h *Handlers) CreateSite(w http.ResponseWriter, r *http.Request) {
	site := siteFromForm(r)
	if err := h.sites.Create(r.Context(), site); err != nil {
		h.render(w, r, "admin/site_form.html", map[string]any{"Site": site, "Action": "/admin/sites", "Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/sites/"+strconv.FormatInt(site.ID, 10), http.StatusFound)
}

func (h *Handlers) SiteSettings(w http.ResponseWriter, r *http.Request) {
	site, err := h.sites.ByID(r.Context(), parseIDParam(r, "id"))
	if err != nil || site == nil {
		http.NotFound(w, r)
		return
	}
	h.render(w, r, "admin/site_settings.html", map[string]any{"Site": site, "Action": "/admin/sites/" + strconv.FormatInt(site.ID, 10) + "/settings"})
}

func (h *Handlers) UpdateSite(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	site := siteFromForm(r)
	site.ID = id
	if err := h.sites.Update(r.Context(), site); err != nil {
		h.render(w, r, "admin/site_settings.html", map[string]any{"Site": site, "Action": r.URL.Path, "Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/sites/"+strconv.FormatInt(id, 10), http.StatusFound)
}

func defaultSite() *domain.Site {
	return &domain.Site{
		DefaultModerationMode: domain.ModerationManual,
		DefaultPageState:      domain.PageOpen,
		DefaultTheme:          domain.ThemeAuto,
		MaxCommentLength:      5000,
		AllowReplies:          true,
	}
}

func siteFromForm(r *http.Request) *domain.Site {
	_ = r.ParseForm()
	maxLen, _ := strconv.Atoi(r.FormValue("max_comment_length"))
	if maxLen <= 0 {
		maxLen = 5000
	}
	return &domain.Site{
		Key:                   r.FormValue("key"),
		Name:                  r.FormValue("name"),
		AllowedOrigins:        splitLines(r.FormValue("allowed_origins")),
		DefaultModerationMode: domain.ModerationMode(r.FormValue("default_moderation_mode")),
		DefaultPageState:      domain.PageState(r.FormValue("default_page_state")),
		DefaultTheme:          domain.Theme(r.FormValue("default_theme")),
		MaxCommentLength:      maxLen,
		AllowReplies:          r.FormValue("allow_replies") == "1",
	}
}
