package middleware

import (
	"net/http"
	"net/http/httptest"
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
