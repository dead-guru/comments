package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
	"deadcomments/internal/http/middleware"
	"deadcomments/internal/repository"
)

func (h *Handlers) Comments(w http.ResponseWriter, r *http.Request) {
	filter := commentListFilterFromRequest(r)
	comments, _ := h.comments.AdminListFiltered(r.Context(), filter)
	h.render(w, r, "admin/comments_queue.html", map[string]any{
		"Comments": comments,
		"Status":   filter.Status,
		"Filters":  r.URL.Query(),
	})
}

func (h *Handlers) PendingComments(w http.ResponseWriter, r *http.Request) {
	comments, _ := h.comments.AdminList(r.Context(), string(domain.CommentPending), "", nil, nil, 200)
	h.render(w, r, "admin/comments_queue.html", map[string]any{"Comments": comments, "Status": domain.CommentPending})
}

func (h *Handlers) CommentDetail(w http.ResponseWriter, r *http.Request) {
	comment, err := h.comments.ByID(r.Context(), chiID(r))
	if err != nil || comment == nil {
		http.NotFound(w, r)
		return
	}
	events, _ := h.moderation.EventsForComment(r.Context(), comment.ID)
	h.render(w, r, "admin/comment_detail.html", map[string]any{"Comment": comment, "Events": events})
}

func (h *Handlers) ApproveComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentStatus(w, r, domain.CommentApproved)
}

func (h *Handlers) RejectComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentStatus(w, r, domain.CommentRejected)
}

func (h *Handlers) SpamComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentStatus(w, r, domain.CommentSpam)
}

func (h *Handlers) DeleteComment(w http.ResponseWriter, r *http.Request) {
	h.setCommentStatus(w, r, domain.CommentDeleted)
}

func (h *Handlers) BulkComments(w http.ResponseWriter, r *http.Request) {
	status, ok := statusFromModerationAction(r.FormValue("action"))
	if !ok {
		redirectAdmin(w, r, "/admin/comments")
		return
	}
	for _, id := range r.Form["comment_id"] {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		_ = h.comments.SetStatus(r.Context(), id, status)
	}
	redirectAdminFlash(w, r, "/admin/comments", moderationFlash(status))
}

func commentListFilterFromRequest(r *http.Request) repository.CommentListFilter {
	q := r.URL.Query()
	filter := repository.CommentListFilter{
		Status:        strings.TrimSpace(q.Get("status")),
		Search:        strings.TrimSpace(q.Get("q")),
		IPHash:        strings.TrimSpace(q.Get("ip_hash")),
		UserAgentHash: strings.TrimSpace(q.Get("ua_hash")),
		Limit:         200,
	}
	if raw := strings.TrimSpace(q.Get("site_id")); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.SiteID = &id
		}
	}
	if raw := strings.TrimSpace(q.Get("page_id")); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.PageID = &id
		}
	}
	return filter
}

func (h *Handlers) EditComment(w http.ResponseWriter, r *http.Request) {
	id := chiID(r)
	_ = h.comments.Edit(r.Context(), id, r.FormValue("body_markdown"))
	http.Redirect(w, r, "/admin/comments/"+id, http.StatusFound)
}

func (h *Handlers) BanIP(w http.ResponseWriter, r *http.Request) {
	admin := middleware.AdminFromContext(r.Context())
	var adminID *int64
	if admin != nil {
		adminID = &admin.ID
	}
	_ = h.comments.BanIPAndSpam(r.Context(), chiID(r), adminID, r.FormValue("reason"))
	http.Redirect(w, r, "/admin/comments/"+chiID(r), http.StatusFound)
}

func (h *Handlers) setCommentStatus(w http.ResponseWriter, r *http.Request, status domain.CommentStatus) {
	id := chiID(r)
	_ = h.comments.SetStatus(r.Context(), id, status)
	redirectAdminFlash(w, r, "/admin/comments/"+id, moderationFlash(status))
}

func chiID(r *http.Request) string {
	return chi.URLParam(r, "id")
}

func statusFromModerationAction(action string) (domain.CommentStatus, bool) {
	switch action {
	case "approve", "restore":
		return domain.CommentApproved, true
	case "reject":
		return domain.CommentRejected, true
	case "spam":
		return domain.CommentSpam, true
	case "delete":
		return domain.CommentDeleted, true
	default:
		return "", false
	}
}

func redirectAdmin(w http.ResponseWriter, r *http.Request, fallback string) {
	redirectAdminFlash(w, r, fallback, "")
}

func redirectAdminFlash(w http.ResponseWriter, r *http.Request, fallback string, flash string) {
	target := safeAdminRedirect(r.FormValue("redirect_to"))
	if target == "" {
		target = fallback
	}
	if flash != "" {
		target = appendQuery(target, "flash", flash)
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func safeAdminRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || (u.Path != "/admin" && !strings.HasPrefix(u.Path, "/admin/")) {
		return ""
	}
	return u.RequestURI()
}

func appendQuery(raw, key, value string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.RequestURI()
}

func moderationFlash(status domain.CommentStatus) string {
	switch status {
	case domain.CommentApproved:
		return "Comment approved."
	case domain.CommentRejected:
		return "Comment rejected."
	case domain.CommentSpam:
		return "Comment marked as spam."
	case domain.CommentDeleted:
		return "Comment deleted."
	default:
		return "Comment updated."
	}
}
