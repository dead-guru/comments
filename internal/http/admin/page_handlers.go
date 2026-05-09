package admin

import (
	"net/http"
	"strconv"

	"deadcomments/internal/domain"
)

func (h *Handlers) Pages(w http.ResponseWriter, r *http.Request) {
	var siteID *int64
	if id := parseIDParam(r, "id"); id > 0 {
		siteID = &id
	}
	pages, _ := h.pages.List(r.Context(), siteID, r.URL.Query().Get("state"), r.URL.Query().Get("q"))
	h.render(w, r, "admin/pages_list.html", map[string]any{"Pages": pages, "SiteID": siteID})
}

func (h *Handlers) PageDetail(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages.ByID(r.Context(), parseIDParam(r, "id"))
	if err != nil || page == nil {
		http.NotFound(w, r)
		return
	}
	comments, _ := h.comments.AdminList(r.Context(), "", "", nil, &page.ID, 200)
	h.render(w, r, "admin/page_detail.html", map[string]any{"Page": page, "Comments": comments})
}

func (h *Handlers) PageState(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	state := domain.PageState(r.FormValue("state"))
	switch state {
	case domain.PageOpen, domain.PageLocked, domain.PageHidden, domain.PageArchived:
		_ = h.pages.SetState(r.Context(), id, state)
	}
	http.Redirect(w, r, "/admin/pages/"+strconv.FormatInt(id, 10), http.StatusFound)
}
