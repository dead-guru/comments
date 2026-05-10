package admin

import (
	"encoding/csv"
	"encoding/json"
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
	page, limit, offset := adminPage(r, "page")
	filter.Limit = limit
	filter.Offset = offset
	comments, err := h.comments.AdminListFiltered(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to load comments", http.StatusInternalServerError)
		return
	}
	comments, hasNext := trimAdminPage(comments)
	h.render(w, r, "admin/comments_queue.html", map[string]any{
		"Comments":   comments,
		"Status":     filter.Status,
		"Filters":    r.URL.Query(),
		"Pagination": newPagination(r, "page", page, hasNext),
	})
}

func (h *Handlers) ExportComments(w http.ResponseWriter, r *http.Request) {
	filter := commentListFilterFromRequest(r)
	filter.Limit = 5000
	comments, err := h.comments.AdminListFiltered(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to export comments", http.StatusInternalServerError)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="deadcomments-comments.csv"`)
		out := csv.NewWriter(w)
		_ = out.Write([]string{"id", "type", "annotation_id", "annotation_text", "site_key", "page_key", "page_title", "author", "status", "moderation_reason", "body_markdown", "created_at"})
		for _, c := range comments {
			reason := ""
			if c.ModerationReason != nil {
				reason = *c.ModerationReason
			}
			commentType := "comment"
			annotationID := ""
			annotationText := ""
			if c.Annotation != nil {
				commentType = "annotation"
				annotationID = c.Annotation.ID
				annotationText = c.Annotation.SelectedText
			}
			_ = out.Write([]string{c.ID, commentType, annotationID, annotationText, c.SiteKey, c.PageKey, c.PageTitle, c.AuthorDisplayName, string(c.Status), reason, c.BodyMarkdown, c.CreatedAt.Format("2006-01-02T15:04:05Z07:00")})
		}
		out.Flush()
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="deadcomments-comments.json"`)
	_ = json.NewEncoder(w).Encode(comments)
}

func (h *Handlers) PendingComments(w http.ResponseWriter, r *http.Request) {
	page, limit, offset := adminPage(r, "page")
	comments, err := h.comments.AdminListFiltered(r.Context(), repository.CommentListFilter{
		Status: string(domain.CommentPending),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, "failed to load pending comments", http.StatusInternalServerError)
		return
	}
	comments, hasNext := trimAdminPage(comments)
	h.render(w, r, "admin/comments_queue.html", map[string]any{
		"Comments":   comments,
		"Status":     domain.CommentPending,
		"Filters":    r.URL.Query(),
		"Pagination": newPagination(r, "page", page, hasNext),
	})
}

func (h *Handlers) CommentDetail(w http.ResponseWriter, r *http.Request) {
	comment, err := h.comments.ByID(r.Context(), chiID(r))
	if err != nil || comment == nil {
		http.NotFound(w, r)
		return
	}
	events, err := h.moderation.EventsForComment(r.Context(), comment.ID)
	if err != nil {
		http.Error(w, "failed to load moderation history", http.StatusInternalServerError)
		return
	}
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
	total := 0
	failed := 0
	for _, id := range r.Form["comment_id"] {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		total++
		if err := h.comments.SetStatus(r.Context(), id, status); err != nil {
			failed++
		}
	}
	redirectAdminFlash(w, r, "/admin/comments", bulkModerationFlash(status, total, failed))
}

func commentListFilterFromRequest(r *http.Request) repository.CommentListFilter {
	q := r.URL.Query()
	filter := repository.CommentListFilter{
		Status:        strings.TrimSpace(q.Get("status")),
		Search:        strings.TrimSpace(q.Get("q")),
		IPHash:        strings.TrimSpace(q.Get("ip_hash")),
		UserAgentHash: strings.TrimSpace(q.Get("ua_hash")),
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
	if err := h.comments.Edit(r.Context(), id, r.FormValue("body_markdown")); err != nil {
		http.Error(w, "failed to edit comment", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/comments/"+id, http.StatusFound)
}

func (h *Handlers) BanIP(w http.ResponseWriter, r *http.Request) {
	admin := middleware.AdminFromContext(r.Context())
	var adminID *int64
	if admin != nil {
		adminID = &admin.ID
	}
	if err := h.comments.BanIPAndSpam(r.Context(), chiID(r), adminID, r.FormValue("reason")); err != nil {
		http.Error(w, "failed to ban IP", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/comments/"+chiID(r), http.StatusFound)
}

func (h *Handlers) setCommentStatus(w http.ResponseWriter, r *http.Request, status domain.CommentStatus) {
	id := chiID(r)
	if err := h.comments.SetStatus(r.Context(), id, status); err != nil {
		http.Error(w, "failed to update comment", http.StatusInternalServerError)
		return
	}
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
	if raw := strings.TrimSpace(r.FormValue("redirect_to")); raw != "" {
		u, err := url.Parse(raw)
		if err == nil && !u.IsAbs() && u.Host == "" && (u.Path == "/admin" || strings.HasPrefix(u.Path, "/admin/")) {
			q := u.Query()
			if flash != "" {
				q.Set("flash", flash)
			}
			http.Redirect(w, r, (&url.URL{Path: u.Path, RawQuery: q.Encode()}).RequestURI(), http.StatusFound)
			return
		}
	}
	http.Redirect(w, r, adminRedirectLocation(fallback, flash), http.StatusFound)
}

func adminRedirectLocation(raw string, flash string) string {
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || (u.Path != "/admin" && !strings.HasPrefix(u.Path, "/admin/")) {
		u = &url.URL{Path: "/admin"}
	}
	q := u.Query()
	if flash != "" {
		q.Set("flash", flash)
	}
	return (&url.URL{Path: u.Path, RawQuery: q.Encode()}).RequestURI()
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

func bulkModerationFlash(status domain.CommentStatus, total int, failed int) string {
	if total == 0 {
		return "No comments selected."
	}
	if failed == 0 {
		return moderationFlash(status)
	}
	if failed == total {
		return "No comments could be updated."
	}
	return "Some comments could not be updated."
}
