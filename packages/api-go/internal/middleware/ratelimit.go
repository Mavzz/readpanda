package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimit allows each client `perMinute` requests to `next` in any rolling
// minute, answering 429 past that. For public endpoints that would otherwise
// let anyone probe at speed — the username check, which reveals whether a
// name exists. Counts are per server instance: on Cloud Run with several
// instances the real ceiling is a multiple of this, which still stops a
// loop.
func RateLimit(perMinute int, next http.HandlerFunc) http.HandlerFunc {
	var mu sync.Mutex
	hits := map[string][]time.Time{}
	lastSweep := time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		cutoff := now.Add(-time.Minute)
		key := clientIP(r)

		mu.Lock()
		// Drop clients that went quiet, so the map can't grow without bound.
		if now.Sub(lastSweep) > time.Minute {
			for k, times := range hits {
				if len(times) == 0 || times[len(times)-1].Before(cutoff) {
					delete(hits, k)
				}
			}
			lastSweep = now
		}
		recent := hits[key][:0]
		for _, t := range hits[key] {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}
		allowed := len(recent) < perMinute
		if allowed {
			recent = append(recent, now)
		}
		hits[key] = recent
		mu.Unlock()

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Too many requests. Try again in a minute."}`))
			return
		}
		next(w, r)
	}
}

// clientIP is the caller's address. Behind Cloud Run, Google's front end
// appends the address it saw to X-Forwarded-For, so the last hop is the one
// to trust — earlier hops are whatever the client chose to send.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		hops := strings.Split(fwd, ",")
		return strings.TrimSpace(hops[len(hops)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
