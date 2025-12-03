package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter holds buckets keyed by client identifier
type Limiter struct {
	buckets sync.Map // map[key]*Bucket
	cap     float64
	rateSec float64
}

// NewLimiter creates a limiter with `capacity` tokens per `per` duration.
func NewLimiter(capacity int, per time.Duration) *Limiter {
	return &Limiter{
		cap:     float64(capacity),
		rateSec: float64(capacity) / per.Seconds(),
	}
}

func clientKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Middleware enforces rate-limits before calling next handler
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientKey(r)
		v, _ := l.buckets.LoadOrStore(key, NewBucket(l.cap, l.rateSec))
		b := v.(*Bucket)

		allowed := b.Allow(1)

		// Simple rate-limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", l.cap))

		if !allowed {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
