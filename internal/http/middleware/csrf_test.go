package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFTokenSetsSecureCookieAndReusesValidToken(t *testing.T) {
	csrf := NewCSRF("csrf-secret", true)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	token := csrf.Token(rec, req)
	if token == "" {
		t.Fatal("expected token")
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one csrf cookie, got %d", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != CSRFCookieName {
		t.Fatalf("expected csrf cookie, got %s", cookie.Name)
	}
	if cookie.Value != token {
		t.Fatal("expected cookie value to match token")
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/admin" {
		t.Fatalf("unexpected csrf cookie attributes: %#v", cookie)
	}

	reuseReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reuseReq.AddCookie(cookie)
	reuseRec := httptest.NewRecorder()
	if got := csrf.Token(reuseRec, reuseReq); got != token {
		t.Fatal("expected valid csrf cookie to be reused")
	}
	if got := len(reuseRec.Result().Cookies()); got != 0 {
		t.Fatalf("expected no replacement cookie, got %d", got)
	}
}

func TestCSRFMiddlewareAllowsSafeMethods(t *testing.T) {
	csrf := NewCSRF("csrf-secret", false)
	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/sites", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected safe method through, got %d", rec.Code)
	}
}

func TestCSRFMiddlewareAcceptsValidDoubleSubmitToken(t *testing.T) {
	csrf := NewCSRF("csrf-secret", false)
	token := issueCSRFToken(t, csrf)
	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := newCSRFPost(token)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected valid csrf token through, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCSRFMiddlewareRejectsMissingMismatchedAndTamperedTokens(t *testing.T) {
	csrf := NewCSRF("csrf-secret", false)
	validToken := issueCSRFToken(t, csrf)
	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	tests := []struct {
		name   string
		cookie string
		form   string
	}{
		{name: "missing cookie", form: validToken},
		{name: "missing form", cookie: validToken},
		{name: "mismatched form", cookie: validToken, form: issueCSRFToken(t, csrf)},
		{name: "tampered signature", cookie: "payload.invalid-signature", form: "payload.invalid-signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newCSRFPost(tt.form)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected forbidden, got %d", rec.Code)
			}
		})
	}
}

func issueCSRFToken(t *testing.T, csrf *CSRF) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	return csrf.Token(rec, req)
}

func newCSRFPost(token string) *http.Request {
	form := url.Values{}
	if token != "" {
		form.Set("csrf_token", token)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/sites", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}
