package public

import (
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
	"deadcomments/internal/i18n"
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
		Locale        string  `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		locale := i18n.Normalize(r.URL.Query().Get("locale"), r.Header.Get("Accept-Language"))
		writeJSONError(w, i18n.Text(locale, "invalid_json"), http.StatusBadRequest)
		return
	}
	locale := i18n.Normalize(payload.Locale, r.Header.Get("Accept-Language"))
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
		writeJSONError(w, createErrorMessage(locale, err), statusForCreateError(err))
		return
	}
	status := statusForCreatedComment(comment.Status)
	message := createMessage(locale, comment.Status, reason)
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
		Locale       string `json:"locale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		locale := i18n.Normalize(r.URL.Query().Get("locale"), r.Header.Get("Accept-Language"))
		writeJSONError(w, i18n.Text(locale, "invalid_json"), http.StatusBadRequest)
		return
	}
	locale := i18n.Normalize(payload.Locale, r.Header.Get("Accept-Language"))
	site, err := h.sites.ByKey(r.Context(), siteKey)
	if err != nil || site == nil {
		writeJSONError(w, i18n.Text(locale, "comments_unavailable"), http.StatusNotFound)
		return
	}
	origin := h.trustedCommentOrigin(r, siteKey, pageKey, payload.ParentOrigin, payload.EmbedToken)
	if !h.sites.OriginAllowed(site, origin) {
		writeJSONError(w, i18n.Text(locale, "origin_not_allowed"), http.StatusForbidden)
		return
	}
	if len([]rune(payload.BodyMarkdown)) > site.MaxCommentLength {
		writeJSONError(w, i18n.Text(locale, "comment_too_long"), http.StatusBadRequest)
		return
	}
	bodyHTML, err := h.markdown.Render(payload.BodyMarkdown)
	if err != nil {
		writeJSONError(w, i18n.Text(locale, "preview_unavailable"), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"body_html": bodyHTML}, http.StatusOK)
}

func createErrorMessage(locale string, err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "author name"), strings.Contains(msg, "name is required"):
		return i18n.Text(locale, "required_name")
	case strings.Contains(msg, "body"), strings.Contains(msg, "comment is required"):
		return i18n.Text(locale, "required_body")
	case strings.Contains(msg, "too long"):
		return i18n.Text(locale, "comment_too_long")
	case strings.Contains(msg, "origin"):
		return i18n.Text(locale, "origin_not_allowed")
	case strings.Contains(msg, "parent"):
		return i18n.Text(locale, "parent_invalid")
	case strings.Contains(msg, "replies"):
		return i18n.Text(locale, "replies_disabled")
	case strings.Contains(msg, "page does not allow"):
		return i18n.Text(locale, "page_posting_closed")
	case strings.Contains(msg, "reserved"):
		return i18n.Text(locale, "reserved_name")
	case strings.Contains(msg, "rate limit"):
		return i18n.Text(locale, "rejected_rate_limit")
	case strings.Contains(msg, "banned"):
		return i18n.Text(locale, "rejected_ip_banned")
	case strings.Contains(msg, "not found"):
		return i18n.Text(locale, "comments_unavailable")
	default:
		return i18n.Text(locale, "comments_unavailable")
	}
}

func createMessage(locale string, status domain.CommentStatus, reason string) string {
	if status == domain.CommentApproved {
		return i18n.Text(locale, "comment_posted")
	}
	if status == domain.CommentPending {
		if reason == "word ban" {
			return i18n.Text(locale, "pending_rule_message")
		}
		return i18n.Text(locale, "pending_message")
	}
	if status == domain.CommentRejected {
		return rejectedMessage(locale, reason)
	}
	if status == domain.CommentSpam {
		return spamMessage(locale, reason)
	}
	return i18n.Text(locale, "not_posted")
}

func rejectedMessage(locale string, reason string) string {
	switch reason {
	case "ip banned":
		return i18n.Text(locale, "rejected_ip_banned")
	case "rate limit":
		return i18n.Text(locale, "rejected_rate_limit")
	case "word ban":
		return i18n.Text(locale, "rejected_word_ban")
	default:
		return i18n.Text(locale, "rejected_default")
	}
}

func spamMessage(locale string, reason string) string {
	switch reason {
	case "honeypot":
		return i18n.Text(locale, "spam_honeypot")
	case "duplicate body":
		return i18n.Text(locale, "spam_duplicate")
	case "too many links":
		return i18n.Text(locale, "spam_links")
	case "word ban":
		return i18n.Text(locale, "spam_word_ban")
	default:
		return i18n.Text(locale, "spam_default")
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
