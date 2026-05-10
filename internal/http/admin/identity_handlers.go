package admin

import (
	"net/http"
	"strconv"

	"deadcomments/internal/domain"
	"deadcomments/internal/http/middleware"
)

func (h *Handlers) Identities(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := adminPage(r, "page")
	identities, err := h.identities.ListPaginated(r.Context(), limit, offset)
	if err != nil {
		http.Error(w, "failed to load identities", http.StatusInternalServerError)
		return
	}
	identities, hasNext := trimAdminPage(identities)
	h.render(w, r, "admin/identities_list.html", map[string]any{"Identities": identities, "Pagination": newPagination(r, "page", page, hasNext)})
}

func (h *Handlers) NewIdentity(w http.ResponseWriter, r *http.Request) {
	sites, _ := h.sites.List(r.Context())
	h.render(w, r, "admin/identity_form.html", map[string]any{"Identity": &domain.Identity{BadgeType: domain.BadgeVerified}, "Sites": sites, "Action": "/admin/identities", "Mode": "create"})
}

func (h *Handlers) CreateIdentity(w http.ResponseWriter, r *http.Request) {
	admin := middleware.AdminFromContext(r.Context())
	var adminID *int64
	if admin != nil {
		adminID = &admin.ID
	}
	identity, err := h.identities.Create(r.Context(), domain.IdentityCreateInput{
		SiteID:           identitySiteID(r),
		DisplayName:      r.FormValue("display_name"),
		Secret:           r.FormValue("secret"),
		PublicTripcode:   r.FormValue("public_tripcode"),
		BadgeType:        domain.BadgeType(r.FormValue("badge_type")),
		BadgeLabel:       r.FormValue("badge_label"),
		CreatedByAdminID: adminID,
	})
	if err != nil {
		sites, _ := h.sites.List(r.Context())
		h.render(w, r, "admin/identity_form.html", map[string]any{"Identity": identityFromForm(r), "Sites": sites, "Action": "/admin/identities", "Mode": "create", "Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/identities/"+strconv.FormatInt(identity.ID, 10), http.StatusFound)
}

func (h *Handlers) IdentityDetail(w http.ResponseWriter, r *http.Request) {
	identity, err := h.identities.ByID(r.Context(), parseIDParam(r, "id"))
	if err != nil || identity == nil {
		http.NotFound(w, r)
		return
	}
	sites, _ := h.sites.List(r.Context())
	commentCount, _ := h.comments.CountByIdentity(r.Context(), identity.ID)
	h.render(w, r, "admin/identity_detail.html", map[string]any{"Identity": identity, "Sites": sites, "CommentCount": commentCount, "Action": "/admin/identities/" + strconv.FormatInt(identity.ID, 10), "Mode": "edit"})
}

func (h *Handlers) UpdateIdentity(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	identity, err := h.identities.Update(r.Context(), domain.IdentityUpdateInput{
		ID:             id,
		SiteID:         identitySiteID(r),
		DisplayName:    r.FormValue("display_name"),
		PublicTripcode: r.FormValue("public_tripcode"),
		BadgeType:      domain.BadgeType(r.FormValue("badge_type")),
		BadgeLabel:     r.FormValue("badge_label"),
	})
	if err != nil {
		sites, _ := h.sites.List(r.Context())
		formIdentity := identityFromForm(r)
		formIdentity.ID = id
		h.render(w, r, "admin/identity_detail.html", map[string]any{"Identity": formIdentity, "Sites": sites, "Action": r.URL.Path, "Mode": "edit", "Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/identities/"+strconv.FormatInt(identity.ID, 10), http.StatusFound)
}

func (h *Handlers) ResetIdentitySecret(w http.ResponseWriter, r *http.Request) {
	id := parseIDParam(r, "id")
	if err := h.identities.ResetSecret(r.Context(), id, r.FormValue("secret")); err != nil {
		identity, _ := h.identities.ByID(r.Context(), id)
		sites, _ := h.sites.List(r.Context())
		h.render(w, r, "admin/identity_detail.html", map[string]any{"Identity": identity, "Sites": sites, "Action": "/admin/identities/" + strconv.FormatInt(id, 10), "Mode": "edit", "Error": err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/identities/"+strconv.FormatInt(id, 10), http.StatusFound)
}

func (h *Handlers) DeleteIdentity(w http.ResponseWriter, r *http.Request) {
	_ = h.identities.Delete(r.Context(), parseIDParam(r, "id"))
	http.Redirect(w, r, "/admin/identities", http.StatusFound)
}

func identitySiteID(r *http.Request) *int64 {
	raw := r.FormValue("site_id")
	if raw == "" || raw == "global" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func identityFromForm(r *http.Request) *domain.Identity {
	identity := &domain.Identity{
		SiteID:         identitySiteID(r),
		DisplayName:    r.FormValue("display_name"),
		PublicTripcode: r.FormValue("public_tripcode"),
		BadgeType:      domain.BadgeType(r.FormValue("badge_type")),
	}
	if identity.BadgeType == "" {
		identity.BadgeType = domain.BadgeVerified
	}
	if label := r.FormValue("badge_label"); label != "" {
		identity.BadgeLabel = &label
	}
	return identity
}
