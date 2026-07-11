// Package ratelimit provides a small in-memory, per-client token-bucket rate
// limiter with an HTTP middleware. It has no external dependencies so it can be
// dropped in front of sensitive routes (e.g. auth) without pulling in a new
// module. Note it limits each process instance independently; a shared,
// cross-instance layer (Cloud Armor / Redis) is the eventual production-grade
// control — see docs/production-readiness.md.
package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/adotomov/fashion-store/apps/api/internal/shared/httpx"
)

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

// Limiter is a per-key token-bucket rate limiter safe for concurrent use.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens refilled per second
	burst   float64 // maximum tokens, and the initial allowance for a new key
}

// New returns a Limiter that allows an immediate burst of `burst` requests and
// refills at `ratePerSec` tokens per second. It starts a background janitor
// that evicts idle keys so the map cannot grow unbounded.
func New(ratePerSec, burst float64) *Limiter {
	l := &Limiter{
		buckets: make(map[string]*bucket),
		rate:    ratePerSec,
		burst:   burst,
	}
	go l.cleanupLoop()
	return l
}

// allow reports whether a request from key may proceed, consuming a token.
func (l *Limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		l.buckets[key] = &bucket{tokens: l.burst - 1, lastSeen: now}
		return true
	}

	b.tokens += now.Sub(b.lastSeen).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.lastSeen = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for key, b := range l.buckets {
			if time.Since(b.lastSeen) > 10*time.Minute {
				delete(l.buckets, key)
			}
		}
		l.mu.Unlock()
	}
}

// Middleware rejects requests exceeding the limit with 429 Too Many Requests,
// keyed by client IP.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, please slow down")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP resolves the originating client IP, honouring the first entry of
// X-Forwarded-For (set by Cloud Run / load balancers) and falling back to the
// transport remote address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
