package public

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
)

func (h *Handlers) signEmbedToken(siteKey, pageKey, parentOrigin string) string {
	parentOrigin = normalizeOrigin(parentOrigin)
	if h.embedSecret == "" || siteKey == "" || pageKey == "" || parentOrigin == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(h.embedSecret))
	_, _ = mac.Write([]byte(siteKey))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(pageKey))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(parentOrigin))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handlers) validEmbedToken(siteKey, pageKey, parentOrigin, token string) bool {
	expected := h.signEmbedToken(siteKey, pageKey, parentOrigin)
	if expected == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(token))
}

func (h *Handlers) trustedCommentOrigin(r *http.Request, siteKey, pageKey, parentOrigin, token string) string {
	origin := originFromRequest(r)
	if h.validEmbedToken(siteKey, pageKey, parentOrigin, token) {
		return normalizeOrigin(parentOrigin)
	}
	return origin
}

func originFromRequest(r *http.Request) string {
	if origin := normalizeOrigin(r.Header.Get("Origin")); origin != "" {
		return origin
	}
	return normalizeOrigin(r.Header.Get("Referer"))
}

func normalizeOrigin(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
