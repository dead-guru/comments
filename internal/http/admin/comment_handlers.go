package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
	"deadcomments/internal/http/middleware"
)

func (h *Handlers) Comments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	comments, _ := h.comments.AdminList(r.Context(), status, r.URL.Query().Get("q"), nil, nil, 200)
	h.render(w, r, "admin/comments_queue.html", map[string]any{"Comments": comments, "Status": status})
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
	http.Redirect(w, r, "/admin/comments/"+id, http.StatusFound)
}

func chiID(r *http.Request) string {
	return chi.URLParam(r, "id")
}
