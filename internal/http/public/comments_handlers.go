package public

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
	"deadcomments/internal/service"
)

type apiCommentList struct {
	Page struct {
		Key   string `json:"key"`
		State string `json:"state"`
	} `json:"page"`
	Sort     domain.CommentSort      `json:"sort"`
	Comments []*domain.PublicComment `json:"comments"`
}

func (h *Handlers) APIListComments(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	pageKey := decodedParam(chi.URLParam(r, "page_key"))
	sort := service.NormalizeCommentSort(r.URL.Query().Get("sort"))
	page, comments, err := h.comments.PublicTree(r.Context(), siteKey, pageKey, sort)
	if err != nil || page == nil {
		writeJSONError(w, "comments unavailable", http.StatusNotFound)
		return
	}
	var resp apiCommentList
	resp.Page.Key = page.PageKey
	resp.Page.State = string(page.State)
	resp.Sort = sort
	resp.Comments = toPublicComments(comments)
	writeJSON(w, resp, http.StatusOK)
}

func (h *Handlers) APICreateComment(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	pageKey := decodedParam(chi.URLParam(r, "page_key"))
	var payload struct {
		AuthorName    string  `json:"author_name"`
		AuthorEmail   string  `json:"author_email"`
		AuthorWebsite string  `json:"author_website"`
		BodyMarkdown  string  `json:"body_markdown"`
		ParentID      *string `json:"parent_id"`
		Honeypot      string  `json:"honeypot"`
		PageTitle     string  `json:"page_title"`
		PageURL       string  `json:"page_url"`
		ParentOrigin  string  `json:"parent_origin"`
		EmbedToken    string  `json:"embed_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, "invalid json", http.StatusBadRequest)
		return
	}
	origin := h.trustedCommentOrigin(r, siteKey, pageKey, payload.ParentOrigin, payload.EmbedToken)
	referer := firstValue(r.Header.Get("Referer"))
	comment, reason, err := h.comments.Create(r.Context(), domain.CommentCreateInput{
		SiteKey:       siteKey,
		PageKey:       pageKey,
		PageTitle:     payload.PageTitle,
		PageURL:       payload.PageURL,
		AuthorName:    payload.AuthorName,
		AuthorEmail:   payload.AuthorEmail,
		AuthorWebsite: payload.AuthorWebsite,
		BodyMarkdown:  payload.BodyMarkdown,
		ParentID:      payload.ParentID,
		Honeypot:      payload.Honeypot,
		Origin:        origin,
		Referer:       referer,
		IP:            clientIP(r),
		UserAgent:     r.UserAgent(),
	})
	if err != nil {
		writeJSONError(w, err.Error(), statusForCreateError(err))
		return
	}
	status := statusForCreatedComment(comment.Status)
	message := createMessage(comment.Status, reason)
	response := map[string]any{
		"id":      comment.ID,
		"status":  comment.Status,
		"message": message,
		"reason":  reason,
		"comment": toPublicComment(comment),
	}
	if status >= http.StatusBadRequest {
		response["error"] = message
	}
	writeJSON(w, response, status)
}

func (h *Handlers) APIPreviewComment(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	pageKey := decodedParam(chi.URLParam(r, "page_key"))
	var payload struct {
		BodyMarkdown string `json:"body_markdown"`
		ParentOrigin string `json:"parent_origin"`
		EmbedToken   string `json:"embed_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, "invalid json", http.StatusBadRequest)
		return
	}
	site, err := h.sites.ByKey(r.Context(), siteKey)
	if err != nil || site == nil {
		writeJSONError(w, "site not found", http.StatusNotFound)
		return
	}
	origin := h.trustedCommentOrigin(r, siteKey, pageKey, payload.ParentOrigin, payload.EmbedToken)
	if !h.sites.OriginAllowed(site, origin) {
		writeJSONError(w, "origin is not allowed for this site", http.StatusForbidden)
		return
	}
	if len([]rune(payload.BodyMarkdown)) > site.MaxCommentLength {
		writeJSONError(w, "comment is too long", http.StatusBadRequest)
		return
	}
	bodyHTML, err := h.markdown.Render(payload.BodyMarkdown)
	if err != nil {
		writeJSONError(w, "preview unavailable", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"body_html": bodyHTML}, http.StatusOK)
}

func createMessage(status domain.CommentStatus, reason string) string {
	if status == domain.CommentApproved {
		return "Comment posted."
	}
	if status == domain.CommentPending {
		if reason == "word ban" {
			return "Comment submitted and waiting for moderation because it matched a moderation rule."
		}
		return "Comment submitted and waiting for moderation."
	}
	if status == domain.CommentRejected {
		return rejectedMessage(reason)
	}
	if status == domain.CommentSpam {
		return spamMessage(reason)
	}
	return "Comment was not posted."
}

func rejectedMessage(reason string) string {
	switch reason {
	case "ip banned":
		return "Comment rejected: this network is blocked for this site."
	case "rate limit":
		return "Comment rejected: too many comments were submitted recently. Please try again later."
	case "word ban":
		return "Comment rejected by this site's moderation rules."
	default:
		return "Comment rejected by moderation."
	}
}

func spamMessage(reason string) string {
	switch reason {
	case "honeypot":
		return "Comment rejected by spam protection."
	case "duplicate body":
		return "Comment rejected by spam protection: duplicate comment."
	case "too many links":
		return "Comment rejected by spam protection: too many links."
	case "word ban":
		return "Comment rejected by this site's moderation rules."
	default:
		return "Comment rejected by spam protection."
	}
}

func statusForCreatedComment(status domain.CommentStatus) int {
	switch status {
	case domain.CommentApproved:
		return http.StatusCreated
	case domain.CommentPending:
		return http.StatusAccepted
	case domain.CommentRejected, domain.CommentSpam:
		return http.StatusForbidden
	default:
		return http.StatusAccepted
	}
}

func statusForCreateError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "not found"):
		return http.StatusNotFound
	case strings.Contains(msg, "origin"), strings.Contains(msg, "rate limit"), strings.Contains(msg, "banned"):
		return http.StatusForbidden
	case strings.Contains(msg, "required"), strings.Contains(msg, "too long"), strings.Contains(msg, "parent"), strings.Contains(msg, "replies"), strings.Contains(msg, "page does not allow"), strings.Contains(msg, "reserved"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func toPublicComments(comments []*domain.Comment) []*domain.PublicComment {
	out := make([]*domain.PublicComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, toPublicComment(c))
	}
	return out
}

func toPublicComment(c *domain.Comment) *domain.PublicComment {
	if c == nil {
		return nil
	}
	return &domain.PublicComment{
		ID:                c.ID,
		ParentID:          c.ParentID,
		AuthorName:        c.AuthorName,
		AuthorDisplayName: c.AuthorDisplayName,
		AuthorWebsite:     c.AuthorWebsite,
		AuthorAvatarHash:  c.AuthorAvatarHash,
		TripcodePublic:    c.TripcodePublic,
		TripcodeKind:      c.TripcodeKind,
		BadgeType:         c.BadgeType,
		BadgeLabel:        c.BadgeLabel,
		BodyHTML:          c.BodyHTML,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt,
		EditedAt:          c.EditedAt,
		ReplyingToAuthor:  c.ReplyingToAuthor,
		Children:          toPublicComments(c.Children),
	}
}

func writeJSON(w http.ResponseWriter, v any, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, msg string, status int) {
	writeJSON(w, map[string]string{"error": msg}, status)
}

func firstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decodedParam(raw string) string {
	v, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return v
}
