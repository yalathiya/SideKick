package logging

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder allows us to capture the response status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware logs method, path, status, time, client IP
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// wrap ResponseWriter to capture status
		rec := &statusRecorder{
			ResponseWriter: w,
			status:         200, // default
		}

		next.ServeHTTP(rec, r)

		duration := time.Since(start)

		// client IP
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		log.Printf(
			"%d %s %s  %v  client=%s",
			rec.status,
			r.Method,
			r.URL.Path,
			duration,
			ip,
		)
	})
}
