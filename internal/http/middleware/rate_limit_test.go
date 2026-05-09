package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRateLimiterSweepsExpiredClients(t *testing.T) {
	rl := NewRateLimiter(10, time.Millisecond)
	rl.clients["198.51.100.10"] = []time.Time{time.Now().Add(-time.Hour)}

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/comments", nil)
	req.RemoteAddr = "203.0.113.20:12345"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, ok := rl.clients["198.51.100.10"]; ok {
		t.Fatal("expected expired client bucket to be removed")
	}
	if _, ok := rl.clients["203.0.113.20"]; !ok {
		t.Fatal("expected active client to be keyed by host without port")
	}
}

func TestClientKeyStripsPort(t *testing.T) {
	if got := clientKey("203.0.113.20:12345"); got != "203.0.113.20" {
		t.Fatalf("expected host key, got %q", got)
	}
	if got := clientKey("203.0.113.20"); got != "203.0.113.20" {
		t.Fatalf("expected raw key fallback, got %q", got)
	}
}

func TestRateLimiterReturnsRetryAfter(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRequest(http.MethodPost, "/comments", nil)
	first.RemoteAddr = "203.0.113.20:12345"
	handler.ServeHTTP(httptest.NewRecorder(), first)

	second := httptest.NewRequest(http.MethodPost, "/comments", nil)
	second.RemoteAddr = "203.0.113.20:12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, second)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("expected Retry-After header")
	}
	if body := rec.Result().Body; body != nil {
		data, _ := io.ReadAll(body)
		if !strings.Contains(string(data), "retry after") {
			t.Fatalf("expected retry body, got %q", string(data))
		}
	}
}

func TestRateLimiterBoundsClientBuckets(t *testing.T) {
	rl := NewRateLimiter(10, time.Hour)
	rl.maxClients = 2
	rl.clients["198.51.100.1"] = []time.Time{time.Now().Add(-2 * time.Minute)}
	rl.clients["198.51.100.2"] = []time.Time{time.Now().Add(-time.Minute)}

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/comments", nil)
	req.RemoteAddr = "198.51.100.3:12345"
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if len(rl.clients) > rl.maxClients {
		t.Fatalf("expected at most %d clients, got %d", rl.maxClients, len(rl.clients))
	}
	if _, ok := rl.clients["198.51.100.1"]; ok {
		t.Fatal("expected oldest client bucket to be evicted")
	}
	if _, ok := rl.clients["198.51.100.3"]; !ok {
		t.Fatal("expected new client bucket to be recorded")
	}
}
