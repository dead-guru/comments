package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"deadcomments/internal/domain"
	"deadcomments/internal/i18n"
)

type apiAnnotationList struct {
	Page struct {
		Key   string `json:"key"`
		State string `json:"state"`
	} `json:"page"`
	Annotations []*domain.PublicAnnotation `json:"annotations"`
}

func (h *Handlers) APIListAnnotations(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	pageKey := decodedParam(chi.URLParam(r, "page_key"))
	if !h.setSiteCORS(w, r, siteKey) {
		writeJSONError(w, "origin is not allowed for this site", http.StatusForbidden)
		return
	}
	page, annotations, err := h.annotations.PublicByPage(r.Context(), siteKey, pageKey)
	if err != nil || page == nil {
		writeJSONError(w, "annotations unavailable", http.StatusNotFound)
		return
	}
	var resp apiAnnotationList
	resp.Page.Key = page.PageKey
	resp.Page.State = string(page.State)
	resp.Annotations = toPublicAnnotations(annotations)
	writeJSON(w, resp, http.StatusOK)
}

func (h *Handlers) APICreateAnnotation(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	pageKey := decodedParam(chi.URLParam(r, "page_key"))
	if !h.setSiteCORS(w, r, siteKey) {
		writeJSONError(w, "origin is not allowed for this site", http.StatusForbidden)
		return
	}
	var payload struct {
		AuthorName      string  `json:"author_name"`
		AuthorEmail     string  `json:"author_email"`
		AuthorWebsite   string  `json:"author_website"`
		BodyMarkdown    string  `json:"body_markdown"`
		Honeypot        string  `json:"honeypot"`
		PageTitle       string  `json:"page_title"`
		PageURL         string  `json:"page_url"`
		Locale          string  `json:"locale"`
		Selector        string  `json:"selector"`
		SelectedText    string  `json:"selected_text"`
		SelectionPrefix string  `json:"selection_prefix"`
		SelectionSuffix string  `json:"selection_suffix"`
		TextStart       *int64  `json:"text_start"`
		TextEnd         *int64  `json:"text_end"`
		MetadataJSON    *string `json:"metadata_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		locale := i18n.Normalize(r.URL.Query().Get("locale"), r.Header.Get("Accept-Language"))
		writeJSONDecodeError(w, locale, err)
		return
	}
	locale := i18n.Normalize(payload.Locale, r.Header.Get("Accept-Language"))
	result, err := h.annotations.CreateDetailed(r.Context(), domain.AnnotationCreateInput{
		CommentCreateInput: domain.CommentCreateInput{
			SiteKey:       siteKey,
			PageKey:       pageKey,
			PageTitle:     payload.PageTitle,
			PageURL:       payload.PageURL,
			AuthorName:    payload.AuthorName,
			AuthorEmail:   payload.AuthorEmail,
			AuthorWebsite: payload.AuthorWebsite,
			BodyMarkdown:  payload.BodyMarkdown,
			Honeypot:      payload.Honeypot,
			Origin:        r.Header.Get("Origin"),
			Referer:       firstValue(r.Header.Get("Referer")),
			IP:            clientIP(r),
			UserAgent:     r.UserAgent(),
		},
		Selector:        payload.Selector,
		SelectedText:    payload.SelectedText,
		SelectionPrefix: payload.SelectionPrefix,
		SelectionSuffix: payload.SelectionSuffix,
		TextStart:       payload.TextStart,
		TextEnd:         payload.TextEnd,
		MetadataJSON:    payload.MetadataJSON,
	})
	if err != nil {
		writeJSONError(w, createAnnotationErrorMessage(locale, err), statusForAnnotationError(err))
		return
	}
	comment := result.CommentResult.Comment
	reason := result.CommentResult.Reason
	status := statusForCreatedComment(comment.Status)
	message := createMessageWithRetry(locale, comment.Status, reason, result.CommentResult.RetryAfter)
	response := map[string]any{
		"id":         result.Annotation.ID,
		"status":     comment.Status,
		"message":    message,
		"reason":     reason,
		"annotation": toPublicAnnotation(result.Annotation),
	}
	if result.CommentResult.RetryAfter > 0 {
		response["retry_after_seconds"] = secondsCeil(result.CommentResult.RetryAfter)
		response["rate_limit"] = map[string]any{
			"limit":          result.CommentResult.Limit,
			"window_seconds": secondsCeil(result.CommentResult.Window),
		}
		w.Header().Set("Retry-After", fmt.Sprintf("%d", secondsCeil(result.CommentResult.RetryAfter)))
	}
	if status >= http.StatusBadRequest {
		response["error"] = message
	}
	writeJSON(w, response, status)
}

func (h *Handlers) APIAnnotationsOptions(w http.ResponseWriter, r *http.Request) {
	siteKey := chi.URLParam(r, "site_key")
	if !h.setSiteCORS(w, r, siteKey) {
		http.Error(w, "origin is not allowed for this site", http.StatusForbidden)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) setSiteCORS(w http.ResponseWriter, r *http.Request, siteKey string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	site, err := h.sites.ByKey(r.Context(), siteKey)
	if err != nil || site == nil || !h.sites.OriginAllowed(site, origin) {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "600")
	w.Header().Add("Vary", "Origin")
	return true
}

func createAnnotationErrorMessage(locale string, err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "selected text"):
		return i18n.Text(locale, "annotation_selected_required")
	case strings.Contains(msg, "annotation selector"):
		return i18n.Text(locale, "annotation_anchor_invalid")
	case strings.Contains(msg, "annotation metadata"):
		return err.Error()
	default:
		return createErrorMessage(locale, err)
	}
}

func statusForAnnotationError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "selected text"), strings.Contains(msg, "annotation selector"), strings.Contains(msg, "annotation metadata"):
		return http.StatusBadRequest
	default:
		return statusForCreateError(err)
	}
}

func toPublicAnnotations(annotations []*domain.Annotation) []*domain.PublicAnnotation {
	out := make([]*domain.PublicAnnotation, 0, len(annotations))
	for _, annotation := range annotations {
		out = append(out, toPublicAnnotation(annotation))
	}
	return out
}

func toPublicAnnotation(annotation *domain.Annotation) *domain.PublicAnnotation {
	if annotation == nil {
		return nil
	}
	return &domain.PublicAnnotation{
		ID:              annotation.ID,
		Selector:        annotation.Selector,
		SelectedText:    annotation.SelectedText,
		SelectionPrefix: annotation.SelectionPrefix,
		SelectionSuffix: annotation.SelectionSuffix,
		TextStart:       annotation.TextStart,
		TextEnd:         annotation.TextEnd,
		TextHash:        annotation.TextHash,
		CreatedAt:       annotation.CreatedAt,
		Comment:         toPublicComment(annotation.Comment),
	}
}
