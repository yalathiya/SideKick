package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics collectors (low-cardinality labels)
var (
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sidekick_requests_total",
			Help: "Total number of HTTP requests handled by sidekick",
		},
		[]string{"method", "route", "status"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "sidekick_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	rateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sidekick_ratelimit_hits_total",
			Help: "Total number of requests blocked by the rate limiter (429)",
		},
	)
)

func Register() {
	collectors := []prometheus.Collector{reqCounter, reqDuration, rateLimitHits}
	for _, c := range collectors {
		// ignore register error in dev to avoid panics on re-register
		_ = prometheus.Register(c)
	}
}

// Handler returns the prometheus HTTP handler to mount at /metrics
func Handler() http.Handler {
	return promhttp.Handler()
}

// routeLabelForRequest returns a stable, low-cardinality route label.
// It prefers the chi route pattern (e.g. "/users/{id}") when present,
// otherwise falls back to the actual URL path.
func routeLabelForRequest(r *http.Request) string {
	// Try to get chi route pattern (low-cardinality)
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	// fallback - safe but may have higher cardinality
	return r.URL.Path
}

// Middleware instruments requests: counts and measures duration.
// NOTE: we use route templates when available to avoid label explosion.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// capture status using a small recorder
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		method := r.Method
		route := routeLabelForRequest(r)
		status := strconv.Itoa(rec.status)

		reqCounter.WithLabelValues(method, route, status).Inc()
		reqDuration.WithLabelValues(method, route).Observe(duration)
	})
}

// IncRateLimitHit should be called by the rate limiter when it rejects a request
func IncRateLimitHit() {
	rateLimitHits.Inc()
}

// statusRecorder copies minimal behaviour to capture status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
