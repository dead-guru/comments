package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	lastSweep time.Time
	clients   map[string][]time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, clients: map[string][]time.Time{}}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientKey(r.RemoteAddr)
		now := time.Now()
		rl.mu.Lock()
		rl.sweepLocked(now)
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

func (rl *RateLimiter) sweepLocked(now time.Time) {
	if !rl.lastSweep.IsZero() && now.Sub(rl.lastSweep) < rl.window {
		return
	}
	for ip, hits := range rl.clients {
		kept := hits[:0]
		for _, t := range hits {
			if now.Sub(t) <= rl.window {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(rl.clients, ip)
			continue
		}
		rl.clients[ip] = kept
	}
	rl.lastSweep = now
}

func clientKey(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}
	return remoteAddr
}
