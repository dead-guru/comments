package admin

import (
	"net/http"
	"strconv"

	"deadcomments/internal/domain"
	"deadcomments/internal/repository"
)

func (h *Handlers) Pages(w http.ResponseWriter, r *http.Request) {
	var siteID *int64
	if id := parseIDParam(r, "id"); id > 0 {
		siteID = &id
	}
	page, limit, offset := adminPage(r, "page")
	pages, err := h.pages.ListPaginated(r.Context(), siteID, r.URL.Query().Get("state"), r.URL.Query().Get("q"), limit, offset)
	if err != nil {
		http.Error(w, "failed to load pages", http.StatusInternalServerError)
		return
	}
	pages, hasNext := trimAdminPage(pages)
	h.render(w, r, "admin/pages_list.html", map[string]any{"Pages": pages, "SiteID": siteID, "Filters": r.URL.Query(), "Pagination": newPagination(r, "page", page, hasNext)})
}

func (h *Handlers) PageDetail(w http.ResponseWriter, r *http.Request) {
	page, err := h.pages.ByID(r.Context(), parseIDParam(r, "id"))
	if err != nil || page == nil {
		http.NotFound(w, r)
		return
	}
	commentPage, limit, offset := adminPage(r, "page")
	comments, err := h.comments.AdminListFiltered(r.Context(), repository.CommentListFilter{PageID: &page.ID, Limit: limit, Offset: offset})
	if err != nil {
		http.Error(w, "failed to load comments", http.StatusInternalServerError)
		return
	}
	comments, hasNext := trimAdminPage(comments)
	h.render(w, r, "admin/page_detail.html", map[string]any{"Page": page, "Comments": comments, "Pagination": newPagination(r, "page", commentPage, hasNext)})
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
