package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string][]time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, clients: map[string][]time.Time{}}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		now := time.Now()
		rl.mu.Lock()
		var kept []time.Time
		for _, t := range rl.clients[ip] {
			if now.Sub(t) <= rl.window {
				kept = append(kept, t)
			}
		}
		if len(kept) >= rl.limit {
			rl.mu.Unlock()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		rl.clients[ip] = append(kept, now)
		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
