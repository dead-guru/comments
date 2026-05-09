package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

const CSRFCookieName = "dc_csrf"

type CSRF struct {
	secret string
	secure bool
}

func NewCSRF(secret string, secure bool) *CSRF {
	return &CSRF{secret: secret, secure: secure}
}

func (c *CSRF) Token(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(CSRFCookieName); err == nil && c.valid(cookie.Value) {
		return cookie.Value
	}
	raw := make([]byte, 24)
	_, _ = rand.Read(raw)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	token := payload + "." + c.sign(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   c.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400,
	})
	return token
}

func (c *CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		cookie, cerr := r.Cookie(CSRFCookieName)
		token := r.FormValue("csrf_token")
		if cerr != nil || token == "" || cookie.Value != token || !c.valid(token) {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *CSRF) sign(payload string) string {
	h := hmac.New(sha256.New, []byte(c.secret))
	_, _ = h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (c *CSRF) valid(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(c.sign(parts[0])))
}
