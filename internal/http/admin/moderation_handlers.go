package admin

import (
	"net/http"
	"strconv"

	"deadcomments/internal/domain"
	"deadcomments/internal/http/middleware"
)

func (h *Handlers) Bans(w http.ResponseWriter, r *http.Request) {
	ipPage, ipLimit, ipOffset := adminPage(r, "ip_page")
	wordPage, wordLimit, wordOffset := adminPage(r, "word_page")
	ipBans, err := h.moderation.ListIPBansPaginated(r.Context(), ipLimit, ipOffset)
	if err != nil {
		http.Error(w, "failed to load IP bans", http.StatusInternalServerError)
		return
	}
	wordBans, err := h.moderation.ListWordBansPaginated(r.Context(), wordLimit, wordOffset)
	if err != nil {
		http.Error(w, "failed to load word bans", http.StatusInternalServerError)
		return
	}
	ipBans, hasNextIP := trimAdminPage(ipBans)
	wordBans, hasNextWord := trimAdminPage(wordBans)
	sites, err := h.sites.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load sites", http.StatusInternalServerError)
		return
	}
	h.render(w, r, "admin/bans.html", map[string]any{
		"IPBans":         ipBans,
		"WordBans":       wordBans,
		"Sites":          sites,
		"IPPagination":   newPagination(r, "ip_page", ipPage, hasNextIP),
		"WordPagination": newPagination(r, "word_page", wordPage, hasNextWord),
	})
}

func (h *Handlers) CreateIPBan(w http.ResponseWriter, r *http.Request) {
	admin := middleware.AdminFromContext(r.Context())
	var adminID *int64
	if admin != nil {
		adminID = &admin.ID
	}
	var siteID *int64
	if raw := r.FormValue("site_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			siteID = &id
		}
	}
	ipHash := r.FormValue("ip_hash")
	if ipHash != "" {
		reason := r.FormValue("reason")
		ban := &domain.IPBan{SiteID: siteID, IPHash: ipHash, CreatedByAdminID: adminID}
		if reason != "" {
			ban.Reason = &reason
		}
		_ = h.moderation.AddIPBan(r.Context(), ban)
	}
	http.Redirect(w, r, "/admin/bans", http.StatusFound)
}

func (h *Handlers) CreateWordBan(w http.ResponseWriter, r *http.Request) {
	var siteID *int64
	if raw := r.FormValue("site_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			siteID = &id
		}
	}
	action := domain.WordBanAction(r.FormValue("action"))
	if action == "" {
		action = domain.WordBanPending
	}
	if pattern := r.FormValue("pattern"); pattern != "" {
		_ = h.moderation.AddWordBan(r.Context(), &domain.WordBan{SiteID: siteID, Pattern: pattern, Action: action})
	}
	http.Redirect(w, r, "/admin/bans", http.StatusFound)
}

func (h *Handlers) DeleteBan(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("kind") == "word" {
		_ = h.moderation.DeleteWordBan(r.Context(), parseIDParam(r, "id"))
	} else {
		_ = h.moderation.DeleteBan(r.Context(), parseIDParam(r, "id"))
	}
	http.Redirect(w, r, "/admin/bans", http.StatusFound)
}
